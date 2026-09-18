import { useMemo } from 'react';
import { Empty, Flex, Typography, theme as antdTheme } from 'antd';
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

interface SeatGridProps {
  hall: Hall;
  seats: Seat[];
  selected: Set<string>;
  onToggleSeat: (seat: Seat) => void;
  onToggleRow: (rowLabel: string) => void;
}

/**
 * Ve luoi ghe cua mot phong. Moi ghe duoc dat vao dung vach luoi tinh tu
 * `col_number`, KHONG phai theo thu tu trong mang: ghe doi nuot mot cot nen so
 * cot co lo hong, ve theo chi so mang se lech het tu ghe doi tro di.
 *
 * Man chieu nam tren hay duoi tuy `screen_position`.
 */
export const SeatGrid = ({ hall, seats, selected, onToggleSeat, onToggleRow }: SeatGridProps) => {
  const { t } = useTranslation();
  const { token } = antdTheme.useToken();

  const rows = useMemo(() => groupSeatsByRow(seats), [seats]);
  // Ve theo so cot THUC SU trong du lieu ghe, khong theo `hall.seats_per_row`:
  // hai ben co the lech nhau - xem ghi chu cua renderedSeatsPerRow.
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
                onClick={() => onToggleRow(row.rowLabel)}
                title={t('hall.selectRow', { row: row.rowLabel })}
                style={{
                  width: 28,
                  height: SEAT_SIZE,
                  border: 'none',
                  background: 'transparent',
                  color: token.colorTextSecondary,
                  cursor: 'pointer',
                  fontSize: 12,
                  fontWeight: 600,
                }}
              >
                {row.rowLabel}
              </button>

              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: layout.templateColumns,
                  gap: 4,
                  alignItems: 'stretch',
                }}
              >
                {row.seats.map((seat) => {
                  const style = SEAT_TYPE_STYLE[seat.seat_type];
                  const isSelected = selected.has(seat.id);
                  const line = layout.lineOf(seat.col_number);
                  const span = layout.spanOf(seat.col_number, seat.col_span);

                  return (
                    <button
                      key={seat.id}
                      type="button"
                      aria-label={seat.label}
                      aria-pressed={isSelected}
                      onClick={() => onToggleSeat(seat)}
                      title={`${seat.label} · ${t(`hall.seatType_${seat.seat_type}`)}${
                        seat.is_gap ? ` · ${t('hall.gap')}` : ''
                      }`}
                      style={{
                        gridColumn: `${line} / span ${span}`,
                        height: SEAT_SIZE,
                        minWidth: 0,
                        padding: 0,
                        fontSize: 10,
                        lineHeight: 1,
                        cursor: 'pointer',
                        borderRadius: token.borderRadiusSM,
                        // O gap la cho trong trong luoi chu khong phai ghe, nen
                        // ve rong voi vien dut - van bam duoc de bo danh dau.
                        background: seat.is_gap ? 'transparent' : style.bg,
                        color: seat.is_gap ? token.colorTextQuaternary : style.fg,
                        border: isSelected
                          ? `2px solid ${token.colorPrimary}`
                          : `1px ${seat.is_gap ? 'dashed' : 'solid'} ${token.colorBorderSecondary}`,
                        boxShadow: isSelected ? `0 0 0 2px ${token.colorPrimaryBg}` : undefined,
                      }}
                    >
                      {seat.is_gap ? '' : seat.col_number}
                    </button>
                  );
                })}
              </div>
            </Flex>
          ))}
        </Flex>

        {hall.screen_position === 'back' ? screen : null}
      </div>
    </div>
  );
};

export default SeatGrid;
