import { useMemo, type ReactNode } from 'react';
import { CloseOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Empty, Flex, Popover, Tooltip, Typography, theme as antdTheme } from 'antd';
import { useTranslation } from 'react-i18next';
import type { Hall, Seat } from '@/types';
import { SEAT_TYPE_STYLE } from '../constants';
import {
  AISLE_WIDTH,
  SEAT_SIZE,
  buildGridLayout,
  groupSeatsByRow,
  renderedSeatsPerRow,
} from '../seatGrid';
import './SeatGrid.css';

interface SeatGridProps {
  hall: Hall;
  seats: Seat[];
  /** Only used for SELECTING WHOLE ROWS (row-label click) - single seats no longer
   *  enter this set, see `onSeatClick`. */
  selected: Set<string>;
  onToggleRow: (rowLabel: string) => void;
  /** View mode: nothing editable anymore; seats/row labels/gaps are no longer buttons. */
  readOnly?: boolean;
  /** Click a standard/couple seat -> quick-edit Popover opens right in that cell. The parent
   *  decides: with a pendingMergeSeatId this may complete a merge instead of opening a Popover. */
  onSeatClick: (seat: Seat) => void;
  /** Double-click a standard seat -> parent enters merge-pending state. */
  onSeatDoubleClick: (seat: Seat) => void;
  /** The ONLY seat with an open Popover (unrelated to `selected`). */
  activeSeatId?: string | null;
  seatPopoverContent?: ReactNode;
  /** Seat awaiting merge (after double-click) - drawn with a blinking border. */
  pendingMergeSeatId?: string | null;
  /** "x" button at each standard seat's top-right corner (on hover/focus) - marks a
   *  gap IMMEDIATELY, no Popover, no confirm. */
  onQuickGap: (seat: Seat) => void;
  /** A gap (is_gap) is a "ghost seat" - clicking it fills a seat STRAIGHT away, no
   *  pre-select needed. This is a local (draft) edit, no API call. */
  onFillGap: (seat: Seat) => void;
  onAddRow?: () => void;
  onDeleteRow?: (rowLabel: string) => void;
}

