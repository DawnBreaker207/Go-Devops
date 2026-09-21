export type SeatShape = 'single' | 'vip' | 'couple';

/** Shape = seat kind (single/vip/couple); text color = status. Couples use a 2:1 frame for 2 columns (`col_span=2`). */
export interface SeatIconProps {
  /** Seat kind to draw (default 'single'). */
  shape?: SeatShape;
  /** HELD seats: dashed outline instead of solid. */
  outline?: boolean;
  className?: string;
}

/**
 * Front-view tub seat icon, rounded. 3 variants: narrow single, vip, long couple.
 */
export const SeatIcon = ({ shape = 'single', outline = false, className }: SeatIconProps) => (
  <svg
    viewBox={shape === 'couple' ? '0 0 1024 512' : '0 0 512 512'}
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    aria-hidden="true"
    className={className}
  >
    {shape === 'couple' ? (
      <>
        <path
          fill={outline ? 'none' : 'currentColor'}
          stroke={outline ? 'currentColor' : 'none'}
          strokeWidth={outline ? 28 : 0}
          strokeDasharray={outline ? '40 24' : undefined}
          d="M70 335 L70 180 C70 120 220 110 512 110 C804 110 954 120 954 180 L954 335 Q 954 355 934 358 Q 512 390 90 358 Q 70 355 70 335 Z"
        />
        <path
          fill={outline ? 'none' : 'currentColor'}
          stroke={outline ? 'currentColor' : 'none'}
          strokeWidth={outline ? 28 : 0}
          strokeDasharray={outline ? '40 24' : undefined}
          d="M70 385 Q 512 425 954 385 C 954 385 954 435 934 455 Q 512 495 90 455 C 70 435 70 385 70 385 Z"
        />
      </>
    ) : shape === 'vip' ? (
      <>
        <path
          fill={outline ? 'none' : 'currentColor'}
          stroke={outline ? 'currentColor' : 'none'}
          strokeWidth={outline ? 28 : 0}
          strokeDasharray={outline ? '40 24' : undefined}
          d="M70 335 L70 180 C70 120 130 110 256 110 C382 110 442 120 442 180 L442 335 Q 442 355 422 358 Q 256 390 90 358 Q 70 355 70 335 Z"
        />
        <path
          fill={outline ? 'none' : 'currentColor'}
          stroke={outline ? 'currentColor' : 'none'}
          strokeWidth={outline ? 28 : 0}
          strokeDasharray={outline ? '40 24' : undefined}
          d="M70 385 Q 256 425 442 385 C 442 385 442 435 422 455 Q 256 495 90 455 C 70 435 70 385 70 385 Z"
        />
      </>
    ) : (
      <>
        <path
          fill={outline ? 'none' : 'currentColor'}
          stroke={outline ? 'currentColor' : 'none'}
          strokeWidth={outline ? 28 : 0}
          strokeDasharray={outline ? '40 24' : undefined}
          d="M92 335 L92 180 C92 120 145 110 256 110 C367 110 420 120 420 180 L420 335 Q 420 355 402 358 Q 256 390 110 358 Q 92 355 92 335 Z"
        />
        <path
          fill={outline ? 'none' : 'currentColor'}
          stroke={outline ? 'currentColor' : 'none'}
          strokeWidth={outline ? 28 : 0}
          strokeDasharray={outline ? '40 24' : undefined}
          d="M92 385 Q 256 425 420 385 C 420 385 420 435 402 455 Q 256 495 110 455 C 92 435 92 385 92 385 Z"
        />
      </>
    )}
  </svg>
);

export default SeatIcon;
