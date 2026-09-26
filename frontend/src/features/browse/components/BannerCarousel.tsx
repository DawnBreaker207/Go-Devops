import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { Movie } from '@/types';
import { formatDuration } from '@/utils/format';
import { movieBackdrop, movieYear } from '../movieImage';
import { usePrefersReducedMotion } from '@/hooks/usePrefersReducedMotion';
import { FOCUS_RING, HOME_SECTION } from '@/theme/customerTw';

interface BannerCarouselProps {
  movies: Movie[];
  /** Opens BookingModal for the active movie; same action as the card "Book" (see HomePage), no separate route. */
  onBook: (movie: Movie) => void;
}

const AUTO_ADVANCE_MS = 5000;
/** Max 5 slides. */
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

/** Home hero, rebuilt to QVisionShow frame 1-101: a FULL-BLEED backdrop with the title, a
 *  `genre | genre` line, a year/duration meta row, three lines of synopsis and one pill call to
 *  action. The route carries `handle.fullBleed` so this can reach the viewport edges; every section
 *  below therefore supplies its own `HOME_SECTION` container.
 *
 *  No banner endpoint exists, so the first 5 `showing` movies stand in; hidden entirely when empty,
 *  never an empty frame. Always-dark (fixed `bg-black`/white text), never `--cp-ink-rgb`, because a
 *  photographic backdrop is dark in both of the customer zone's themes.
 *
 *  The image is `backdrop_url` with `poster_url` as the fallback — before migration 000011 there was
 *  no landscape field at all and this hero stretched a 2:3 portrait across the full width.
 *
 *  Kept from the previous version and NOT in the frame: the arrows, the dots, and the autoplay
 *  pause/play toggle. The toggle is the viewer's only control over an animation that moves by
 *  itself, so it stays regardless of what the design shows. The old "Watch trailer" button is gone:
 *  it was a stub toast, and there is now a real Trailers section further down the page. */
export const BannerCarousel = ({ movies, onBook }: BannerCarouselProps) => {
  const { t } = useTranslation();
  const [index, setIndex] = useState(0);
  const [autoPlay, setAutoPlay] = useState(true);
  // motion.md: "no autoplay" under reduced motion. The toggle still works, so someone who wants the
  // carousel moving can still ask for it - this only decides what happens without being asked.
  const reducedMotion = usePrefersReducedMotion();
  const autoAdvancing = autoPlay && !reducedMotion;
  const slides = movies.slice(0, MAX_SLIDES);

  useEffect(() => {
    if (!autoAdvancing || slides.length < 2) return;
    const timer = window.setInterval(() => {
      // Clamp before advancing: a raw stale index (left over from a longer list) would skip a slide
      // on the first tick, landing somewhere other than the one after the visible slide.
      setIndex((i) => ((i < slides.length ? i : 0) + 1) % slides.length);
    }, AUTO_ADVANCE_MS);
    return () => window.clearInterval(timer);
  }, [autoAdvancing, slides.length]);

  if (slides.length === 0) return null;

  // Clamp stale index after list changes; avoids an empty frame.
  const activeIndex = index < slides.length ? index : 0;

  const goPrev = () => setIndex((activeIndex - 1 + slides.length) % slides.length);
  const goNext = () => setIndex((activeIndex + 1) % slides.length);

  return (
    // `z-0` is load-bearing: it gives the hero its own stacking context, so the slide layers and the
    // dots (z-10/z-20 INSIDE it) can never paint over the sticky header, which is also z-20 but comes
    // earlier in the DOM and would otherwise lose the tie.
    <div className="relative z-0 h-140 w-full overflow-hidden bg-black sm:h-160 md:h-180 lg:h-195">
      {slides.map((movie, i) => {
        const isActive = i === activeIndex;
        const image = movieBackdrop(movie);
        const year = movieYear(movie);
        // Genre is one free-text field ("Action, Adventure"); the frame separates them with pipes.
        const genres = movie.genre
          .split(',')
          .map((part) => part.trim())
          .filter(Boolean)
          .join(' | ');

        return (
          <div
            key={movie.id}
            aria-hidden={!isActive}
            // duration-moderate, not a raw 700ms: motion.md forbids hardcoded numbers, and 700
            // also escaped the reduced-motion clamp, which only rewrites the tokens.
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
            {/* Two scrims, as the frame has: one up from the bottom for the page seam, one in from
                the left so the copy stays legible over a bright backdrop. */}
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

                  {/* tabIndex -1 while hidden: the slide is aria-hidden, and a focusable element
                      inside an aria-hidden container is an accessibility violation - the keyboard
                      lands on a button no screen reader can announce. */}
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
