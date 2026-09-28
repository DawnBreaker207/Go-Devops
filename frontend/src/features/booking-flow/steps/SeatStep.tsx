import { useEffect, useMemo, useRef, useState, type CSSProperties } from 'react';
import { useTranslation } from 'react-i18next';
import type { SeatMapSeat } from '@/types';
import {
  clampZoom,
  computeFitScale,
  gridPixelWidth,
  ZOOM_DEFAULT,
  ZOOM_MAX,
  ZOOM_MIN,
  ZOOM_STEP,
  type GridLayout,
  type SeatRow,
} from '@/features/hall/seatGrid';
import SeatIcon, { type SeatShape } from '../components/SeatIcon';
import SeatButton, { type SeatVisual } from '../components/SeatButton';
import ScreenArch from '../components/ScreenArch';
import Notice from '@/components/ui/Notice';
import Button from '@/components/ui/Button';
import usePrefersReducedMotion from '@/hooks/usePrefersReducedMotion';
import { INK_60, INK_65, INK_75, INK_BG_035, INK_BORDER_14 } from '@/theme/customerTw';
import type { SeatMap } from '@/types';

// Per-viewer zoom only; broken storage falls back silently, never blocks zoom.
const ZOOM_STORAGE_KEY = 'cp_seatmap_zoom';

const readStoredZoom = (): number | null => {
  try {
    const raw = window.localStorage.getItem(ZOOM_STORAGE_KEY);
    if (!raw) return null;
    const value = Number(raw);
    return Number.isFinite(value) ? clampZoom(value) : null;
  } catch {
    return null;
  }
};

const writeStoredZoom = (value: number): void => {
  try {
    window.localStorage.setItem(ZOOM_STORAGE_KEY, String(value));
  } catch {
    return;
  }
};

const seatVisual = (seat: SeatMapSeat, selected: boolean): SeatVisual => {
  // Gaps/missing showtime_seat_id still report 'available', so check first.
  if (seat.is_gap || !seat.showtime_seat_id) return 'blank';
  if (selected) return 'selected';
  if (seat.status === 'sold') return 'sold';
  if (seat.status === 'held') return 'held';
  return 'available';
};

interface SeatStepProps {
  show: SeatMap;
  rows: SeatRow<SeatMapSeat>[];
  layout: GridLayout;
  selected: Set<string>;
  holdError: string | null;
  isCustomer: boolean;
  onToggle: (seatId: string) => void;
}

