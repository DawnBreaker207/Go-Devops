import { useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import {
  Alert,
  App,
  Button,
  Card,
  Descriptions,
  Flex,
  Select,
  Space,
  Spin,
  Tag,
  Typography,
} from 'antd';
import { ArrowLeftOutlined, ReloadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import SeatGrid from './components/SeatGrid';
import LayoutRegenerateModal from './components/LayoutRegenerateModal';
import { useBulkUpdateSeats, useHall, useHallPrices, useHallSeats } from './hooks/useHalls';
import { SEAT_TYPE_STYLE } from './constants';
import { chunkLabels, cleanSeatLabel, summarizeGrid } from './seatGrid';
import type { Seat, SeatChange, SeatType } from '@/types';
import { SEAT_TYPES } from '@/types';
import { PATHS } from '@/routes/paths';
import { errorMessage } from '@/utils/error';
import { formatNumber } from '@/utils/format';

export const HallSeatsPage = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const { id } = useParams<{ id: string }>();

  const hall = useHall(id);
  const seatsQuery = useHallSeats(id);
  const prices = useHallPrices(id);
  const bulkUpdate = useBulkUpdateSeats();

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [regenerateOpen, setRegenerateOpen] = useState(false);

  const seats = useMemo(() => seatsQuery.data ?? [], [seatsQuery.data]);
  const seatById = useMemo(() => new Map(seats.map((s) => [s.id, s])), [seats]);
  const summary = hall.data ? summarizeGrid(hall.data, seats) : null;

  const selectedSeats: Seat[] = useMemo(
    () => [...selected].map((sid) => seatById.get(sid)).filter((s): s is Seat => Boolean(s)),
    [selected, seatById]
  );

  const toggleSeat = (seat: Seat) =>
    setSelected((current) => {
      const next = new Set(current);
      if (next.has(seat.id)) next.delete(seat.id);
      else next.add(seat.id);
      return next;
    });

  const toggleRow = (rowLabel: string) =>
    setSelected((current) => {
      const rowSeats = seats.filter((s) => s.row_label === rowLabel);
      const allSelected = rowSeats.every((s) => current.has(s.id));
      const next = new Set(current);
      rowSeats.forEach((s) => (allSelected ? next.delete(s.id) : next.add(s.id)));
      return next;
    });

  /**
   * Gui mot thay doi cho toan bo ghe dang chon. Chon selector `labels` chu khong
   * phai `rows`/`range` vi day la mot tap tuy y; nhan duoc lam sach truoc khi
   * gui, boi backend chi VIET HOA nhan chu khong trim, va mot nhan khong khop
   * ghe nao la loi 400 lam ROLLBACK CA LO.
   */
  const applyToSelection = async (patch: Pick<SeatChange, 'seat_type' | 'is_gap'>) => {
    if (!id || selectedSeats.length === 0) return;
    const labels = selectedSeats.map((s) => cleanSeatLabel(s.label));
    const batches = chunkLabels(labels);

    try {
      for (const batch of batches) {
        await bulkUpdate.mutateAsync({
          id,
          payload: { changes: batch.map((group) => ({ selector: { labels: group }, ...patch })) },
        });
      }
      message.success(t('hall.seatsUpdated', { count: selectedSeats.length }));
      setSelected(new Set());
    } catch (error) {
      // 409/40900 "hall layout can not be changed when it has bookings" la
      // truong hop thuong gap nhat o day: mot don DANG GIU (pending) cung chan.
      // Thong bao cua backend la tieng Anh nhung doc duoc, dung nuot di.
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  if (hall.isLoading || seatsQuery.isLoading) {
    return (
      <Flex justify="center" style={{ padding: 48 }}>
        <Spin />
      </Flex>
    );
  }

  if (hall.error || seatsQuery.error) {
    return (
      <Alert
        type="error"
        showIcon
        message={errorMessage(hall.error ?? seatsQuery.error, t('common.somethingWrong'))}
      />
    );
  }

  if (!hall.data || !id) return null;

  return (
    <>
      <PageHeader
        title={t('hall.seatsTitle', { name: hall.data.name })}
        extra={
          <Space>
            <Link to={PATHS.halls}>
              <Button icon={<ArrowLeftOutlined />}>{t('hall.backToList')}</Button>
            </Link>
            <Button
              danger
              icon={<ReloadOutlined />}
              onClick={() => setRegenerateOpen(true)}
              loading={prices.isFetching}
            >
              {t('hall.regenerate')}
            </Button>
          </Space>
        }
      />

      <Card size="small" style={{ marginBottom: 16 }}>
        <Descriptions size="small" column={{ xs: 1, sm: 2, lg: 4 }}>
          <Descriptions.Item label={t('hall.grid')}>
            {hall.data.rows} × {hall.data.seats_per_row}
          </Descriptions.Item>
          <Descriptions.Item label={t('hall.screenPosition')}>
            {t(`hall.screen_${hall.data.screen_position}`)}
          </Descriptions.Item>
          <Descriptions.Item label={t('hall.aisles')}>
            {hall.data.aisle_after_cols.length > 0
              ? hall.data.aisle_after_cols.join(', ')
              : t('hall.noAisles')}
          </Descriptions.Item>
          <Descriptions.Item label={t('hall.active')}>
            <Tag color={hall.data.active ? 'green' : 'default'} bordered={false}>
              {t(hall.data.active ? 'hall.activeYes' : 'hall.activeNo')}
            </Tag>
          </Descriptions.Item>
        </Descriptions>

        {summary?.declaredMismatch ? (
          // Bat bien, khong phai loi nhap lieu: CSDL khong ep halls.rows khop
          // voi bang seats. Ve thi da ve theo ghe that roi, nhung van phai noi
          // cho nguoi truc biet thay vi im lang.
          <Alert
            type="warning"
            showIcon
            style={{ marginTop: 8 }}
            message={t('hall.gridMismatch', {
              declaredRows: hall.data.rows,
              declaredCols: hall.data.seats_per_row,
              actualRows: summary.actualRows,
              actualCols: summary.actualSeatsPerRow,
            })}
          />
        ) : null}

        {summary ? (
          // So o luoi va so ghe co the LECH NHAU mot cach hop le: ghe doi nuot
          // mot cot. Va suc chua ban duoc con tru tiep ghe is_gap.
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {t('hall.gridSummary', {
              cells: formatNumber(summary.gridCells),
              seats: formatNumber(summary.seatCount),
              sellable: formatNumber(summary.sellable),
              gaps: formatNumber(summary.gaps),
              doubles: formatNumber(summary.doubleSeats),
            })}
          </Typography.Text>
        ) : null}
      </Card>

      <Card
        size="small"
        style={{ marginBottom: 16 }}
        title={
          <Flex wrap align="center" gap={12}>
            <span>{t('hall.selectedCount', { count: selectedSeats.length })}</span>
            {selectedSeats.length > 0 ? (
              <Button size="small" type="link" onClick={() => setSelected(new Set())}>
                {t('hall.clearSelection')}
              </Button>
            ) : null}
            <Button
              size="small"
              type="link"
              onClick={() => setSelected(new Set(seats.map((s) => s.id)))}
            >
              {t('hall.selectAll')}
            </Button>
          </Flex>
        }
      >
        <Flex wrap align="center" gap={12}>
          <Select<SeatType>
            style={{ width: 200 }}
            placeholder={t('hall.setSeatType')}
            value={null}
            disabled={selectedSeats.length === 0 || bulkUpdate.isPending}
            onChange={(value) => void applyToSelection({ seat_type: value })}
            options={SEAT_TYPES.map((type) => ({
              value: type,
              label: t(`hall.seatType_${type}`),
            }))}
          />
          {/*
            Hai nut LENH chu khong phai mot cong tac trang thai. Dung Segmented
            o day thi antd to sang muc dau tien va nguoi dung doc ra "dang la o
            trong", trong khi that ra chua chon gi ca.
          */}
          <Button
            disabled={selectedSeats.length === 0 || bulkUpdate.isPending}
            onClick={() => void applyToSelection({ is_gap: true })}
          >
            {t('hall.markAsGap')}
          </Button>
          <Button
            disabled={selectedSeats.length === 0 || bulkUpdate.isPending}
            onClick={() => void applyToSelection({ is_gap: false })}
          >
            {t('hall.markAsSeat')}
          </Button>
          <Flex wrap align="center" gap={8} style={{ marginLeft: 'auto' }}>
            {SEAT_TYPES.map((type) => (
              <Tag
                key={type}
                bordered={false}
                style={{
                  background: SEAT_TYPE_STYLE[type].bg,
                  color: SEAT_TYPE_STYLE[type].fg,
                  marginInlineEnd: 0,
                }}
              >
                {t(`hall.seatType_${type}`)}
              </Tag>
            ))}
          </Flex>
        </Flex>
      </Card>

      <Alert
        type="info"
        showIcon
        closable
        style={{ marginBottom: 16 }}
        message={t('hall.seatEditHint')}
      />

      <Card size="small">
        <Spin spinning={bulkUpdate.isPending}>
          <SeatGrid
            hall={hall.data}
            seats={seats}
            selected={selected}
            onToggleSeat={toggleSeat}
            onToggleRow={toggleRow}
          />
        </Spin>
      </Card>

      <LayoutRegenerateModal
        open={regenerateOpen}
        hall={hall.data}
        prices={prices.data ?? []}
        onCancel={() => setRegenerateOpen(false)}
        onDone={() => {
          setRegenerateOpen(false);
          setSelected(new Set());
        }}
      />
    </>
  );
};

export default HallSeatsPage;
