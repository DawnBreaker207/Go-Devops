import { Alert, Card, Col, Empty, Row, Skeleton, Spin, Statistic, theme as antdTheme } from 'antd';
import {
  ArrowDownOutlined,
  ArrowUpOutlined,
  DollarOutlined,
  LineChartOutlined,
  PieChartOutlined,
  TagOutlined,
} from '@ant-design/icons';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Area,
  AreaChart,
  CartesianGrid,
  Cell,
  Legend,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { useBreakdown, useDailyReport } from '../hooks/useDashboard';
import {
  bucketize,
  fillDays,
  filterWindows,
  trendOf,
  type FilterSelection,
} from './dashboardRange';
import TimeFilter from './TimeFilter';
import NowShowingGrid from './NowShowingGrid';
import { formatNumber, formatVND } from '@/utils/format';
import { safeMessage } from '@/utils/error';

const PROVIDER_COLORS = ['#E4002B', '#2563EB', '#D97706', '#2E7D32', '#7A4405', '#1D4ED8'];

const TrendBadge = ({
  value,
  suffix,
  unit = '%',
}: {
  value: number | null;
  suffix: string;
  unit?: string;
}) => {
  const { token } = antdTheme.useToken();
  if (value === null) {
    return <span style={{ color: token.colorTextSecondary, fontSize: 12 }}>— {suffix}</span>;
  }
  const up = value >= 0;
  return (
    <span style={{ color: up ? '#2E7D32' : token.colorError, fontSize: 12 }}>
      {up ? <ArrowUpOutlined /> : <ArrowDownOutlined />} {Math.abs(value)}
      {unit} {suffix}
    </span>
  );
};

