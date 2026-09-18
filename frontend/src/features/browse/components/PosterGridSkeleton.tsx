import './PosterCard.css';

interface PosterGridSkeletonProps {
  count?: number;
}

/** Khung xuong dung ty le 2:3 nhu the that, nen doi sang noi dung khong nhay layout. */
export const PosterGridSkeleton = ({ count = 8 }: PosterGridSkeletonProps) => (
  <div className="cp-poster-grid" aria-hidden="true">
    {Array.from({ length: count }, (_, i) => (
      <div key={i} className="cp-poster__skeleton" />
    ))}
  </div>
);

export default PosterGridSkeleton;
