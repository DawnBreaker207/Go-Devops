import { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { seatMapApi } from '@/api/seatmap.api';
import { buildGridLayout, groupSeatsByRow, seatsPerRowFromSeats } from '@/features/hall/seatGrid';
import { useHasRole } from '@/hooks/useHasRole';
import { useAuthStore } from '@/stores/authStore';
import { PATHS } from '@/routes/paths';
import type { SeatMapSeat } from '@/types';
import { errorMessage } from '@/utils/error';
import { formatDateTime, formatVND } from '@/utils/format';
import './SelectSeatPage.css';

/**
 * Man chon ghe cua khach.
 *
 * Ba dieu quyet dinh logic o day, deu tu hop dong backend:
 *
 * 1. `showtime_seat_id` la field DUY NHAT co omitempty. Ghe thieu no van ve ra
 *    `status: 'available'` - trong nhu dat duoc nhung khong dat duoc, vi
 *    /orders/hold chi nhan id do. Coi nhu khong chon duoc.
 * 2. Ghe `is_gap` VAN duoc tra ve, VAN co status available va VAN co
 *    showtime_seat_id. Giu no la 400/40001. Khong ve, khong cho bam.
 * 3. `price: 0` nghia la phong CHUA cau hinh gia cho loai ghe do, khong phai
 *    mien phi. Giu ghe do la 409/40900.
 *
 * Toan hoc luoi dung lai `@/features/hall/seatGrid` - cung mot bai toan cot co
 * lo hong va loi di la track rieng, khong viet lai lan thu hai.
 */

/** Toi da cua backend tren mot lan giu cho. */
const MAX_SEATS = 8;

type SeatVisual = 'available' | 'selected' | 'sold' | 'held' | 'blank';

const seatVisual = (seat: SeatMapSeat, selected: boolean): SeatVisual => {
  // Thu tu kiem tra co y: gap va thieu showtime_seat_id phai thang truoc
  // `status`, vi ca hai deu bao cao status 'available'.
  if (seat.is_gap || !seat.showtime_seat_id) return 'blank';
  if (selected) return 'selected';
  if (seat.status === 'sold') return 'sold';
  if (seat.status === 'held') return 'held';
  return 'available';
};

export const SelectSeatPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { showtimeId } = useParams<{ showtimeId: string }>();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  // Giu cho la RequireRoles(customer): admin/staff xem duoc so do nhung bam
  // "thanh toan" se an 403. Noi truoc thay vi de ho bam roi moi biet.
  const isCustomer = useHasRole('customer');

  const [selected, setSelected] = useState<Set<string>>(new Set());

  const seatMap = useQuery({
    queryKey: ['seatmap', showtimeId],
    queryFn: () => seatMapApi.forShowtime(showtimeId as string),
    enabled: Boolean(showtimeId) && isAuthenticated,
    // Ghe nguoi khac giu doi rat nhanh; backend co SSE nhung man nay chua dung,
    // nen giu du lieu tuoi trong thoi gian ngan.
    staleTime: 10_000,
  });

  const seats = useMemo(() => seatMap.data?.seats ?? [], [seatMap.data]);
  const rows = useMemo(() => groupSeatsByRow(seats), [seats]);
  const layout = useMemo(
    () => buildGridLayout(seatsPerRowFromSeats(seats), seatMap.data?.aisle_after_cols ?? []),
    [seats, seatMap.data?.aisle_after_cols]
  );

  const selectedSeats = useMemo(
    () => seats.filter((s) => s.showtime_seat_id && selected.has(s.showtime_seat_id)),
    [seats, selected]
  );
  const total = selectedSeats.reduce((sum, s) => sum + s.price, 0);

  const toggle = (seat: SeatMapSeat) => {
    const id = seat.showtime_seat_id;
    if (!id) return;
    setSelected((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else if (next.size < MAX_SEATS) next.add(id);
      return next;
    });
  };

  if (!isAuthenticated) {
    // /shows/:id/seats can JWT, nen chua dang nhap thi khong goi duoc.
    return (
      <div className="cp-notice cp-notice--info">
        <span>{t('customer.seatsNeedLogin')}</span>
        <button
          type="button"
          className="cp-btn cp-btn--primary"
          onClick={() => navigate(PATHS.login, { state: { from: location.pathname } })}
        >
          {t('customer.login')}
        </button>
      </div>
    );
  }

  if (seatMap.error) {
    return (
      <div className="cp-notice cp-notice--error" role="alert">
        {/* 404 o day co hai nghia ("khong ton tai" va "suat khong con mo ban")
            dung chung ma 40400, chi phan biet duoc bang cau tieng Anh cua
            backend - nen hien nguyen van thay vi doan. */}
        {errorMessage(seatMap.error, t('common.somethingWrong'))}
      </div>
    );
  }

  if (seatMap.isLoading) return <p className="cp-muted">{t('common.loading')}</p>;
  if (!seatMap.data) return null;

  const show = seatMap.data;
  const screenBar = (
    <>
      <div className="cp-screen" />
      <div className="cp-screen__label">{t('customer.screen')}</div>
    </>
  );

  return (
    <>
      <div className="cp-seatmap__head">
        <span className="cp-seatmap__movie">{show.movie_title}</span>
        <span className="cp-seatmap__when">
          {show.hall_name} · {formatDateTime(show.start_at)}
        </span>
      </div>

      {!isCustomer ? (
        <div className="cp-notice cp-notice--info">{t('customer.operatorCannotBook')}</div>
      ) : null}

      <div className="cp-legend">
        {(
          [
            ['available', 'var(--cp-seat-available)'],
            ['selected', 'var(--cp-seat-selected)'],
            ['held', 'var(--cp-seat-held)'],
            ['sold', 'var(--cp-seat-sold)'],
          ] as const
        ).map(([key, colour]) => (
          <span key={key} className="cp-legend__item">
            <span className="cp-legend__swatch" style={{ background: colour }} />
            {t(`customer.seat_${key}`)}
          </span>
        ))}
      </div>

      <div className="cp-seatmap__scroll">
        <div className="cp-seatmap__grid">
          {show.screen_position === 'front' ? screenBar : null}

          {rows.map((row) => (
            <div key={row.rowLabel} className="cp-seatrow">
              <span className="cp-seatrow__label">{row.rowLabel}</span>
              <div
                className="cp-seatrow__seats"
                style={{ gridTemplateColumns: layout.templateColumns }}
              >
                {row.seats.map((seat) => {
                  const isSelected = Boolean(
                    seat.showtime_seat_id && selected.has(seat.showtime_seat_id)
                  );
                  const visual = seatVisual(seat, isSelected);
                  const disabled = visual !== 'available' && visual !== 'selected';
                  return (
                    <button
                      key={seat.id}
                      type="button"
                      className={`cp-seat cp-seat--${visual}`}
                      style={{
                        gridColumn: `${layout.lineOf(seat.col_number)} / span ${layout.spanOf(
                          seat.col_number,
                          seat.col_span
                        )}`,
                      }}
                      aria-label={seat.label}
                      aria-pressed={isSelected}
                      disabled={disabled}
                      title={`${seat.label} · ${formatVND(seat.price)}`}
                      onClick={() => toggle(seat)}
                    >
                      {visual === 'blank' ? '' : seat.col_number}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}

          {show.screen_position === 'back' ? screenBar : null}
        </div>
      </div>

      <div className="cp-summary">
        <div>
          <span className="cp-summary__block-label">{t('customer.total')}</span>
          <span className="cp-summary__amount tabular-nums">{formatVND(total)}</span>
        </div>
        <div>
          <span className="cp-summary__block-label">
            {t('customer.seatCount', { count: selectedSeats.length, max: MAX_SEATS })}
          </span>
          <span className="cp-summary__seats">
            {selectedSeats.length > 0
              ? selectedSeats.map((s) => s.label).join(', ')
              : t('customer.noSeatPicked')}
          </span>
        </div>
        <div className="cp-summary__actions">
          <button type="button" className="cp-btn cp-btn--ghost" onClick={() => navigate(-1)}>
            {t('customer.back')}
          </button>
          <button
            type="button"
            className="cp-btn cp-btn--primary"
            disabled={selectedSeats.length === 0 || !isCustomer}
          >
            {t('customer.proceed')}
          </button>
        </div>
      </div>
    </>
  );
};

export default SelectSeatPage;
