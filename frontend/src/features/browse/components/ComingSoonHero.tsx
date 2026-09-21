import { useTranslation } from 'react-i18next';
import { formatDate, formatVND, toCinemaTime } from '@/utils/format';
import { errorMessage } from '@/utils/error';
import type { ShowtimeListItem } from '@/types';
import Notice from '@/components/ui/Notice';
import { INK_55, INK_60 } from '@/theme/customerTw';
import {
  showtimeCardClass,
  showtimeHallClass,
  showtimePriceClass,
  showtimeTimeClass,
} from './showtimeStyles';

interface ComingSoonHeroProps {
  releaseDate: string;
  showtimes: ShowtimeListItem[] | undefined;
  isFetchingShowtimes: boolean;
  showtimesError: unknown;
  onSelectSeat: (show: ShowtimeListItem) => void;
}

/** `coming_soon` variant of the picker block: expected release date only, no day strip (no stable schedule yet). Early `open` shows already created still render exactly like normal FilmPage shows (shared showtimeStyles + onSelectSeat). No "remind me" button: out of scope. */
export const ComingSoonHero = ({
  releaseDate,
  showtimes,
  isFetchingShowtimes,
  showtimesError,
  onSelectSeat,
}: ComingSoonHeroProps) => {
  const { t } = useTranslation();
  const earlyShowtimes = showtimes ?? [];

  return (
    <div>
      <span className="mb-4 inline-block rounded-full bg-brand-soft px-3.5 py-1 text-xs font-bold tracking-[0.4px] text-brand uppercase">
        {t('customer.comingSoon')}
      </span>

      <div className="mb-2 text-[15px]">
        <span className={`mb-0.5 block text-xs ${INK_55}`}>
          {t('customer.comingSoonReleaseDate')}
        </span>
        {formatDate(releaseDate)}
      </div>

      {showtimesError ? (
        <Notice variant="error">{errorMessage(showtimesError, t('common.somethingWrong'))}</Notice>
      ) : null}

      {isFetchingShowtimes ? <p className={`text-sm ${INK_60}`}>{t('common.loading')}</p> : null}

      {!isFetchingShowtimes && earlyShowtimes.length > 0 ? (
        <>
          <h2 className="mt-6 mb-4 text-lg font-bold">{t('customer.comingSoonEarlyShowtimes')}</h2>
          <div className="grid grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-3">
            {earlyShowtimes.map((show) => (
              <button
                key={show.id}
                type="button"
                className={showtimeCardClass}
                onClick={() => onSelectSeat(show)}
              >
                <span className={showtimeTimeClass}>
                  {toCinemaTime(show.start_at).format('DD/MM HH:mm')}
                </span>
                <span className={showtimeHallClass}>{show.hall_name}</span>
                {show.from_price ? (
                  <span className={showtimePriceClass}>
                    {t('customer.fromPrice', { price: formatVND(show.from_price) })}
                  </span>
                ) : null}
              </button>
            ))}
          </div>
        </>
      ) : null}
    </div>
  );
};

export default ComingSoonHero;
