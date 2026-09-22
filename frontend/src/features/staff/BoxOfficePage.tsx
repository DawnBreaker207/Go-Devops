import { useState } from 'react';
import { Alert, Card, Col, DatePicker, Row, Statistic, Table, Tabs, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import type dayjs from 'dayjs';
import PageHeader from '@/components/PageHeader';
import TableCard from '@/components/TableCard';
import CounterSellPanel from './components/CounterSellPanel';
import OrderLookupPanel from './components/OrderLookupPanel';
import ShowtimeTicketsPanel from './components/ShowtimeTicketsPanel';
import { useStaffOverview } from './hooks/useBoxOffice';
import type { ShowtimeStatus, StaffShowtime } from '@/types';
import { errorMessage } from '@/utils/error';
import { API_DATE_FORMAT, DATE_FORMAT, formatDateTime, formatVND } from '@/utils/format';

const STATUS_COLOR: Record<ShowtimeStatus, string> = {
  open: 'green',
  closed: 'default',
  // Cancelled is a refund, not a quiet close - it must not read as "default".
  cancelled: 'red',
};

/** A map, not a ternary. The old `value === 'open' ? 'Open' : 'Closed'` rendered
 *  a CANCELLED showtime as "closed", which is a different thing entirely. */
const STATUS_LABEL_KEY: Record<ShowtimeStatus, string> = {
  open: 'showtime.statusOpen',
  closed: 'showtime.statusClosed',
  cancelled: 'showtime.statusCancelled',
};

/** One-day board: shows with held/sold/free/checked-in plus counter totals and pending check-ins. GET /staff/overview joins all three in one call. */
const DashboardTab = () => {
  const { t } = useTranslation();
  const [date, setDate] = useState<dayjs.Dayjs | null>(null);

  const { data, isFetching, error } = useStaffOverview(date?.format(API_DATE_FORMAT));

  const columns: ColumnsType<StaffShowtime> = [
    { title: t('showtime.movie'), dataIndex: 'movie_title', key: 'movie_title', ellipsis: true },
    { title: t('showtime.hall'), dataIndex: 'hall_name', key: 'hall_name', width: 140 },
    {
      title: t('showtime.startAt'),
      dataIndex: 'start_at',
      key: 'start_at',
      width: 160,
      render: (value: string) => formatDateTime(value),
    },
    {
      title: t('showtime.status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (value: ShowtimeStatus) => (
        <Tag color={STATUS_COLOR[value]} bordered={false}>
          {t(STATUS_LABEL_KEY[value])}
        </Tag>
      ),
    },
    {
      title: t('boxOffice.capacity'),
      dataIndex: 'capacity',
      key: 'capacity',
      width: 90,
      align: 'right',
    },
    { title: t('boxOffice.held'), dataIndex: 'held', key: 'held', width: 80, align: 'right' },
    { title: t('boxOffice.sold'), dataIndex: 'sold', key: 'sold', width: 80, align: 'right' },
    {
      title: t('boxOffice.available'),
      dataIndex: 'available',
      key: 'available',
      width: 100,
      align: 'right',
    },
    {
      title: t('boxOffice.checkedIn'),
      dataIndex: 'checked_in',
      key: 'checked_in',
      width: 100,
      align: 'right',
    },
  ];

  return (
    <>
      <DatePicker
        style={{ marginBottom: 16 }}
        format={DATE_FORMAT}
        value={date}
        placeholder={t('boxOffice.today')}
        onChange={setDate}
        allowClear
      />

      {error ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={errorMessage(error, t('common.somethingWrong'))}
        />
      ) : null}

      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title={t('boxOffice.counterSalesCount')}
              value={data?.counter_sales_count}
              loading={isFetching}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title={t('boxOffice.counterSalesTotal')}
              value={data ? formatVND(data.counter_sales_total) : undefined}
              loading={isFetching}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title={t('boxOffice.awaitingCheckin')}
              value={data?.awaiting_checkin}
              loading={isFetching}
            />
          </Card>
        </Col>
      </Row>

      <TableCard>
        <Table<StaffShowtime>
          rowKey="id"
          columns={columns}
          dataSource={data?.showtimes ?? []}
          loading={isFetching}
          scroll={{ x: 900 }}
          pagination={false}
        />
      </TableCard>
    </>
  );
};

export const BoxOfficePage = () => {
  const { t } = useTranslation();

  return (
    <>
      <PageHeader title={t('boxOffice.title')} />
      <Tabs
        defaultActiveKey="dashboard"
        items={[
          { key: 'dashboard', label: t('boxOffice.tabDashboard'), children: <DashboardTab /> },
          { key: 'sell', label: t('boxOffice.tabSell'), children: <CounterSellPanel /> },
          { key: 'lookup', label: t('boxOffice.tabLookup'), children: <OrderLookupPanel /> },
          { key: 'checkin', label: t('boxOffice.tabCheckin'), children: <ShowtimeTicketsPanel /> },
        ]}
      />
    </>
  );
};

export default BoxOfficePage;
