import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { Movie } from '@/types';
import { formatDuration } from '@/utils/format';
import { movieBackdrop, movieYear } from '../movieImage';
import { usePrefersReducedMotion } from '@/hooks/usePrefersReducedMotion';
import { FOCUS_RING, HOME_SECTION } from '@/theme/customerTw';

interface BannerCarouselProps {
  movies: Movie[];
  // Opens BookingModal for the visible film; no dedicated route.
  onBook: (movie: Movie) => void;
}

const AUTO_ADVANCE_MS = 5000;
const MAX_SLIDES = 5;

const CalendarIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <rect x="3" y="5" width="18" height="16" rx="2" stroke="currentColor" strokeWidth="1.7" />
    <path d="M3 10h18M8 3v4M16 3v4" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
  </svg>
);

const ClockIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <circle cx="12" cy="12" r="9" stroke="currentColor" strokeWidth="1.7" />
    <path d="M12 7v5.2l3.2 2" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
  </svg>
);

const ArrowIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path
      d="M5 12h13m0 0-5-5m5 5-5 5"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
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

// Hero is always dark; with no banner endpoint, take 5 showing films, backdrop falling back to poster.
export const BannerCarousel = ({ movies, onBook }: BannerCarouselProps) => {
  const { t } = useTranslation();
  const [index, setIndex] = useState(0);
  const [autoPlay, setAutoPlay] = useState(true);
  // Reduced motion disables autoplay by default; the toggle can still re-enable it.
  const reducedMotion = usePrefersReducedMotion();
  const autoAdvancing = autoPlay && !reducedMotion;
  const slides = movies.slice(0, MAX_SLIDES);

  useEffect(() => {
    if (!autoAdvancing || slides.length < 2) return;
    const timer = window.setInterval(() => {
      // Clamp the old index before advancing so a shortened list never jumps to a wrong slide.
      setIndex((i) => ((i < slides.length ? i : 0) + 1) % slides.length);
    }, AUTO_ADVANCE_MS);
    return () => window.clearInterval(timer);
  }, [autoAdvancing, slides.length]);

  if (slides.length === 0) return null;

  // Clamp the old index when the list changes to avoid an empty frame.
  const activeIndex = index < slides.length ? index : 0;

  const goPrev = () => setIndex((activeIndex - 1 + slides.length) % slides.length);
  const goNext = () => setIndex((activeIndex + 1) % slides.length);

  return (
    <div className="relative z-0 h-140 w-full overflow-hidden bg-black sm:h-160 md:h-180 lg:h-195">
      {slides.map((movie, i) => {
        const isActive = i === activeIndex;
        const image = movieBackdrop(movie);
        const year = movieYear(movie);
        const genres = movie.genre
          .split(',')
          .map((part) => part.trim())
          .filter(Boolean)
          .join(' | ');

        return (
          <div
            key={movie.id}
            aria-hidden={!isActive}
            className={`absolute inset-0 transition-opacity duration-moderate ease-out ${
              isActive ? 'z-10 opacity-100' : 'pointer-events-none z-0 opacity-0'
            }`}
          >
            {image ? (
              <img
                className="absolute inset-0 h-full w-full object-cover"
                src={image}
                alt=""
                loading={i === 0 ? 'eager' : 'lazy'}
              />
            ) : null}
            <div
              className="pointer-events-none absolute inset-0 bg-linear-to-t from-black via-black/55 to-black/10"
              aria-hidden="true"
            />
            <div
              className="pointer-events-none absolute inset-0 bg-linear-to-r from-black/85 via-black/35 to-transparent"
              aria-hidden="true"
            />

            <div className="absolute inset-0 flex items-end pb-16 sm:pb-20 md:items-center md:pb-0">
              <div className={HOME_SECTION}>
                <div className="max-w-xl">
                  <h1 className="mb-3 text-3xl leading-[1.05] font-extrabold text-white sm:text-5xl md:text-6xl">
                    {movie.title}
                  </h1>

                  {genres ? (
                    <p className="mb-3 text-sm text-white/85 sm:text-base">{genres}</p>
                  ) : null}

                  <div className="mb-4 flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-white/80">
                    {year ? (
                      <span className="inline-flex items-center gap-2">
                        <CalendarIcon />
                        {year}
                      </span>
                    ) : null}
                    {movie.duration ? (
                      <span className="inline-flex items-center gap-2">
                        <ClockIcon />
                        {formatDuration(movie.duration)}
                      </span>
                    ) : null}
                  </div>

                  {movie.description ? (
                    <p className="mb-7 line-clamp-3 max-w-lg text-[13px] leading-relaxed text-white/75 sm:text-sm">
                      {movie.description}
                    </p>
                  ) : null}

                  {/* an trinh doc man hinh bat nut an: tabIndex -1 khi slide an. */}
                  <button
                    type="button"
                    onClick={() => onBook(movie)}
                    tabIndex={isActive ? undefined : -1}
                    className={`inline-flex min-h-12 cursor-pointer items-center gap-3 rounded-full border-none bg-brand px-7 text-sm font-bold text-on-brand transition-colors duration-fast ease-out hover-fine:bg-brand-hover ${FOCUS_RING}`}
                  >
                    {t('customer.bannerBookNow')}
                    <ArrowIcon />
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
            className={`absolute top-1/2 left-3 z-20 flex h-10 w-10 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full border-none bg-black/40 text-white transition-colors duration-fast ease-out hover-fine:bg-black/65 sm:h-11 sm:w-11 ${FOCUS_RING}`}
          >
            <ChevronIcon direction="left" />
          </button>
          <button
            type="button"
            onClick={goNext}
            aria-label={t('customer.bannerNext')}
            className={`absolute top-1/2 right-3 z-20 flex h-10 w-10 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full border-none bg-black/40 text-white transition-colors duration-fast ease-out hover-fine:bg-black/65 sm:h-11 sm:w-11 ${FOCUS_RING}`}
          >
            <ChevronIcon direction="right" />
          </button>

          <div className="absolute inset-x-0 bottom-5 z-20 flex items-center justify-center gap-1.5">
            {slides.map((movie, i) => (
              <button
                key={movie.id}
                type="button"
                className={`h-1 cursor-pointer rounded-full border-none p-0 transition-[width,background-color] duration-fast ease-out ${FOCUS_RING} ${
                  i === activeIndex ? 'w-6 bg-brand' : 'w-2 bg-white/50'
                }`}
                aria-label={t('customer.bannerGoTo', { title: movie.title })}
                aria-current={i === activeIndex}
                onClick={() => setIndex(i)}
              />
            ))}
            <button
              type="button"
              onClick={() => setAutoPlay((value) => !value)}
              aria-label={t(
                autoAdvancing ? 'customer.bannerAutoplayOn' : 'customer.bannerAutoplayOff'
              )}
              aria-pressed={autoAdvancing}
              className={`ml-3 flex h-8 w-8 cursor-pointer items-center justify-center rounded-full border-none bg-black/40 text-white transition-colors duration-fast ease-out hover-fine:bg-black/65 ${FOCUS_RING}`}
            >
              {autoAdvancing ? <PauseIcon /> : <PlayGlyphIcon />}
            </button>
          </div>
        </>
      ) : null}
    </div>
  );
};

export default BannerCarousel;
