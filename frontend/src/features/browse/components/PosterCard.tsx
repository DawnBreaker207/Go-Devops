import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import type { Movie } from '@/types';
import { filmPath } from '@/routes/paths';
import LinkButton from '@/components/ui/LinkButton';
import Button from '@/components/ui/Button';
import { useFavoriteMovies } from '../hooks/useFavoriteMovies';
import {
  FOCUS_RING,
  INK_35,
  INK_60,
  INK_BG_035,
  INK_BG_06,
  INK_BORDER_14,
  TRANSITION_FAST,
} from '@/theme/customerTw';

interface PosterCardProps {
  movie: Movie;
  /** Passed only for `showing` movies; opens BookingModal in place. Absent means no "Book" button. */
  onBook?: (movie: Movie) => void;
}

const HeartIcon = ({ filled }: { filled: boolean }) => (
  <svg
    width="15"
    height="15"
    viewBox="0 0 24 24"
    fill={filled ? 'currentColor' : 'none'}
    aria-hidden="true"
  >
    <path
      d="M12 21s-7.5-4.87-10.2-9.6C.36 8.1 1.86 4.5 5.4 3.9c2.1-.36 4.02.6 5.1 2.4a5.7 5.7 0 0 1 1.5-1.62c1.5-1.08 3.6-1.14 5.1-.78 3.54.6 5.04 4.2 3.6 7.5C20.5 16.13 12 21 12 21Z"
      stroke="currentColor"
      strokeWidth="1.7"
      strokeLinejoin="round"
    />
  </svg>
);

/** One movie card in the home//films grid. Portrait 2:3 kept (data has only `poster_url`, no landscape backdrop; letterboxing it would crop most of the art). `coming_soon` shows "Details" instead of a dead "Book" button. Image and title are two SEPARATE <Link>s: one link wrapping the favorite button would nest interactives. */
export const PosterCard = ({ movie, onBook }: PosterCardProps) => {
  const { t } = useTranslation();
  const { isFavorite, toggle } = useFavoriteMovies();
  const favorite = isFavorite(movie.id);

  return (
    <article
      className={`flex h-full flex-col overflow-hidden rounded-xl border ${INK_BORDER_14} ${INK_BG_035}`}
    >
      <Link
        to={filmPath(movie.id)}
        aria-label={movie.title}
        className={`group relative block aspect-2/3 overflow-hidden ${INK_BG_06}`}
      >
        {movie.poster_url ? (
          <img
            src={movie.poster_url}
            alt=""
            loading="lazy"
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
        {movie.age_rating ? (
          <span
            className="absolute top-2 left-2 rounded-full px-2 py-0.5 text-[11px] font-bold"
            style={{
              background: `var(--cp-rating-${movie.age_rating.toLowerCase()})`,
              color: `var(--cp-rating-${movie.age_rating.toLowerCase()}-text)`,
            }}
          >
            {movie.age_rating}
          </span>
        ) : null}
      </Link>

      <div className="flex flex-1 flex-col gap-2 p-3">
        <div className="flex items-start justify-between gap-2">
          <Link
            to={filmPath(movie.id)}
            className={`truncate text-[15px] font-semibold no-underline ${TRANSITION_FAST} hover-fine:text-brand`}
          >
            {movie.title}
          </Link>
          <button
            type="button"
            aria-label={t(favorite ? 'customer.favoriteRemove' : 'customer.favoriteAdd')}
            aria-pressed={favorite}
            onClick={() => toggle(movie.id)}
            className={`flex h-8 w-8 flex-none items-center justify-center rounded-full border-none bg-transparent p-0 ${
              favorite ? 'text-brand' : INK_60
            } ${TRANSITION_FAST} hover-fine:text-brand ${FOCUS_RING}`}
          >
            <HeartIcon filled={favorite} />
          </button>
        </div>

        <p className={`truncate text-xs ${INK_60}`}>{movie.genre || t('customer.genreUnknown')}</p>

        <div className="mt-auto pt-1">
          {movie.status === 'showing' && onBook ? (
            <Button variant="primary" block onClick={() => onBook(movie)}>
              {t('customer.bookNow')}
            </Button>
          ) : movie.status === 'coming_soon' ? (
            <LinkButton to={filmPath(movie.id)} variant="ghost" block>
              {t('customer.viewDetail')}
            </LinkButton>
          ) : null}
        </div>
      </div>
    </article>
  );
};

export default PosterCard;
