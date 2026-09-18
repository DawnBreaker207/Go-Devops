import { Link } from 'react-router-dom';
import type { Movie } from '@/types';
import { filmPath } from '@/routes/paths';
import { formatDuration } from '@/utils/format';
import './PosterCard.css';

interface PosterCardProps {
  movie: Movie;
}

/**
 * Mot poster trong luoi trang chu. CSS thuan, khong dung antd - khu khach la
 * mot he thi giac rieng (xem .claude/context/decisions.md #15).
 */
export const PosterCard = ({ movie }: PosterCardProps) => (
  <Link to={filmPath(movie.id)} className="cp-poster" aria-label={movie.title}>
    <div className="cp-poster__media">
      {movie.poster_url ? (
        <img src={movie.poster_url} alt="" loading="lazy" />
      ) : (
        <span className="cp-poster__fallback" aria-hidden="true">
          {movie.title.charAt(0).toUpperCase()}
        </span>
      )}
      {movie.age_rating ? <span className="cp-poster__badge">{movie.age_rating}</span> : null}
    </div>
    <span className="cp-poster__title">{movie.title}</span>
    <span className="cp-poster__meta">
      {[movie.genre, formatDuration(movie.duration)].filter(Boolean).join(' · ')}
    </span>
  </Link>
);

export default PosterCard;
