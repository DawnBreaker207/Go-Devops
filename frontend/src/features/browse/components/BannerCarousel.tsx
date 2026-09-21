import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { Movie } from '@/types';
import { useSoonToast } from '@/hooks/useSoonToast';
import SoonToast from '@/components/ui/SoonToast';

interface BannerCarouselProps {
  movies: Movie[];
  /** Opens BookingModal for the active movie; same action as PosterCard "Book" (see HomePage), no separate route. */
  onBook: (movie: Movie) => void;
}

const AUTO_ADVANCE_MS = 5000;
/** Max 5 slides. */
const MAX_SLIDES = 5;

const TicketIcon = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path
      d="M3 8a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v2a1.5 1.5 0 0 0 0 3v2a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-2a1.5 1.5 0 0 0 0-3V8Z"
      stroke="currentColor"
      strokeWidth="1.7"
      strokeLinejoin="round"
    />
    <path d="M10 6.5v11" stroke="currentColor" strokeWidth="1.7" strokeDasharray="2.2 2.2" />
  </svg>
);

const PlayIcon = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <circle cx="12" cy="12" r="9" stroke="currentColor" strokeWidth="1.7" />
    <path d="M10 8.5v7l6-3.5-6-3.5Z" fill="currentColor" />
  </svg>
);

const PauseIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
    <rect x="6" y="5" width="4" height="14" rx="1" />
    <rect x="14" y="5" width="4" height="14" rx="1" />
  </svg>
);

const PlayGlyphIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
    <path d="M7 5v14l12-7L7 5Z" />
  </svg>
);

const ChevronIcon = ({ direction }: { direction: 'left' | 'right' }) => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path
      d={direction === 'left' ? 'M15 5l-7 7 7 7' : 'M9 5l7 7-7 7'}
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

