import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { Movie } from '@/types';
import { movieBackdrop } from '../movieImage';
import { extractIframeSrc, youtubeEmbedUrl } from '../trailerEmbed';
import PlayGlyph from './PlayGlyph';
import {
  FOCUS_RING,
  HOME_HEADING,
  INK_BG_06,
  INK_BORDER_10,
  TRANSITION_FAST,
} from '@/theme/customerTw';

interface FilmTrailerProps {
  movie: Movie;
}

/** This film's own trailer on its detail page - one video, no thumbnail picker (that's the home
 *  page's TrailersSection, which showcases many movies at once). Renders nothing without a trailer_url. */
export const FilmTrailer = ({ movie }: FilmTrailerProps) => {
  const { t } = useTranslation();
  const [playing, setPlaying] = useState(false);

  if (!movie.trailer_url) return null;

  const embed = youtubeEmbedUrl(movie.trailer_url);
  const rawUrl = extractIframeSrc(movie.trailer_url.trim());
  const image = movieBackdrop(movie);

  return (
    <section className="mt-2 mb-7" aria-labelledby="film-trailer">
      <h2 id="film-trailer" className={`mb-3 ${HOME_HEADING}`}>
        {t('customer.trailers')}
      </h2>

      <div
        className={`relative aspect-video w-full overflow-hidden rounded-2xl border ${INK_BORDER_10} ${INK_BG_06}`}
      >
        {playing && embed ? (
          <iframe
            className="absolute inset-0 h-full w-full border-none"
            src={embed}
            title={t('customer.trailerOf', { title: movie.title })}
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
            allowFullScreen
          />
        ) : (
          <>
            {image ? (
              <img src={image} alt="" loading="lazy" className="h-full w-full object-cover" />
            ) : null}
            <span className="pointer-events-none absolute inset-0 bg-black/35" aria-hidden="true" />
            {embed ? (
              <button
                type="button"
                onClick={() => setPlaying(true)}
                aria-label={t('customer.trailerPlay', { title: movie.title })}
                className={`absolute inset-0 flex cursor-pointer items-center justify-center border-none bg-transparent p-0 text-white ${TRANSITION_FAST} hover-fine:text-brand ${FOCUS_RING}`}
              >
                <PlayGlyph size={64} />
              </button>
            ) : (
              <a
                href={rawUrl}
                target="_blank"
                rel="noreferrer"
                aria-label={t('customer.trailerPlay', { title: movie.title })}
                className={`absolute inset-0 flex items-center justify-center no-underline ${FOCUS_RING}`}
              >
                <span className={`text-white ${TRANSITION_FAST} hover-fine:text-brand`}>
                  <PlayGlyph size={64} />
                </span>
              </a>
            )}
          </>
        )}
      </div>
    </section>
  );
};

export default FilmTrailer;
