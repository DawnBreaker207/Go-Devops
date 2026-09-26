import { HOME_GRID, INK_BG_06, INK_BORDER_10 } from '@/theme/customerTw';

interface WideGridSkeletonProps {
  count?: number;
}

/** Loading placeholder for the home's wide grid.
 *
 *  Every box mirrors `MovieCardWide`'s real one - the same wrapper classes, the same 3:2 image, two
 *  lines of title, one meta line and a 40px pill - so the swap to real cards shifts nothing.
 *  `PosterGridSkeleton` cannot be reused: it hardcodes its own grid and `aspect-2/3`, which would
 *  make every card change shape the moment the data lands.
 *
 *  `motion-reduce:animate-none` on each pulse, as `PosterGridSkeleton` already does: an
 *  indefinitely looping animation is exactly what a reduced-motion preference is asking to stop. */
export const WideGridSkeleton = ({ count = 8 }: WideGridSkeletonProps) => (
  <div className={HOME_GRID} aria-hidden="true">
    {Array.from({ length: count }, (_, i) => (
      <div key={i} className={`flex flex-col gap-3 rounded-2xl border p-3 ${INK_BORDER_10}`}>
        <div
          className={`aspect-3/2 animate-pulse rounded-xl ${INK_BG_06} motion-reduce:animate-none`}
        />
        {/* Two lines, because the card's title is `line-clamp-2`. */}
        <div className={`h-10 animate-pulse rounded ${INK_BG_06} motion-reduce:animate-none`} />
        <div
          className={`h-4.5 w-3/5 animate-pulse rounded ${INK_BG_06} motion-reduce:animate-none`}
        />
        <div className="mt-auto flex items-center justify-between gap-2 pt-1">
          <div
            className={`h-10 w-28 animate-pulse rounded-full ${INK_BG_06} motion-reduce:animate-none`}
          />
          <div
            className={`h-6 w-8 animate-pulse rounded-full ${INK_BG_06} motion-reduce:animate-none`}
          />
        </div>
      </div>
    ))}
  </div>
);

export default WideGridSkeleton;
