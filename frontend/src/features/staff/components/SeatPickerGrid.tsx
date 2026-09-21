import { useMemo } from 'react';
import { Space, Typography, theme as antdTheme } from 'antd';
import { useTranslation } from 'react-i18next';
import { buildGridLayout, groupSeatsByRow, seatsPerRowFromSeats } from '@/features/hall/seatGrid';
import type { SeatMap, SeatMapSeat } from '@/types';
import { formatVND } from '@/utils/format';

interface SeatPickerGridProps {
  seatMap: SeatMap;
  selected: Set<string>;
  onToggle: (seat: SeatMapSeat) => void;
  maxSeats: number;
}

type SeatVisual = 'available' | 'selected' | 'sold' | 'held' | 'blank';

/** Gap/missing showtime_seat_id must win over status, since both still report 'available'. */
const seatVisual = (seat: SeatMapSeat, isSelected: boolean): SeatVisual => {
  if (seat.is_gap || !seat.showtime_seat_id) return 'blank';
  if (isSelected) return 'selected';
  if (seat.status === 'sold') return 'sold';
  if (seat.status === 'held') return 'held';
  return 'available';
};

/** Seat map for counter sales using theme tokens. */
export const SeatPickerGrid = ({ seatMap, selected, onToggle, maxSeats }: SeatPickerGridProps) => {
  const { t } = useTranslation();
  const { token } = antdTheme.useToken();

  const seats = seatMap.seats;
  const rows = useMemo(() => groupSeatsByRow(seats), [seats]);
  const layout = useMemo(
    () => buildGridLayout(seatsPerRowFromSeats(seats), seatMap.aisle_after_cols ?? []),
    [seats, seatMap.aisle_after_cols]
  );

  const colorOf = (visual: SeatVisual): { bg: string; color: string; border: string } => {
    switch (visual) {
      case 'selected':
        return { bg: token.colorPrimary, color: token.colorWhite, border: token.colorPrimary };
      case 'sold':
        return {
          bg: token.colorFillTertiary,
          color: token.colorTextDisabled,
          border: token.colorBorder,
        };
      case 'held':
        return {
          bg: token.colorWarningBg,
          color: token.colorWarningText,
          border: token.colorWarning,
        };
      case 'blank':
        return { bg: 'transparent', color: 'transparent', border: 'transparent' };
      default:
        return { bg: token.colorBgContainer, color: token.colorText, border: token.colorBorder };
    }
  };

  return (
    <Space direction="vertical" size="middle" style={{ width: '100%' }}>
      <Space size={16} wrap>
        {(['available', 'selected', 'held', 'sold'] as const).map((key) => {
          const c = colorOf(key);
          return (
            <Space key={key} size={6} align="center">
              <span
                style={{
                  width: 16,
                  height: 16,
                  borderRadius: 4,
                  background: c.bg,
                  border: `1px solid ${c.border}`,
                  display: 'inline-block',
                }}
              />
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {t(`customer.seat_${key}`)}
              </Typography.Text>
            </Space>
          );
        })}
      </Space>

      <div style={{ overflowX: 'auto', paddingBottom: 8 }}>
        <div style={{ display: 'inline-flex', flexDirection: 'column', gap: 6 }}>
          {rows.map((row) => (
            <div key={row.rowLabel} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ width: 20, fontSize: 12, color: token.colorTextSecondary }}>
                {row.rowLabel}
              </span>
              <div style={{ display: 'grid', gridTemplateColumns: layout.templateColumns, gap: 0 }}>
                {row.seats.map((seat) => {
                  const isSelected = Boolean(
                    seat.showtime_seat_id && selected.has(seat.showtime_seat_id)
                  );
                  const visual = seatVisual(seat, isSelected);
                  const c = colorOf(visual);
                  const atMax = !isSelected && selected.size >= maxSeats;
                  // Only available seats (within the cap) or already-picked ones (to unpick) are clickable.
                  const disabled =
                    visual === 'blank' || (visual === 'available' && atMax)
                      ? true
                      : visual !== 'available' && visual !== 'selected';
                  return (
                    <button
                      key={seat.id}
                      type="button"
                      title={`${seat.label} - ${formatVND(seat.price)}`}
                      disabled={disabled}
                      onClick={() => onToggle(seat)}
                      style={{
                        gridColumn: `${layout.lineOf(seat.col_number)} / span ${layout.spanOf(
                          seat.col_number,
                          seat.col_span
                        )}`,
                        width: 32,
                        height: 32,
                        margin: 2,
                        fontSize: 11,
                        borderRadius: 6,
                        background: c.bg,
                        color: c.color,
                        border: `1px solid ${c.border}`,
                        cursor: visual === 'blank' ? 'default' : 'pointer',
                        opacity: visual === 'blank' ? 0 : 1,
                      }}
                    >
                      {visual === 'blank' ? '' : seat.col_number}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      </div>
    </Space>
  );
};

export default SeatPickerGrid;
