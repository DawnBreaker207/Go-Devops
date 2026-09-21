import {
  Alert,
  Card,
  Col,
  Empty,
  Row,
  Skeleton,
  Table,
  Typography,
  theme as antdTheme,
} from 'antd';
import { AlertOutlined, EyeOutlined, LinkOutlined } from '@ant-design/icons';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { PolarAngleAxis, RadialBar, RadialBarChart, ResponsiveContainer } from 'recharts';
import { Bar, BarChart, CartesianGrid, Tooltip, XAxis, YAxis } from 'recharts';
import { useAdminOverview } from '../hooks/useDashboard';
import { PATHS } from '@/routes/paths';
import { formatDateTime, formatNumber, formatVND } from '@/utils/format';
import { safeMessage } from '@/utils/error';
import type { ReactNode } from 'react';

const { Title } = Typography;

/** Alert section: urgency header, count badge, top rows, view-all link. Renders nothing when empty. */
const QueueSection = ({
  title,
  count,
  urgency,
  viewAll,
  children,
}: {
  title: string;
  count: number;
  urgency: 'high' | 'normal';
  viewAll: { to: string; label: string };
  children: ReactNode;
}) => {
  const { token } = antdTheme.useToken();
  if (count === 0) return null;
  const high = urgency === 'high';
  return (
    <div
      style={{
        borderRadius: token.borderRadiusLG,
        border: `1px solid ${high ? token.colorErrorBorder : token.colorBorderSecondary}`,
        background: token.colorBgContainer,
        boxShadow: token.boxShadowTertiary,
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 8,
          padding: '10px 14px',
          borderBottom: `1px solid ${high ? token.colorErrorBorder : token.colorBorderSecondary}`,
          background: high ? token.colorErrorBg : undefined,
        }}
      >
        <span style={{ display: 'flex', alignItems: 'center', gap: 8, minWidth: 0 }}>
          {high ? (
            <AlertOutlined style={{ color: token.colorError, flex: 'none' }} />
          ) : (
            <EyeOutlined style={{ color: token.colorTextSecondary, flex: 'none' }} />
          )}
          <strong style={{ whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {title}
          </strong>
          <span
            style={{
              borderRadius: 12,
              padding: '1px 8px',
              fontSize: 12,
              fontWeight: 700,
              background: high ? token.colorError : token.colorFillSecondary,
              color: high ? '#fff' : token.colorTextSecondary,
              flex: 'none',
            }}
          >
            {count}
          </span>
        </span>
        <Link to={viewAll.to} style={{ fontSize: 12, flex: 'none' }}>
          {viewAll.label} →
        </Link>
      </div>
      <div>{children}</div>
    </div>
  );
};

const QueueRow = ({
  to,
  code,
  middle,
  right,
}: {
  to: string;
  code: string;
  middle: string;
  right: string;
}) => {
  const { token } = antdTheme.useToken();
  return (
    <Link
      to={to}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 10,
        padding: '9px 14px',
        color: 'inherit',
        textDecoration: 'none',
        borderBottom: `1px solid ${token.colorBorderSecondary}`,
      }}
    >
      <span style={{ fontFamily: 'monospace', fontSize: 13, fontWeight: 600, flex: 'none' }}>
        {code}
      </span>
      <span
        style={{
          flex: 1,
          minWidth: 0,
          whiteSpace: 'nowrap',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          fontSize: 13,
          color: token.colorTextSecondary,
        }}
      >
        {middle}
      </span>
      <span style={{ fontSize: 12, color: token.colorTextSecondary, flex: 'none' }}>{right}</span>
      <LinkOutlined style={{ fontSize: 11, color: token.colorTextQuaternary, flex: 'none' }} />
    </Link>
  );
};