/** Seat grid placing each seat by its own column. */
export const SeatGrid = ({
  hall,
  seats,
  selected,
  onToggleRow,
  readOnly = false,
  onSeatClick,
  onSeatDoubleClick,
  activeSeatId = null,
  seatPopoverContent,
  pendingMergeSeatId = null,
  onQuickGap,
  onFillGap,
  onAddRow,
  onDeleteRow,
}: SeatGridProps) => {
  const { t } = useTranslation();
  const { token } = antdTheme.useToken();

  const rows = useMemo(() => groupSeatsByRow(seats), [seats]);
  // Render by the ACTUAL column count in the seat data, not `hall.seats_per_row`:
  // the two can disagree - see the renderedSeatsPerRow note.
  const seatsPerRow = useMemo(() => renderedSeatsPerRow(hall, seats), [hall, seats]);
  const layout = useMemo(
    () => buildGridLayout(seatsPerRow, hall.aisle_after_cols),
    [seatsPerRow, hall.aisle_after_cols]
  );

  if (seats.length === 0) {
    return <Empty description={t('hall.noSeats')} />;
  }

  const screen = (
    <Flex vertical align="center" gap={4} style={{ marginBlock: 16 }}>
      <div
        style={{
          width: '100%',
          maxWidth: seatsPerRow * SEAT_SIZE + layout.aisles.length * AISLE_WIDTH,
          height: 6,
          borderRadius: 3,
          background: token.colorTextQuaternary,
        }}
      />
      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        {t('hall.screen')}
      </Typography.Text>
    </Flex>
  );

  return (
    <div style={{ overflowX: 'auto', paddingBottom: 8 }}>
      <div style={{ display: 'inline-block', minWidth: '100%' }}>
        {hall.screen_position === 'front' ? screen : null}

        <Flex vertical gap={4} align="center">
          {rows.map((row) => (
            <Flex key={row.rowLabel} align="center" gap={8}>
              <button
                type="button"
                disabled={readOnly}
                onClick={readOnly ? undefined : () => onToggleRow(row.rowLabel)}
                title={t('hall.selectRow', { row: row.rowLabel })}
                style={{
                  position: 'sticky',
                  left: 0,
                  zIndex: 1,
                  width: 32,
                  height: SEAT_SIZE,
                  border: 'none',
                  // Opaque background so scrolled-under seats don't show through the sticky
                  // left label - from the token, never hardcoded.
                  background: token.colorBgContainer,
                  color: token.colorTextSecondary,
                  cursor: readOnly ? 'default' : 'pointer',
                  fontSize: 13,
                  fontWeight: 600,
                }}
              >
                {row.rowLabel}
              </button>

              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: layout.templateColumns,
                  gap: 6,
                  alignItems: 'stretch',
                }}
              >
                {row.seats.map((seat) => {
                  const isSelected = selected.has(seat.id);
                  const line = layout.lineOf(seat.col_number);
                  const span = layout.spanOf(seat.col_number, seat.col_span);
                  const gridColumn = `${line} / span ${span}`;

                  // Gaps are "ghost seats": never handled like a normal seat that gets
                  // SELECTED then converted via toolbar - clicking fills a seat STRAIGHT
                  // away (one step, no pre-select). Icon-only, no text.
                  if (seat.is_gap) {
                    return (
                      <Tooltip key={seat.id} title={readOnly ? undefined : t('hall.fillGapHint')}>
                        <button
                          type="button"
                          disabled={readOnly}
                          aria-label={seat.label}
                          onClick={readOnly ? undefined : () => onFillGap(seat)}
                          style={{
                            gridColumn,
                            height: SEAT_SIZE,
                            minWidth: 0,
                            padding: 0,
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            cursor: readOnly ? 'default' : 'pointer',
                            borderRadius: token.borderRadiusSM,
                            background: token.colorPrimaryBg,
                            color: token.colorPrimary,
                            border: `1px dashed ${token.colorPrimary}`,
                            opacity: readOnly ? 0.5 : 1,
                          }}
                        >
                          {!readOnly ? <PlusOutlined style={{ fontSize: 12 }} /> : null}
                        </button>
                      </Tooltip>
                    );
                  }

                  // View mode must never show a "selected" ring/glow - even if
                  // `selected` still holds some id for any other reason,
                  // this is the last line of defense at render time.
                  const showSelected = isSelected && !readOnly;
                  const isPendingMerge = !readOnly && seat.id === pendingMergeSeatId;

                  const style = SEAT_TYPE_STYLE[seat.seat_type];
                  const seatButton = (
                    <button
                      type="button"
                      disabled={readOnly}
                      aria-label={seat.label}
                      aria-pressed={showSelected}
                      onClick={readOnly ? undefined : () => onSeatClick(seat)}
                      onDoubleClick={readOnly ? undefined : () => onSeatDoubleClick(seat)}
                      title={`${seat.label} · ${t(`hall.seatType_${seat.seat_type}`)}`}
                      className={isPendingMerge ? 'seat-pending-merge' : undefined}
                      style={{
                        width: '100%',
                        height: SEAT_SIZE,
                        minWidth: 0,
                        padding: 0,
                        fontSize: 11,
                        lineHeight: 1,
                        cursor: readOnly ? 'default' : 'pointer',
                        borderRadius: token.borderRadiusSM,
                        background: style.bg,
                        color: style.fg,
                        border: isPendingMerge
                          ? `2px dashed ${token.colorInfo}`
                          : showSelected
                            ? `2px solid ${token.colorPrimary}`
                            : `1px solid ${token.colorBorderSecondary}`,
                        boxShadow: showSelected ? `0 0 0 2px ${token.colorPrimaryBg}` : undefined,
                      }}
                    >
                      {seat.col_number}
                    </button>
                  );

                  // In-place edit: clicking a seat anchors the Popover right there,
                  // instead of dragging eyes down to the bottom pill.
                  const showPopover = !readOnly && seat.id === activeSeatId && seatPopoverContent;

                  return (
                    <span
                      key={seat.id}
                      className="seat-cell"
                      style={{ gridColumn, position: 'relative', display: 'inline-block' }}
                    >
                      {showPopover ? (
                        <Popover
                          open
                          placement="top"
                          content={seatPopoverContent}
                          title={t('hall.quickEdit', { label: seat.label })}
                        >
                          {seatButton}
                        </Popover>
                      ) : (
                        seatButton
                      )}
                      {!readOnly ? (
                        <Tooltip title={t('hall.quickGap')}>
                          <button
                            type="button"
                            className="seat-x"
                            aria-label={t('hall.quickGap')}
                            onClick={(e) => {
                              e.stopPropagation();
                              onQuickGap(seat);
                            }}
                            style={{
                              position: 'absolute',
                              top: -6,
                              right: -6,
                              width: 16,
                              height: 16,
                              padding: 0,
                              borderRadius: '50%',
                              border: `1px solid ${token.colorBorderSecondary}`,
                              background: token.colorBgContainer,
                              color: token.colorTextSecondary,
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'center',
                              cursor: 'pointer',
                              lineHeight: 1,
                            }}
                          >
                            <CloseOutlined style={{ fontSize: 8 }} />
                          </button>
                        </Tooltip>
                      ) : null}
                    </span>
                  );
                })}
              </div>

              {!readOnly && onDeleteRow ? (
                <Tooltip title={t('hall.deleteRow')}>
                  <button
                    type="button"
                    onClick={() => onDeleteRow(row.rowLabel)}
                    aria-label={t('hall.deleteRow')}
                    style={{
                      width: 24,
                      height: SEAT_SIZE,
                      border: 'none',
                      background: 'transparent',
                      color: token.colorError,
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    <DeleteOutlined />
                  </button>
                </Tooltip>
              ) : null}
            </Flex>
          ))}

          {!readOnly && onAddRow ? (
            <Flex align="center" gap={8}>
              <div style={{ width: 32 }} />
              <Tooltip title={t('hall.addRowHint', { count: seatsPerRow })}>
                <button
                  type="button"
                  onClick={onAddRow}
                  aria-label={t('hall.addRow')}
                  style={{
                    width: seatsPerRow * SEAT_SIZE + (seatsPerRow - 1) * 6,
                    height: SEAT_SIZE,
                    border: `1px dashed ${token.colorPrimary}`,
                    borderRadius: token.borderRadiusSM,
                    background: 'transparent',
                    color: token.colorPrimary,
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    opacity: 0.7,
                  }}
                >
                  <PlusOutlined style={{ fontSize: 16 }} />
                </button>
              </Tooltip>
              {/* Spacer as wide as the row-delete icon (24px) so the ghost row's right edge
                  aligns with the real rows above - without it the ghost row runs
                  short and breaks the grid. */}
              {onDeleteRow ? <div style={{ width: 24 }} /> : null}
            </Flex>
          ) : null}
        </Flex>

        {hall.screen_position === 'back' ? screen : null}
      </div>
    </div>
  );
};

export default SeatGrid;