export const SeatStep = ({
  show,
  rows,
  layout,
  selected,
  holdError,
  isCustomer,
  onToggle,
}: SeatStepProps) => {
  const { t } = useTranslation();
  const prefersReducedMotion = usePrefersReducedMotion();
  const statusLegend: SeatVisual[] = ['available', 'selected', 'held', 'sold'];
  const typeLegend: Array<{ shape: SeatShape; labelKey: string }> = [
    { shape: 'single', labelKey: 'customer.legendTypeStandard' },
    { shape: 'vip', labelKey: 'customer.legendTypeVip' },
    { shape: 'couple', labelKey: 'customer.legendTypeCouple' },
  ];
  const screenArch = <ScreenArch />;

  // Zoom the inner content, never scale the scroller itself or the scrollbox breaks.
  const scrollerRef = useRef<HTMLDivElement>(null);
  const [zoom, setZoom] = useState<number>(() => readStoredZoom() ?? ZOOM_DEFAULT);

  useEffect(() => {
    writeStoredZoom(zoom);
  }, [zoom]);

  const gridWidthPx = useMemo(
    () => gridPixelWidth(layout.templateColumns),
    [layout.templateColumns]
  );

  const zoomIn = () => setZoom((z) => clampZoom(Math.round((z + ZOOM_STEP) * 100) / 100));
  const zoomOut = () => setZoom((z) => clampZoom(Math.round((z - ZOOM_STEP) * 100) / 100));
  const zoomReset = () => setZoom(ZOOM_DEFAULT);
  // Fit-to-room must also reset scrollLeft, or it re-scales at the old offset and looks dead.
  const zoomToFit = () => {
    const containerWidth = scrollerRef.current?.clientWidth ?? 0;
    setZoom(computeFitScale(gridWidthPx, containerWidth, ZOOM_MIN));
    if (scrollerRef.current) scrollerRef.current.scrollLeft = 0;
  };

  const zoomStyle: CSSProperties = {
    transform: `scale(${zoom})`,
    transformOrigin: 'top center',
    transition: prefersReducedMotion
      ? 'none'
      : 'transform var(--motion-duration-fast) var(--motion-ease-out)',
  };

  return (
    <>
      {!isCustomer ? (
        <Notice variant="info" role="status">
          {t('customer.operatorCannotBook')}
        </Notice>
      ) : null}
      {holdError ? <Notice variant="error">{holdError}</Notice> : null}

      <div className={`relative rounded-2xl border p-4 ${INK_BORDER_14} ${INK_BG_035}`}>
        <div className="absolute top-3 right-3 z-10 flex items-center gap-1">
          <Button
            variant="ghost"
            aria-label={t('customer.zoomOut')}
            title={t('customer.zoomOut')}
            className="min-h-8 px-2.5 text-base leading-none"
            disabled={zoom <= ZOOM_MIN}
            onClick={zoomOut}
          >
            −
          </Button>
          <Button
            variant="ghost"
            aria-label={t('customer.zoomReset')}
            title={t('customer.zoomReset')}
            className="min-h-8 px-2.5 text-xs tabular-nums"
            onClick={zoomReset}
          >
            {Math.round(zoom * 100)}%
          </Button>
          <Button
            variant="ghost"
            aria-label={t('customer.zoomIn')}
            title={t('customer.zoomIn')}
            className="min-h-8 px-2.5 text-base leading-none"
            disabled={zoom >= ZOOM_MAX}
            onClick={zoomIn}
          >
            +
          </Button>
          <Button
            variant="ghost"
            aria-label={t('customer.zoomFit')}
            title={t('customer.zoomFit')}
            className="min-h-8 px-2.5 text-xs"
            onClick={zoomToFit}
          >
            ⤢
          </Button>
        </div>

        {show.screen_position === 'front' ? screenArch : null}

        <div ref={scrollerRef} className="mt-4 overflow-x-auto pt-2 pb-1">
          <div className="inline-flex min-w-full flex-col items-center gap-1.5" style={zoomStyle}>
            {rows.map((row) => (
              <div key={row.rowLabel} className="flex flex-nowrap items-center gap-2">
                <span className={`w-5.5 flex-none text-center text-xs font-bold ${INK_65}`}>
                  {row.rowLabel}
                </span>
                <div
                  className="grid flex-none items-stretch gap-1"
                  style={{ gridTemplateColumns: layout.templateColumns }}
                >
                  {row.seats.map((seat) => {
                    const isSelected = Boolean(
                      seat.showtime_seat_id && selected.has(seat.showtime_seat_id)
                    );
                    const visual = seatVisual(seat, isSelected);
                    const span = layout.spanOf(seat.col_number, seat.col_span);
                    return (
                      <SeatButton
                        key={seat.id}
                        seatId={seat.showtime_seat_id}
                        label={seat.label}
                        price={seat.price}
                        colNumber={seat.col_number}
                        seatType={seat.seat_type}
                        visual={visual}
                        gridColumn={`${layout.lineOf(seat.col_number)} / span ${span}`}
                        wide={span > 1}
                        onToggle={onToggle}
                      />
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
        </div>

        {show.screen_position === 'back' ? screenArch : null}

        <div
          className={`mt-5 flex flex-col gap-3 border-t pt-4 text-xs sm:flex-row sm:gap-8 ${INK_BORDER_14} ${INK_75}`}
        >
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
            <span className={`text-[11px] font-bold tracking-wider uppercase ${INK_60}`}>
              {t('customer.legendStatus')}
            </span>
            {statusLegend.map((key) => (
              <span key={key} className="inline-flex items-center gap-1.5">
                <SeatIcon
                  shape="single"
                  outline={key === 'held'}
                  className={`h-5 w-5 flex-none text-(--cp-seat-${key})`}
                />
                {t(`customer.seat_${key}`)}
              </span>
            ))}
          </div>
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
            <span className={`text-[11px] font-bold tracking-wider uppercase ${INK_60}`}>
              {t('customer.legendType')}
            </span>
            {typeLegend.map((item) => (
              <span key={item.shape} className="inline-flex items-center gap-1.5">
                <SeatIcon
                  shape={item.shape}
                  className="h-5 w-5 flex-none text-(--cp-seat-available)"
                />
                {t(item.labelKey)}
              </span>
            ))}
          </div>
        </div>
      </div>
    </>
  );
};

export default SeatStep;
