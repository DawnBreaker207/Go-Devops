import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { Movie } from '@/types';
import { movieBackdrop } from '../movieImage';
import { extractIframeSrc, youtubeEmbedUrl } from '../trailerEmbed';
import PlayGlyph from './PlayGlyph';
import {
  FOCUS_RING,
  HOME_HEADING,
  HOME_SECTION,
  INK_60,
  INK_BG_06,
  INK_BORDER_10,
  TRANSITION_FAST,
} from '@/theme/customerTw';

interface TrailersSectionProps {
  movies: Movie[];
}

const MAX_TRAILERS = 5;

// Each click reloads YouTube: site trailers open a new tab, missing trailers hide.
export const TrailersSection = ({ movies }: TrailersSectionProps) => {
  const { t } = useTranslation();
  const withTrailer = movies.filter((movie) => movie.trailer_url).slice(0, MAX_TRAILERS);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [playing, setPlaying] = useState(false);

  if (withTrailer.length === 0) return null;

  // Clamp the old id on list change, like the carousel.
  const active = withTrailer.find((movie) => movie.id === activeId) ?? withTrailer[0];
  const embed = youtubeEmbedUrl(active.trailer_url);
  // A snippet iframe is not a valid href, so unwrap the URL first.
  const rawUrl = extractIframeSrc((active.trailer_url ?? '').trim());
  const image = movieBackdrop(active);

  const select = (movie: Movie) => {
    setActiveId(movie.id);
    setPlaying(false);
  };

  return (
    <section className={`${HOME_SECTION} pt-14 pb-4`} aria-labelledby="home-trailers">
      <h2 id="home-trailers" className={`mb-6 ${HOME_HEADING}`}>
        {t('customer.trailers')}
      </h2>

      <div
        className={`relative aspect-video w-full overflow-hidden rounded-2xl border ${INK_BORDER_10} ${INK_BG_06}`}
      >
        {playing && embed ? (
          <iframe
            className="absolute inset-0 h-full w-full border-none"
            src={embed}
            title={t('customer.trailerOf', { title: active.title })}
            // Non-allowlisted hosts break the whole chain in Chrome, so gate on the token.
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
                aria-label={t('customer.trailerPlay', { title: active.title })}
                className={`absolute inset-0 flex cursor-pointer items-center justify-center border-none bg-transparent p-0 text-white ${TRANSITION_FAST} hover-fine:text-brand ${FOCUS_RING}`}
              >
                <PlayGlyph size={72} />
              </button>
            ) : (
              <a
                href={rawUrl}
                target="_blank"
                rel="noreferrer"
                aria-label={t('customer.trailerPlay', { title: active.title })}
                className={`absolute inset-0 flex items-center justify-center no-underline ${FOCUS_RING}`}
              >
                {/* Colour on the child: unlayered `.cp-customer a` would otherwise paint this brand. */}
                <span className={`text-white ${TRANSITION_FAST} hover-fine:text-brand`}>
                  <PlayGlyph size={72} />
                </span>
              </a>
            )}
          </>
        )}
      </div>

      <p className={`mt-3 text-sm ${INK_60}`}>{active.title}</p>

      {withTrailer.length > 1 ? (
        <ul className="m-0 mt-5 flex list-none flex-wrap justify-center gap-3 p-0 sm:justify-start">
          {withTrailer.map((movie) => {
            const isActive = movie.id === active.id;
            return (
              <li key={movie.id}>
                <button
                  type="button"
                  onClick={() => select(movie)}
                  aria-current={isActive}
                  aria-label={t('customer.trailerOf', { title: movie.title })}
                  className={`relative h-20 w-32 cursor-pointer overflow-hidden rounded-xl border-2 bg-transparent p-0 ${TRANSITION_FAST} ${FOCUS_RING} ${
                    isActive ? 'border-brand' : INK_BORDER_10
                  }`}
                >
                  {movieBackdrop(movie) ? (
                    <img
                      src={movieBackdrop(movie)}
                      alt=""
                      loading="lazy"
                      className="h-full w-full object-cover"
                    />
                  ) : null}
                  <span
                    className="absolute inset-0 flex items-center justify-center bg-black/30 text-white"
                    aria-hidden="true"
                  >
                    <PlayGlyph size={26} />
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      ) : null}
    </section>
  );
};

export default TrailersSection;
