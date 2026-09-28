interface PlayGlyphProps {
  size: number;
}

export const PlayGlyph = ({ size }: PlayGlyphProps) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <circle cx="12" cy="12" r="11" stroke="currentColor" strokeWidth="1.5" />
    <path d="M10 8.2v7.6l6.2-3.8L10 8.2Z" fill="currentColor" />
  </svg>
);

export default PlayGlyph;
