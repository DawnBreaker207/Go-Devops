import { INK_BG_07 } from '@/theme/customerTw';

interface PosterGridSkeletonProps {
  count?: number;
}

/** 2:3 skeleton frames matching real cards, so swap-in doesn't shift layout. */
export const PosterGridSkeleton = ({ count = 8 }: PosterGridSkeletonProps) => (
  <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-4" aria-hidden="true">
    {Array.from({ length: count }, (_, i) => (
      <div
        key={i}
        className={`aspect-2/3 animate-pulse rounded-xl ${INK_BG_07} motion-reduce:animate-none`}
      />
    ))}
  </div>
);

export default PosterGridSkeleton;
