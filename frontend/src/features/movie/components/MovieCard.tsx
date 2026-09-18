import { Button, Skeleton, Tag, theme as antdTheme } from 'antd';
import { PlayCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { Movie, MovieStatus } from '@/types';
import { formatDuration } from '@/utils/format';
import './MovieCard.css';

const STATUS_COLOR: Record<MovieStatus, string> = {
  draft: 'default',
  showing: 'green',
  ended: 'red',
};

interface MovieCardProps {
  movie: Movie;
  onBook?: (movie: Movie) => void;
  onTrailer?: (movie: Movie) => void;
}

export const MovieCard = ({ movie, onBook, onTrailer }: MovieCardProps) => {
  const { t } = useTranslation();
  const { token } = antdTheme.useToken();

  // Bien cua card lay tu token antd de dark mode dung mau, con moi gia tri
  // chuyen dong deu den tu var(--motion-*) trong MovieCard.css.
  const cssVars = {
    '--movie-card-bg': token.colorBgContainer,
    '--movie-card-radius': `${token.borderRadiusLG}px`,
    '--movie-card-focus': token.colorPrimary,
    '--movie-card-placeholder': token.colorFillSecondary,
  } as React.CSSProperties;

  return (
    <article className="movie-card" style={cssVars} tabIndex={0} aria-label={movie.title}>
      <div className="movie-card__media">
        {movie.poster_url ? <img src={movie.poster_url} alt="" loading="lazy" /> : null}
        <div className="movie-card__overlay">
          <Button type="primary" size="small" onClick={() => onBook?.(movie)}>
            {t('movie.book')}
          </Button>
          {movie.trailer_url ? (
            <Button
              size="small"
              ghost
              icon={<PlayCircleOutlined />}
              onClick={() => onTrailer?.(movie)}
            >
              {t('movie.trailer')}
            </Button>
          ) : null}
        </div>
      </div>

      <div className="movie-card__body">
        <div className="movie-card__title">{movie.title}</div>
        <div className="movie-card__meta">
          <Tag color={STATUS_COLOR[movie.status]} bordered={false}>
            {t(`movie.status${movie.status.charAt(0).toUpperCase()}${movie.status.slice(1)}`)}
          </Tag>
          {formatDuration(movie.duration)}
        </div>
      </div>
    </article>
  );
};

/** Skeleton co dung khung cua card that nen khong gay layout shift khi load xong. */
export const MovieCardSkeleton = () => {
  const { token } = antdTheme.useToken();
  return (
    <article
      className="movie-card"
      style={
        {
          '--movie-card-bg': token.colorBgContainer,
          '--movie-card-radius': `${token.borderRadiusLG}px`,
          cursor: 'default',
        } as React.CSSProperties
      }
      aria-hidden
    >
      <div className="movie-card__media">
        <Skeleton.Node active style={{ width: '100%', height: '100%' }} />
      </div>
      <div className="movie-card__body">
        <div className="movie-card__title">
          <Skeleton.Input active size="small" block />
        </div>
        <div className="movie-card__meta">
          <Skeleton.Input active size="small" style={{ width: 120 }} />
        </div>
      </div>
    </article>
  );
};

export default MovieCard;