/** Overview tab: filter, 4 KPI cards with trend, revenue area, payment donut, now-showing grid (only here). */
export const OverviewTab = () => {
  const { t } = useTranslation();
  const { token } = antdTheme.useToken();
  const [filter, setFilter] = useState<FilterSelection>({ type: 'week', date: new Date() });
  const { current, previous, bucket, span } = filterWindows(filter);

  const cur = useBreakdown(current, true);
  const prev = useBreakdown(previous, true);
  const daily = useDailyReport(current, true);
  const prevDaily = useDailyReport(previous, true);

  const error = cur.error ?? prev.error ?? daily.error ?? prevDaily.error;
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
  // Skeleton only before the FIRST payload; later refetches (filter change)
  // keep old data on screen via placeholderData (no flash).
  if (cur.isLoading || !cur.data || prev.isLoading || !prev.data) {
    return <Skeleton active paragraph={{ rows: 6 }} />;
  }
  const updating = cur.isFetching || prev.isFetching || daily.isFetching || prevDaily.isFetching;

  const revenueTrend = trendOf(cur.data.total_revenue, prev.data.total_revenue);
  const ticketsTrend = trendOf(cur.data.tickets_sold, prev.data.tickets_sold);
  const suffix = t('dashboard.vsPrev', { days: span });

  const cap = (daily.data?.days ?? []).reduce((s, d) => s + d.capacity, 0);
  const seats = (daily.data?.days ?? []).reduce((s, d) => s + d.seats_sold, 0);
  const occupancy = cap > 0 ? (100 * seats) / cap : 0;
  const prevCap = (prevDaily.data?.days ?? []).reduce((s, d) => s + d.capacity, 0);
  const prevSeats = (prevDaily.data?.days ?? []).reduce((s, d) => s + d.seats_sold, 0);
  const prevOccupancy = prevCap > 0 ? (100 * prevSeats) / prevCap : 0;
  const occupancyTrend = prevCap > 0 ? Math.round((occupancy - prevOccupancy) * 10) / 10 : null;
  const avgTicket = cur.data.tickets_sold > 0 ? cur.data.total_revenue / cur.data.tickets_sold : 0;

  const filled = fillDays(current.from, current.to, cur.data.days, (date) => ({
    date,
    revenue: 0,
    tickets: 0,
  }));
  const line = bucketize(filled, bucket).map((b) => ({
    date: b.label,
    fullDate: b.fullLabel,
    revenue: b.revenue,
  }));
  const pie = cur.data.providers.map((p) => ({
    name: p.provider,
    value: p.revenue,
    count: p.count,
  }));

  const kpis = [
    {
      key: 'revenue',
      title: t('dashboard.kpiRevenue'),
      value: formatVND(cur.data.total_revenue),
      prefix: <DollarOutlined />,
      trend: <TrendBadge value={revenueTrend} suffix={suffix} />,
      color: token.colorPrimary,
    },
    {
      key: 'tickets',
      title: t('dashboard.kpiTickets'),
      value: formatNumber(cur.data.tickets_sold),
      prefix: <TagOutlined />,
      trend: <TrendBadge value={ticketsTrend} suffix={suffix} />,
    },
    {
      key: 'occupancy',
      title: t('dashboard.kpiOccupancy'),
      value: `${formatNumber(Math.round(occupancy * 100) / 100)}%`,
      prefix: <PieChartOutlined />,
      trend: <TrendBadge value={occupancyTrend} suffix={suffix} unit="pp" />,
    },
    {
      key: 'avg',
      title: t('dashboard.kpiAvgTicket'),
      value: formatVND(Math.round(avgTicket)),
      prefix: <LineChartOutlined />,
      trend: null,
    },
  ];

  return (
    <>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <TimeFilter value={filter} onChange={setFilter} />
        </div>
        {updating ? <Spin size="small" style={{ marginBottom: 16, flex: 'none' }} /> : null}
      </div>
      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        {kpis.map((k) => (
          <Col key={k.key} xs={24} sm={12} lg={6}>
            <Card style={{ boxShadow: token.boxShadowTertiary }}>
              <Statistic
                title={k.title}
                value={k.value}
                prefix={k.prefix}
                valueStyle={k.color ? { color: k.color } : undefined}
              />
              <div style={{ marginTop: 4 }}>{k.trend}</div>
            </Card>
          </Col>
        ))}
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={14}>
          <Card title={t('dashboard.revenueByDay')} style={{ boxShadow: token.boxShadowTertiary }}>
            <ResponsiveContainer width="100%" height={260}>
              <AreaChart data={line} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
                <defs>
                  <linearGradient id="dashRevenue" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={token.colorPrimary} stopOpacity={0.45} />
                    <stop offset="100%" stopColor={token.colorPrimary} stopOpacity={0.05} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke={token.colorFillSecondary} />
                <XAxis
                  dataKey="date"
                  tick={{ fontSize: 11, fill: token.colorTextSecondary }}
                  tickLine={false}
                  minTickGap={24}
                />
                <YAxis
                  tick={{ fontSize: 11, fill: token.colorTextSecondary }}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(v: number) =>
                    v >= 1_000_000 ? `${Math.round(v / 1_000_000)}M` : `${Math.round(v / 1000)}k`
                  }
                  width={44}
                />
                <Tooltip
                  formatter={(v) => [formatVND(Number(v)), t('dashboard.kpiRevenue')]}
                  labelFormatter={(_, payload) => payload?.[0]?.payload?.fullDate ?? ''}
                />
                <Area
                  type="monotone"
                  dataKey="revenue"
                  stroke={token.colorPrimary}
                  strokeWidth={2.5}
                  fill="url(#dashRevenue)"
                  dot={false}
                  activeDot={{ r: 4 }}
                />
              </AreaChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={10}>
          <Card title={t('dashboard.paySplit')} style={{ boxShadow: token.boxShadowTertiary }}>
            {pie.length === 0 ? (
              <Empty description={t('common.noData')} />
            ) : (
              <ResponsiveContainer width="100%" height={260}>
                <PieChart>
                  <Tooltip
                    formatter={(v, name, entry) => [
                      `${formatVND(Number(v))} · ${entry?.payload?.count ?? 0}`,
                      name,
                    ]}
                  />
                  <Legend iconSize={10} wrapperStyle={{ fontSize: 12 }} />
                  <Pie
                    data={pie}
                    dataKey="value"
                    nameKey="name"
                    innerRadius={52}
                    outerRadius={84}
                    paddingAngle={2}
                  >
                    {pie.map((_, i) => (
                      <Cell key={i} fill={PROVIDER_COLORS[i % PROVIDER_COLORS.length]} />
                    ))}
                  </Pie>
                </PieChart>
              </ResponsiveContainer>
            )}
          </Card>
        </Col>
      </Row>

      <NowShowingGrid />
    </>
  );
};

export default OverviewTab;