/** Operations tab: alert sections + occupancy gauge + hourly bars + upcoming board. Names/times only, never raw errors. */
export const OperationsTab = () => {
  const { t } = useTranslation();
  const { token } = antdTheme.useToken();
  const { data, isLoading, error, dataUpdatedAt } = useAdminOverview(true);

  if (error) {
    return (
      <Alert
        type="error"
        showIcon
        style={{ marginBottom: 16 }}
        message={safeMessage(error, t('common.somethingWrong'))}
      />
    );
  }
  if (isLoading || !data) {
    return <Skeleton active paragraph={{ rows: 6 }} />;
  }

  const { alerts, upcoming_showtimes: upcoming, today } = data;
  const updated =
    dataUpdatedAt > 0
      ? new Date(dataUpdatedAt).toLocaleTimeString([], {
          hour: '2-digit',
          minute: '2-digit',
          second: '2-digit',
        })
      : null;
  const allClear =
    alerts.stuck_refunds.length === 0 &&
    alerts.failed_jobs.length === 0 &&
    alerts.given_up_emails.length === 0;

  const occupancy = today.capacity > 0 ? (100 * today.seats_sold) / today.capacity : 0;

  const byHour = new Map<string, number>();
  for (const s of upcoming) {
    const h = new Date(s.start_at).getHours();
    const key = `${String(h).padStart(2, '0')}:00`;
    byHour.set(key, (byHour.get(key) ?? 0) + s.sold);
  }
  const hourly = [...byHour.entries()]
    .sort(([a], [b]) => (a < b ? -1 : 1))
    .map(([hour, sold]) => ({ hour, sold }));

  return (
    <>
      <div
        style={{
          display: 'flex',
          alignItems: 'baseline',
          justifyContent: 'space-between',
          gap: 8,
          marginBottom: 12,
        }}
      >
        <Title level={5} style={{ margin: 0 }}>
          {t('dashboard.opsAlerts')}
        </Title>
        {updated ? (
          <span style={{ fontSize: 12, color: token.colorTextSecondary }}>
            {t('dashboard.opsUpdatedAt', { time: updated })}
          </span>
        ) : null}
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 12, marginBottom: 16 }}>
        <QueueSection
          title={t('dashboard.stuckRefunds')}
          count={alerts.stuck_refunds.length}
          urgency="high"
          viewAll={{ to: PATHS.bookings, label: t('dashboard.opsViewAll') }}
        >
          {alerts.stuck_refunds.slice(0, 3).map((a) => (
            <QueueRow
              key={a.payment_id}
              to={PATHS.bookings}
              code={`#${a.booking_id.slice(0, 8)}`}
              middle={`${formatVND(a.amount)} · ${t('dashboard.opsAttempts', { count: a.attempts })}`}
              right=""
            />
          ))}
        </QueueSection>
        <QueueSection
          title={t('dashboard.failedJobs')}
          count={alerts.failed_jobs.length}
          urgency="high"
          viewAll={{ to: PATHS.batchJobs, label: t('dashboard.opsViewAll') }}
        >
          {alerts.failed_jobs.slice(0, 3).map((j) => (
            <QueueRow
              key={j.id}
              to={PATHS.batchJobs}
              code={j.job_name}
              middle={formatDateTime(j.started_at)}
              right=""
            />
          ))}
        </QueueSection>
        <QueueSection
          title={t('dashboard.givenUpEmails')}
          count={alerts.given_up_emails.length}
          urgency="normal"
          viewAll={{ to: PATHS.bookings, label: t('dashboard.opsViewAll') }}
        >
          {alerts.given_up_emails.slice(0, 3).map((m) => (
            <QueueRow
              key={m.booking_id}
              to={PATHS.bookings}
              code={`#${m.booking_id.slice(0, 8)}`}
              middle={t('dashboard.opsAttempts', { count: m.attempts })}
              right=""
            />
          ))}
        </QueueSection>
        {allClear ? <Empty description={t('dashboard.opsEmpty')} /> : null}
      </div>

      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} lg={10}>
          <Card
            title={t('dashboard.opsOccupancyToday')}
            style={{ boxShadow: token.boxShadowTertiary }}
          >
            <ResponsiveContainer width="100%" height={190}>
              <RadialBarChart
                cx="50%"
                cy="50%"
                innerRadius="68%"
                outerRadius="100%"
                data={[
                  {
                    name: 'occupancy',
                    value: Math.round(occupancy * 10) / 10,
                    fill: token.colorPrimary,
                  },
                ]}
                startAngle={180}
                endAngle={0}
              >
                <PolarAngleAxis type="number" domain={[0, 100]} tick={false} />
                <RadialBar
                  dataKey="value"
                  background={{ fill: token.colorFillSecondary }}
                  cornerRadius={8}
                />
                <text
                  x="50%"
                  y="62%"
                  textAnchor="middle"
                  fontSize={26}
                  fontWeight={700}
                  fill={token.colorText}
                >
                  {`${formatNumber(Math.round(occupancy * 10) / 10)}%`}
                </text>
                <text
                  x="50%"
                  y="76%"
                  textAnchor="middle"
                  fontSize={12}
                  fill={token.colorTextSecondary}
                >
                  {`${formatNumber(today.seats_sold)}/${formatNumber(today.capacity)}`}
                </text>
              </RadialBarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={14}>
          <Card
            title={t('dashboard.opsHourlySales')}
            style={{ boxShadow: token.boxShadowTertiary }}
          >
            {hourly.length === 0 ? (
              <Empty description={t('common.noData')} />
            ) : (
              <ResponsiveContainer width="100%" height={190}>
                <BarChart data={hourly} margin={{ top: 8, right: 8, bottom: 0, left: -8 }}>
                  <CartesianGrid
                    strokeDasharray="3 3"
                    stroke={token.colorFillSecondary}
                    vertical={false}
                  />
                  <XAxis
                    dataKey="hour"
                    tick={{ fontSize: 11, fill: token.colorTextSecondary }}
                    tickLine={false}
                  />
                  <YAxis
                    tick={{ fontSize: 11, fill: token.colorTextSecondary }}
                    tickLine={false}
                    axisLine={false}
                    allowDecimals={false}
                    width={36}
                  />
                  <Tooltip formatter={(v) => [formatNumber(Number(v)), t('dashboard.colSold')]} />
                  <Bar
                    dataKey="sold"
                    fill={token.colorPrimary}
                    radius={[4, 4, 0, 0]}
                    barSize={26}
                  />
                </BarChart>
              </ResponsiveContainer>
            )}
          </Card>
        </Col>
      </Row>

      <Title level={5} style={{ marginBottom: 12 }}>
        {t('dashboard.upcomingToday')}
      </Title>
      {upcoming.length === 0 ? (
        <Empty description={t('common.noData')} />
      ) : (
        <Table
          rowKey="id"
          size="small"
          pagination={false}
          dataSource={upcoming}
          columns={[
            {
              title: t('dashboard.colTime'),
              dataIndex: 'start_at',
              key: 'time',
              render: (v: string) => formatDateTime(v),
            },
            { title: t('dashboard.colMovie'), dataIndex: 'movie_title', key: 'movie' },
            { title: t('dashboard.colHall'), dataIndex: 'hall_name', key: 'hall' },
            {
              title: t('dashboard.colSold'),
              key: 'sold',
              render: (_, r) => `${formatNumber(r.sold)}/${formatNumber(r.capacity)}`,
            },
            {
              title: t('dashboard.colLeft'),
              dataIndex: 'available',
              key: 'left',
              render: (v: number) => formatNumber(v),
            },
            {
              title: t('dashboard.colCheckedIn'),
              dataIndex: 'checked_in',
              key: 'checkedin',
              render: (v: number) => formatNumber(v),
            },
          ]}
        />
      )}
    </>
  );
};

export default OperationsTab;
