import { useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Input,
  Radio,
  Row,
  Select,
  Table,
  Tag,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { useRedeemTicket, useShowtimeOptions, useShowtimeTickets } from '../hooks/useBoxOffice';
import type { RedeemResult, StaffTicket, StaffTicketStatus } from '@/types';
import { errorMessage } from '@/utils/error';
import { formatDateTime } from '@/utils/format';

const REDEEM_STATUS_COLOR: Record<RedeemResult['status'], string> = {
  ok: 'green',
  used: 'orange',
  wrong_show: 'red',
  not_found: 'red',
  too_early: 'gold',
  closed: 'red',
};

/** Per-showtime ticket list with gate check-in. */
export const ShowtimeTicketsPanel = () => {
  const { t } = useTranslation();

  const [search, setSearch] = useState('');
  const [showtimeId, setShowtimeId] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<StaffTicketStatus | undefined>(undefined);
  const [ticketCode, setTicketCode] = useState('');
  const [redeemResult, setRedeemResult] = useState<RedeemResult | null>(null);
  const [redeemError, setRedeemError] = useState<string | null>(null);

  const showtimes = useShowtimeOptions(search || undefined);
  const tickets = useShowtimeTickets(showtimeId, statusFilter);
  const redeem = useRedeemTicket();

  const checkin = async () => {
    const code = ticketCode.trim();
    if (!showtimeId || !code) return;
    setRedeemResult(null);
    setRedeemError(null);
    try {
      const result = await redeem.mutateAsync({ id: code, payload: { showtime_id: showtimeId } });
      setRedeemResult(result);
      setTicketCode('');
    } catch (error) {
      setRedeemError(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const columns: ColumnsType<StaffTicket> = [
    { title: t('booking.seat'), dataIndex: 'seat_label', key: 'seat_label', width: 90 },
    {
      title: t('booking.seatType'),
      dataIndex: 'seat_type',
      key: 'seat_type',
      width: 110,
      render: (value: string) => t(`booking.seatType_${value}`, value),
    },
    {
      title: t('booking.ticketStatus'),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (value: StaffTicketStatus) => (
        <Tag color={value === 'redeemed' ? 'blue' : 'green'} bordered={false}>
          {t(`booking.ticket_${value}`, value)}
        </Tag>
      ),
    },
    { title: t('booking.code'), dataIndex: 'booking_id', key: 'booking_id', ellipsis: true },
    {
      title: t('boxOffice.updatedAt'),
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 160,
      render: (value: string) => formatDateTime(value),
    },
  ];

  return (
    <Row gutter={16}>
      <Col xs={24} lg={14}>
        <Card title={t('boxOffice.pickShowtime')} style={{ marginBottom: 16 }}>
          <Select
            showSearch
            allowClear
            style={{ width: '100%', marginBottom: 16 }}
            placeholder={t('boxOffice.pickShowtimePlaceholder')}
            value={showtimeId ?? undefined}
            filterOption={false}
            loading={showtimes.isFetching}
            onSearch={setSearch}
            onChange={(value) => setShowtimeId(value ?? null)}
            options={(showtimes.data?.items ?? []).map((s) => ({
              value: s.id,
              label: `${s.movie_title} · ${s.hall_name} · ${formatDateTime(s.start_at)}`,
            }))}
          />

          <Radio.Group
            value={statusFilter ?? 'all'}
            onChange={(e) => setStatusFilter(e.target.value === 'all' ? undefined : e.target.value)}
            style={{ marginBottom: 16 }}
          >
            <Radio.Button value="all">{t('boxOffice.ticketFilterAll')}</Radio.Button>
            <Radio.Button value="issued">{t('booking.ticket_issued')}</Radio.Button>
            <Radio.Button value="redeemed">{t('booking.ticket_redeemed')}</Radio.Button>
          </Radio.Group>

          {tickets.error ? (
            <Alert
              type="error"
              showIcon
              style={{ marginBottom: 16 }}
              message={errorMessage(tickets.error, t('common.somethingWrong'))}
            />
          ) : null}

          <Table<StaffTicket>
            rowKey="id"
            size="small"
            columns={columns}
            dataSource={tickets.data ?? []}
            loading={tickets.isFetching}
            locale={{
              emptyText: showtimeId ? t('common.noData') : t('boxOffice.pickShowtimeFirst'),
            }}
            pagination={false}
            scroll={{ y: 420 }}
          />
        </Card>
      </Col>

      <Col xs={24} lg={10}>
        <Card title={t('boxOffice.checkin')}>
          <Input.Search
            placeholder={t('boxOffice.checkinPlaceholder')}
            value={ticketCode}
            disabled={!showtimeId}
            onChange={(e) => setTicketCode(e.target.value)}
            onSearch={checkin}
            enterButton={t('boxOffice.checkinAction')}
            loading={redeem.isPending}
          />
          {!showtimeId ? (
            <Alert
              type="info"
              showIcon
              style={{ marginTop: 16 }}
              message={t('boxOffice.checkinNeedsShowtime')}
            />
          ) : null}

          {redeemError ? (
            <Alert type="error" showIcon style={{ marginTop: 16 }} message={redeemError} />
          ) : null}

          {redeemResult ? (
            <Card
              size="small"
              style={{ marginTop: 16 }}
              title={
                <Tag color={REDEEM_STATUS_COLOR[redeemResult.status]} bordered={false}>
                  {t(`boxOffice.redeemStatus_${redeemResult.status}`)}
                </Tag>
              }
            >
              <Descriptions column={1} size="small">
                {redeemResult.movie_title ? (
                  <Descriptions.Item label={t('showtime.movie')}>
                    {redeemResult.movie_title}
                  </Descriptions.Item>
                ) : null}
                {redeemResult.seat_label ? (
                  <Descriptions.Item label={t('booking.seat')}>
                    {redeemResult.seat_label}
                  </Descriptions.Item>
                ) : null}
                {redeemResult.hall_name ? (
                  <Descriptions.Item label={t('showtime.hall')}>
                    {redeemResult.hall_name}
                  </Descriptions.Item>
                ) : null}
                {redeemResult.start_at ? (
                  <Descriptions.Item label={t('showtime.startAt')}>
                    {formatDateTime(redeemResult.start_at)}
                  </Descriptions.Item>
                ) : null}
                {redeemResult.checkin_opens_at ? (
                  <Descriptions.Item label={t('boxOffice.checkinWindow')}>
                    {formatDateTime(redeemResult.checkin_opens_at)} -{' '}
                    {formatDateTime(redeemResult.checkin_closes_at)}
                  </Descriptions.Item>
                ) : null}
              </Descriptions>
            </Card>
          ) : null}

          <div style={{ marginTop: 16 }}>
            <Button
              disabled={!redeemResult && !redeemError}
              onClick={() => {
                setRedeemResult(null);
                setRedeemError(null);
              }}
            >
              {t('boxOffice.clearResult')}
            </Button>
          </div>
        </Card>
      </Col>
    </Row>
  );
};

export default ShowtimeTicketsPanel;
