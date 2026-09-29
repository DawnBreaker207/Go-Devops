import { useMemo } from 'react';
import { CloseOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Checkbox, Empty, Flex, Tooltip, Typography, theme as antdTheme } from 'antd';
import { useTranslation } from 'react-i18next';
import type { Hall, Seat } from '@/types';
import { SEAT_TYPE_STYLE } from '../constants';
import {
  AISLE_WIDTH,
  SEAT_SIZE,
  buildGridLayout,
  groupSeatsByRow,
  isVirtualGapSeatId,
  widestColumn,
} from '../seatGrid';
import './SeatGrid.css';

interface SeatGridProps {
  hall: Hall;
  seats: Seat[];
  selected: Set<string>;
  onToggleRow: (rowLabel: string) => void;
  onToggleSelectAll: (selectAll: boolean) => void;
  readOnly?: boolean;
  onSeatClick: (seat: Seat) => void;
  onSeatDoubleClick: (seat: Seat) => void;
  pendingMergeSeatId?: string | null;
  onQuickGap: (seat: Seat) => void;
  onFillGap: (seat: Seat) => void;
  onAddRow?: () => void;
  onDeleteRow?: (rowLabel: string) => void;
  onAddSeat?: (rowLabel: string) => void;
  aisleAfterColsOverride?: number[];
}

export const SeatGrid = ({
  hall,
  seats,
  selected,
  onToggleRow,
  onToggleSelectAll,
  readOnly = false,
  onSeatClick,
  onSeatDoubleClick,
  pendingMergeSeatId = null,
  onQuickGap,
  onFillGap,
  onAddRow,
  onDeleteRow,
  onAddSeat,
  aisleAfterColsOverride,
}: SeatGridProps) => {
  const { t } = useTranslation();
  const { token } = antdTheme.useToken();

  const rows = useMemo(() => groupSeatsByRow(seats), [seats]);
  // Rendered seats decide the width; declared seats_per_row goes stale after trim.
  const seatsPerRow = useMemo(() => widestColumn(seats), [seats]);
  const isAislePreview = aisleAfterColsOverride !== undefined;
  const aisleAfterCols = aisleAfterColsOverride ?? hall.aisle_after_cols;
  const layout = useMemo(
    () => buildGridLayout(seatsPerRow, aisleAfterCols),
    [seatsPerRow, aisleAfterCols]
  );

  // Virtual slots are display-only: never part of bulk selection.
  const seatIds = useMemo(
    () => seats.filter((s) => !isVirtualGapSeatId(s.id)).map((s) => s.id),
    [seats]
  );
  const allSelected = seatIds.length > 0 && seatIds.every((id) => selected.has(id));
  const someSelected = seatIds.some((id) => selected.has(id));

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
          {!readOnly ? (
            <Flex align="center" gap={8}>
              <Tooltip title={t('hall.selectAll')}>
                <span
                  style={{
                    width: 32,
                    height: SEAT_SIZE,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  <Checkbox
                    checked={allSelected}
                    indeterminate={someSelected && !allSelected}
                    onChange={() => onToggleSelectAll(!allSelected)}
                    aria-label={t('hall.selectAll')}
                  />
                </span>
              </Tooltip>
            </Flex>
          ) : null}
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
                  // Solid backdrop so sliding seats never show through the sticky label.
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
                {layout.aisles.map((col) => (
                  <div
                    key={`aisle-${col}`}
                    aria-hidden
                    style={{
                      gridColumn: `${layout.lineOf(col) + 1} / span 1`,
                      // Pinned: sparse auto-flow must never decide the row.
                      gridRow: 1,
                      alignSelf: 'stretch',
                      justifySelf: 'center',
                      width: 0,
                      borderLeft: `2px dashed ${isAislePreview ? token.colorWarning : token.colorBorderSecondary}`,
                      pointerEvents: 'none',
                    }}
                  />
                ))}
                {row.seats.map((seat) => {
                  const isSelected = selected.has(seat.id);
                  const line = layout.lineOf(seat.col_number);
                  const span = layout.spanOf(seat.col_number, seat.col_span);
                  const gridColumn = `${line} / span ${span}`;

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
                            gridRow: 1,
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
                      {seat.col_span === 2
                        ? `${seat.col_number}-${seat.col_number + 1}`
                        : seat.col_number}
                    </button>
                  );

                  return (
                    <span
                      key={seat.id}
                      className="seat-cell"
                      style={{
                        gridColumn,
                        gridRow: 1,
                        position: 'relative',
                        display: 'inline-block',
                      }}
                    >
                      {seatButton}
                      {!readOnly ? (
                        <Tooltip
                          title={seat.col_span === 2 ? t('hall.splitCouple') : t('hall.quickGap')}
                        >
                          <button
                            type="button"
                            className="seat-x"
                            aria-label={
                              seat.col_span === 2 ? t('hall.splitCouple') : t('hall.quickGap')
                            }
                            onClick={(e) => {
                              e.stopPropagation();
                              if (seat.col_span === 2) {
                                onSeatClick(seat);
                              } else {
                                onQuickGap(seat);
                              }
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

              {!readOnly && onAddSeat ? (
                <Tooltip title={t('hall.addSeatHint')}>
                  <button
                    type="button"
                    onClick={() => onAddSeat(row.rowLabel)}
                    aria-label={t('hall.addSeat', { row: row.rowLabel })}
                    style={{
                      width: SEAT_SIZE,
                      height: SEAT_SIZE,
                      minWidth: 0,
                      flexShrink: 0,
                      padding: 0,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      cursor: 'pointer',
                      borderRadius: token.borderRadiusSM,
                      border: `1px dashed ${token.colorPrimary}`,
                      background: 'transparent',
                      color: token.colorPrimary,
                      opacity: 0.7,
                    }}
                  >
                    <PlusOutlined style={{ fontSize: 12 }} />
                  </button>
                </Tooltip>
              ) : null}

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
              {onAddSeat ? <div style={{ width: SEAT_SIZE }} /> : null}
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