/** Home hero carousel. No banner endpoint/field exists, so the first 5 "showing" movies stand in; hidden when empty, never an empty frame. Always-dark card (fixed bg-black), never --cp-ink-rgb. Same max-w container as the page below, not full-bleed. "Watch trailer" is a stub toast (trailer_url exists but video UI is out of scope). Extras: taller (420/500/560px), arrow buttons, crossfade slides (stacked absolute layers; only active is opacity-100 + interactive). */
export const BannerCarousel = ({ movies, onBook }: BannerCarouselProps) => {
  const { t } = useTranslation();
  const [index, setIndex] = useState(0);
  const [autoPlay, setAutoPlay] = useState(true);
  const { toastMessage, showSoon } = useSoonToast();
  const slides = movies.slice(0, MAX_SLIDES);

  useEffect(() => {
    if (!autoPlay || slides.length < 2) return;
    const timer = window.setInterval(() => {
      setIndex((i) => (i + 1) % slides.length);
    }, AUTO_ADVANCE_MS);
    return () => window.clearInterval(timer);
  }, [autoPlay, slides.length]);

  if (slides.length === 0) return null;

  // Clamp stale index after list changes; avoids an empty frame.
  const activeIndex = index < slides.length ? index : 0;

  const goPrev = () => setIndex((activeIndex - 1 + slides.length) % slides.length);
  const goNext = () => setIndex((activeIndex + 1) % slides.length);

  return (
    <div className="mb-6 overflow-hidden rounded-xl border border-white/15 bg-black shadow-2xl">
      <div className="relative h-105 w-full sm:h-125 md:h-140">
        {slides.map((movie, i) => {
          const isActive = i === activeIndex;
          return (
            <div
              key={movie.id}
              aria-hidden={!isActive}
              className={`absolute inset-0 transition-opacity duration-700 ease-in-out ${
                isActive ? 'z-10 opacity-100' : 'pointer-events-none z-0 opacity-0'
              }`}
            >
              {movie.poster_url ? (
                <img
                  className="absolute inset-0 h-full w-full object-cover"
                  src={movie.poster_url}
                  alt=""
                  loading={i === 0 ? 'eager' : 'lazy'}
                />
              ) : null}
              <div
                className="pointer-events-none absolute inset-0 bg-linear-to-t from-black via-black/75 to-black/15"
                aria-hidden="true"
              />

              <div className="absolute inset-0 flex flex-col justify-end p-5 sm:p-8 md:p-10">
                <div className="max-w-2xl">
                  <h1 className="mb-2.5 text-xl leading-tight font-extrabold text-white sm:text-3xl md:text-4xl">
                    {movie.title}
                  </h1>
                  {movie.description ? (
                    <p className="mb-5 line-clamp-2 max-w-xl text-[13px] text-white/90 sm:text-sm md:text-base">
                      {movie.description}
                    </p>
                  ) : null}

                  <div className="flex flex-wrap items-center gap-2.5">
                    <button
                      type="button"
                      onClick={() => onBook(movie)}
                      className="flex min-h-11 cursor-pointer items-center gap-2 rounded-lg border-none bg-brand px-5 text-[13px] font-semibold text-on-brand transition-colors duration-fast ease-out hover-fine:bg-brand-hover sm:text-sm"
                    >
                      <TicketIcon />
                      {t('customer.bannerBookNow')}
                    </button>
                    <button
                      type="button"
                      onClick={showSoon}
                      className="flex min-h-11 cursor-pointer items-center gap-2 rounded-lg border border-white/40 bg-transparent px-5 text-[13px] font-semibold text-white transition-colors duration-fast ease-out hover-fine:border-brand hover-fine:text-brand sm:text-sm"
                    >
                      <PlayIcon />
                      {t('customer.bannerWatchTrailer')}
                    </button>
                    <button
                      type="button"
                      onClick={() => setAutoPlay((value) => !value)}
                      aria-label={t(
                        autoPlay ? 'customer.bannerAutoplayOn' : 'customer.bannerAutoplayOff'
                      )}
                      aria-pressed={autoPlay}
                      className="flex min-h-11 min-w-11 cursor-pointer items-center justify-center rounded-lg border border-white/40 bg-transparent text-white transition-colors duration-fast ease-out hover-fine:border-brand hover-fine:text-brand"
                    >
                      {autoPlay ? <PauseIcon /> : <PlayGlyphIcon />}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          );
        })}

        {slides.length > 1 ? (
          <>
            <button
              type="button"
              onClick={goPrev}
              aria-label={t('customer.bannerPrev')}
              className="absolute top-1/2 left-3 z-20 flex h-10 w-10 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full border-none bg-black/40 text-white transition-colors duration-fast ease-out hover-fine:bg-black/65 sm:h-11 sm:w-11"
            >
              <ChevronIcon direction="left" />
            </button>
            <button
              type="button"
              onClick={goNext}
              aria-label={t('customer.bannerNext')}
              className="absolute top-1/2 right-3 z-20 flex h-10 w-10 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full border-none bg-black/40 text-white transition-colors duration-fast ease-out hover-fine:bg-black/65 sm:h-11 sm:w-11"
            >
              <ChevronIcon direction="right" />
            </button>

            <div className="absolute inset-x-0 bottom-3.5 z-20 flex justify-center gap-1.5">
              {slides.map((movie, i) => (
                <button
                  key={movie.id}
                  type="button"
                  className={`h-1 cursor-pointer rounded-full border-none p-0 transition-[width,background-color] duration-fast ease-out ${
                    i === activeIndex ? 'w-6 bg-brand' : 'w-2 bg-white/50'
                  }`}
                  aria-label={t('customer.bannerGoTo', { title: movie.title })}
                  aria-current={i === activeIndex}
                  onClick={() => setIndex(i)}
                />
              ))}
            </div>
          </>
        ) : null}
      </div>

      <SoonToast message={toastMessage} />
    </div>
  );
};

export default BannerCarousel;
