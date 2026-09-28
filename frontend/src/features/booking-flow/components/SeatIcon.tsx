export type SeatShape = 'single' | 'vip' | 'couple';

export interface SeatIconProps {
  shape?: SeatShape;
  /** HELD seats: dashed outline instead of solid. */
  outline?: boolean;
  className?: string;
}

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
          d="M4 292 L4 120 C4 53 176 42 512 42 C848 42 1020 53 1020 120 L1020 292 Q 1020 314 997 318 Q 512 353 27 318 Q 4 314 4 292 Z"
        />
        <path
          fill={outline ? 'none' : 'currentColor'}
          stroke={outline ? 'currentColor' : 'none'}
          strokeWidth={outline ? 28 : 0}
          strokeDasharray={outline ? '40 24' : undefined}
          d="M4 348 Q 512 392 1020 348 C 1020 348 1020 403 997 426 Q 512 470 27 426 C 4 403 4 348 4 348 Z"
        />
      </>
    ) : shape === 'vip' ? (
      <>
        <path
          fill={outline ? 'none' : 'currentColor'}
          stroke={outline ? 'currentColor' : 'none'}
          strokeWidth={outline ? 28 : 0}
          strokeDasharray={outline ? '40 24' : undefined}
          d="M40 208 L40 83 C40 29 110 20 256 20 C402 20 472 29 472 83 L472 208 Q 472 232 449 254 Q 256 293 63 254 Q 40 232 40 208 Z"
        />
        <path
          fill={outline ? 'none' : 'currentColor'}
          stroke={outline ? 'currentColor' : 'none'}
          strokeWidth={outline ? 28 : 0}
          strokeDasharray={outline ? '40 24' : undefined}
          d="M40 287 Q 256 304 472 287 C 472 287 472 308 449 316 Q 256 333 63 316 C 40 308 40 313 40 287 Z"
        />
        <path
          fill={outline ? 'none' : 'currentColor'}
          stroke={outline ? 'currentColor' : 'none'}
          strokeWidth={outline ? 28 : 0}
          strokeDasharray={outline ? '40 24' : undefined}
          d="M40 338 Q 256 355 472 338 C 472 338 472 416 449 424 Q 256 441 63 424 C 40 416 40 364 40 338 Z"
        />
      </>
    ) : (
      <>
        <path
          fill={outline ? 'none' : 'currentColor'}
          stroke={outline ? 'currentColor' : 'none'}
          strokeWidth={outline ? 28 : 0}
          strokeDasharray={outline ? '40 24' : undefined}
          d="M28 296 L28 106 C28 32 102 20 256 20 C410 20 484 32 484 106 L484 296 Q 484 320 460 324 Q 256 363 52 324 Q 28 320 28 296 Z"
        />
        <path
          fill={outline ? 'none' : 'currentColor'}
          stroke={outline ? 'currentColor' : 'none'}
          strokeWidth={outline ? 28 : 0}
          strokeDasharray={outline ? '40 24' : undefined}
          d="M28 357 Q 256 406 484 357 C 484 357 484 418 460 443 Q 256 492 52 443 C 28 418 28 357 28 357 Z"
        />
      </>
    )}
  </svg>
);

export default SeatIcon;
