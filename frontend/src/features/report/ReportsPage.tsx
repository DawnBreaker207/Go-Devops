import { useMemo, useState } from 'react';
import {
  Alert,
  Card,
  Col,
  DatePicker,
  Empty,
  Row,
  Space,
  Statistic,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import PageHeader from '@/components/PageHeader';
import ShowtimeBreakdownTable from './components/ShowtimeBreakdownTable';
import { useDailyReport } from './hooks/useReports';
import { type MovieRollupRow, missingDays, occupancyPercent, rollupByMovie } from './reportRollup';
import type { DailyAggregate } from '@/types';
import { errorMessage } from '@/utils/error';
import {
  API_DATE_FORMAT,
  DATE_FORMAT,
  formatNumber,
  formatVND,
  toCinemaTime,
} from '@/utils/format';

/** Mac dinh cua backend khi khong truyen tham so: 7 ngay tinh den hom nay. */
const DEFAULT_DAYS = 7;

export const ReportsPage = () => {
  const { t } = useTranslation();

  // Khoang ngay la trang thai cua rieng man nay, giong bo loc cua cac bang khac
  // (useListQuery chi lo page/page_size/search dung chung).
  const [range, setRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>(() => [
    dayjs().subtract(DEFAULT_DAYS - 1, 'day'),
    dayjs(),
  ]);

  const { data, isFetching, error } = useDailyReport({
    from: range[0].format(API_DATE_FORMAT),
    to: range[1].format(API_DATE_FORMAT),
  });

  const days = useMemo(() => data?.days ?? [], [data]);
  const movieRows = useMemo(() => rollupByMovie(days), [days]);
  const gaps = useMemo(() => (data ? missingDays(data.from, data.to, days) : []), [data, days]);

  const totalSeats = days.reduce((sum, d) => sum + d.seats_sold, 0);
  const totalCapacity = days.reduce((sum, d) => sum + d.capacity, 0);

  const movieColumns: ColumnsType<MovieRollupRow> = [
    { title: t('report.movie'), dataIndex: 'movie', key: 'movie', ellipsis: true },
    {
      title: t('report.showtimeCount'),
      dataIndex: 'showtimes',
      key: 'showtimes',
      width: 110,
      align: 'right',
      render: (value: number) => <span className="tabular-nums">{formatNumber(value)}</span>,
    },
    {
      title: t('report.ticketSales'),
      key: 'seats',
      width: 150,
      align: 'right',
      render: (_, row) => (
        <span className="tabular-nums">
          {formatNumber(row.seatsSold)} / {formatNumber(row.capacity)}
        </span>
      ),
    },
    {
      title: t('report.occupancy'),
      key: 'occupancy',
      width: 110,
      align: 'right',
      render: (_, row) => (
        <span className="tabular-nums">{occupancyPercent(row.seatsSold, row.capacity)}%</span>
      ),
    },
    {
      title: t('report.totalSales'),
      dataIndex: 'revenue',
      key: 'revenue',
      width: 150,
      align: 'right',
      render: (value: number) => <span className="tabular-nums">{formatVND(value)}</span>,
    },
  ];

  const dayColumns: ColumnsType<DailyAggregate> = [
    {
      title: t('report.date'),
      dataIndex: 'report_date',
      key: 'report_date',
      width: 130,
      render: (value: string) => dayjs(value).format(DATE_FORMAT),
    },
    {
      title: t('report.revenue'),
      dataIndex: 'total_revenue',
      key: 'total_revenue',
      width: 150,
      align: 'right',
      render: (value: number) => <span className="tabular-nums">{formatVND(value)}</span>,
    },
    {
      title: t('report.tickets'),
      dataIndex: 'tickets_sold',
      key: 'tickets_sold',
      width: 110,
      align: 'right',
      render: (value: number) => <span className="tabular-nums">{formatNumber(value)}</span>,
    },
    {
      title: t('report.seats'),
      key: 'seats',
      width: 150,
      align: 'right',
      render: (_, row) => (
        <span className="tabular-nums">
          {formatNumber(row.seats_sold)} / {formatNumber(row.capacity)}
        </span>
      ),
    },
    {
      title: t('report.occupancy'),
      dataIndex: 'occupancy_rate',
      key: 'occupancy_rate',
      width: 110,
      align: 'right',
      // Backend da lam ROUND(100.0 * seats_sold / capacity), tuc DA la phan
      // tram. Nhan them 100 lan nua se ra 333% thay vi 3.33%.
      render: (value: number) => <span className="tabular-nums">{value}%</span>,
    },
    {
      title: t('report.closedAt'),
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 160,
      render: (value: string) => (
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {toCinemaTime(value).format('DD/MM/YYYY HH:mm')}
        </Typography.Text>
      ),
    },
  ];

  return (
    <>
      <PageHeader
        title={t('report.title')}
        extra={
          <DatePicker.RangePicker
            allowClear={false}
            format={DATE_FORMAT}
            value={range}
            // Khong cho chon tuong lai: mot ngay chua den thi khong the chot so.
            disabledDate={(current) => current.isAfter(dayjs(), 'day')}
            onChange={(value) => {
              if (value?.[0] && value[1]) setRange([value[0], value[1]]);
            }}
          />
        }
      />

      {error ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          // Loi kiem tra khoang ngay (sai dinh dang, from sau to, qua 366 ngay)
          // la 400/40001 KHONG kem details, nen chi hien duoc cau cua server.
          message={errorMessage(error, t('common.somethingWrong'))}
        />
      ) : null}

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={8}>
          <Card size="small">
            <Statistic
              title={t('report.totalRevenue')}
              value={data?.total_revenue ?? 0}
              formatter={(value) => formatVND(Number(value))}
              loading={isFetching}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card size="small">
            <Statistic
              title={t('report.ticketsSold')}
              value={data?.tickets_sold ?? 0}
              loading={isFetching}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card size="small">
            <Statistic
              title={t('report.occupancy')}
              value={occupancyPercent(totalSeats, totalCapacity)}
              suffix="%"
              loading={isFetching}
            />
          </Card>
        </Col>
      </Row>

      {/*
        Hai con so tren dem theo hai truc khac nhau: doanh thu theo ngay TRA
        TIEN, do lap day theo ngay SUAT CHIEU chay. Phai noi ra, khong thi nguoi
        doc se tuong bao cao bi sai khi thay chung khong khop.
      */}
      <Alert
        type="info"
        showIcon
        closable
        style={{ marginBottom: 16 }}
        message={t('report.axesHint')}
      />

      {gaps.length > 0 ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          message={t('report.missingDaysTitle', { count: gaps.length })}
          description={t('report.missingDaysBody', {
            days: gaps.map((d) => dayjs(d).format(DATE_FORMAT)).join(', '),
          })}
        />
      ) : null}

      <Card
        size="small"
        title={t('report.byMovie')}
        style={{ marginBottom: 16 }}
        styles={{ body: { padding: 0 } }}
      >
        <Table<MovieRollupRow>
          rowKey="movie"
          size="small"
          columns={movieColumns}
          dataSource={movieRows}
          loading={isFetching}
          pagination={false}
          scroll={{ x: 700 }}
          locale={{
            emptyText: (
              <Empty
                description={
                  days.length === 0 ? t('report.emptyNotClosed') : t('report.emptyNoShowtimes')
                }
              />
            ),
          }}
        />
      </Card>

      <Card size="small" title={t('report.byDay')} styles={{ body: { padding: 0 } }}>
        <Table<DailyAggregate>
          rowKey="report_date"
          size="small"
          columns={dayColumns}
          dataSource={days}
          loading={isFetching}
          pagination={false}
          scroll={{ x: 800 }}
          expandable={{
            // Chi mo duoc ngay thuc su co suat chieu; ngay 0 suat thi mui ten
            // chi to ra co gi de xem ma khong co.
            rowExpandable: (row) => (row.breakdown?.showtimes?.length ?? 0) > 0,
            expandedRowRender: (row) => (
              <ShowtimeBreakdownTable showtimes={row.breakdown?.showtimes ?? []} />
            ),
          }}
          locale={{ emptyText: <Empty description={t('report.emptyNotClosed')} /> }}
          summary={() =>
            days.length > 0 ? (
              <Table.Summary fixed>
                <Table.Summary.Row>
                  <Table.Summary.Cell index={0}>
                    <Space size={6}>
                      <strong>{t('report.rangeTotal')}</strong>
                      <Tag bordered={false}>{t('report.dayCount', { count: days.length })}</Tag>
                    </Space>
                  </Table.Summary.Cell>
                  <Table.Summary.Cell index={1} align="right">
                    <strong className="tabular-nums">{formatVND(data?.total_revenue ?? 0)}</strong>
                  </Table.Summary.Cell>
                  <Table.Summary.Cell index={2} align="right">
                    <strong className="tabular-nums">
                      {formatNumber(data?.tickets_sold ?? 0)}
                    </strong>
                  </Table.Summary.Cell>
                  <Table.Summary.Cell index={3} align="right">
                    <span className="tabular-nums">
                      {formatNumber(totalSeats)} / {formatNumber(totalCapacity)}
                    </span>
                  </Table.Summary.Cell>
                  <Table.Summary.Cell index={4} align="right">
                    <span className="tabular-nums">
                      {occupancyPercent(totalSeats, totalCapacity)}%
                    </span>
                  </Table.Summary.Cell>
                  <Table.Summary.Cell index={5} />
                </Table.Summary.Row>
              </Table.Summary>
            ) : null
          }
        />
      </Card>
    </>
  );
};

export default ReportsPage;
