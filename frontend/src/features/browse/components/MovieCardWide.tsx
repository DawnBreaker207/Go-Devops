import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import type { Movie } from '@/types';
import { filmPath } from '@/routes/paths';
import { formatDuration } from '@/utils/format';
import Button from '@/components/ui/Button';
import LinkButton from '@/components/ui/LinkButton';
import { movieBackdrop, movieYear } from '../movieImage';
import {
  CARD_TITLE,
  FOCUS_RING,
  INK_35,
  INK_60,
  INK_BG_05,
  INK_BG_06,
  INK_BORDER_10,
} from '@/theme/customerTw';

interface MovieCardWideProps {
  movie: Movie;
  /** Passed only for `showing` movies; opens BookingModal in place. Absent means no "Book" button. */
  onBook?: (movie: Movie) => void;
  /** First row loads eagerly so the grid does not pop in under the fold-line. */
  eager?: boolean;
}

/** The home grid card from QVisionShow frame 1-101: a 3:2 landscape image, the title over two lines,
 *  a "year · genre · duration" meta line, then a brand pill and the age-rating badge.
 *
 *  Two deliberate departures from the frame:
 *  - The frame puts a `★ 4.5` beside the button. The backend has NO rating field, so that number
 *    would be invented. The age rating goes there instead — real data, and it already owns a tested
 *    colour ramp in `--cp-rating-*`.
 *  - This is a NEW component rather than an edit of `PosterCard`, which is shared with `/films` and
 *    stays portrait 2:3. Editing that one would have redesigned a page nobody asked about. */
export const MovieCardWide = ({ movie, onBook, eager }: MovieCardWideProps) => {
  const { t } = useTranslation();
  const image = movieBackdrop(movie);
  const meta = [movieYear(movie), movie.genre, formatDuration(movie.duration)]
    .filter(Boolean)
    .join(' · ');

  return (
    <article
      className={`flex h-full flex-col gap-3 rounded-2xl border p-3 ${INK_BORDER_10} ${INK_BG_05}`}
    >
      <Link
        to={filmPath(movie.id)}
        aria-label={movie.title}
        className={`group relative block aspect-3/2 overflow-hidden rounded-xl ${INK_BG_06} ${FOCUS_RING}`}
      >
        {image ? (
          <img
            src={image}
            alt=""
            loading={eager ? 'eager' : 'lazy'}
            className="h-full w-full object-cover transition-transform duration-moderate ease-out [@media(hover:hover)_and_(pointer:fine)]:group-hover:scale-(--motion-scale-zoom)"
          />
        ) : (
          <span
            className={`absolute inset-0 flex items-center justify-center text-[40px] font-bold ${INK_35}`}
            aria-hidden="true"
          >
            {movie.title.charAt(0).toUpperCase()}
          </span>
        )}
      </Link>

      <Link to={filmPath(movie.id)} className={`group no-underline ${FOCUS_RING}`}>
        <span className={`line-clamp-2 text-[15px] leading-snug font-bold ${CARD_TITLE}`}>
          {movie.title}
        </span>
      </Link>

      <p className={`truncate text-[13px] ${INK_60}`}>{meta || t('customer.genreUnknown')}</p>

      <div className="mt-auto flex items-center justify-between gap-2 pt-1">
        {movie.status === 'showing' && onBook ? (
          <Button variant="primary" pill onClick={() => onBook(movie)}>
            {t('customer.bookNow')}
          </Button>
        ) : (
          <LinkButton to={filmPath(movie.id)} variant="ghost" pill>
            {t('customer.viewDetail')}
          </LinkButton>
        )}

        {movie.age_rating ? (
          <span
            className="flex-none rounded-full px-2.5 py-1 text-[11px] font-bold"
            style={{
              background: `var(--cp-rating-${movie.age_rating.toLowerCase()})`,
              color: `var(--cp-rating-${movie.age_rating.toLowerCase()}-text)`,
            }}
          >
            {movie.age_rating}
          </span>
        ) : null}
      </div>
    </article>
  );
};

export default MovieCardWide;
