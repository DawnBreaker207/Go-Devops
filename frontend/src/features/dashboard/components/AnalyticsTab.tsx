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
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Bar,
  CartesianGrid,
  ComposedChart,
  Legend,
  Line,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { useBreakdown } from '../hooks/useDashboard';
import { filterWindows, type FilterSelection } from './dashboardRange';
import TimeFilter from './TimeFilter';
import { formatNumber, formatVND } from '@/utils/format';
import { safeMessage } from '@/utils/error';

const { Title } = Typography;

/** Dark rank pill for the top three. */
const RankPill = ({ rank }: { rank: number }) => {
  const { token } = antdTheme.useToken();
  const top = rank <= 3;
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        minWidth: 24,
        height: 24,
        borderRadius: 12,
        padding: '0 6px',
        fontSize: 12,
        fontWeight: 700,
        background: top ? token.colorText : token.colorFillSecondary,
        color: top ? token.colorBgContainer : token.colorTextSecondary,
      }}
    >
      {rank}
    </span>
  );
};

/**
 * Analytics tab: top-film/hall Pareto charts + tables, sharing the time filter.
 */
export const AnalyticsTab = () => {
  const { t } = useTranslation();
  const { token } = antdTheme.useToken();
  const [filter, setFilter] = useState<FilterSelection>({ type: 'week', date: new Date() });
  const { current } = filterWindows(filter);
  const { data, isLoading, error } = useBreakdown(current, true);

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
  // Skeleton only before the FIRST payload; later refetches keep old data (placeholderData).
  if (isLoading || !data) {
    return <Skeleton active paragraph={{ rows: 6 }} />;
  }

  const maxMovie = Math.max(1, ...data.movies.map((m) => m.revenue));
  const maxHall = Math.max(1, ...data.halls.map((h) => h.revenue));
  const movieBars = data.movies.slice(0, 8).map((m) => ({
    name: m.title.length > 18 ? `${m.title.slice(0, 17)}…` : m.title,
    revenue: m.revenue,
  }));
  const hallBars = data.halls.slice(0, 8).map((h) => ({
    name: h.name.length > 18 ? `${h.name.slice(0, 17)}…` : h.name,
    revenue: h.revenue,
  }));

  // Pareto: bars + cumulative % + 80% line. Recomputed per row (no render-time mutation); n <= 8.
  const pareto = (items: { name: string; revenue: number }[]) => {
    const total = Math.max(
      1,
      items.reduce((s, i) => s + i.revenue, 0)
    );
    const rows = items.map((item, i) => ({
      ...item,
      cumulative:
        Math.round((items.slice(0, i + 1).reduce((s, x) => s + x.revenue, 0) / total) * 100 * 10) /
        10,
    }));
    return (
      <ResponsiveContainer width="100%" height={Math.max(160, rows.length * 40)}>
        <ComposedChart data={rows} margin={{ top: 8, right: 0, bottom: 0, left: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke={token.colorFillSecondary} />
          <XAxis
            dataKey="name"
            tick={{ fontSize: 11, fill: token.colorTextSecondary }}
            tickLine={false}
            interval={0}
            angle={-18}
            dy={10}
            height={52}
          />
          <YAxis
            yAxisId="left"
            tick={{ fontSize: 11, fill: token.colorTextSecondary }}
            tickLine={false}
            axisLine={false}
            tickFormatter={(v: number) =>
              v >= 1_000_000 ? `${Math.round(v / 1_000_000)}M` : `${Math.round(v / 1000)}k`
            }
            width={44}
          />
          <YAxis
            yAxisId="right"
            orientation="right"
            domain={[0, 100]}
            tick={{ fontSize: 11, fill: token.colorTextSecondary }}
            tickLine={false}
            axisLine={false}
            tickFormatter={(v: number) => `${v}%`}
            width={44}
          />
          <Tooltip
            formatter={(v, name) =>
              name === 'cumulative'
                ? [`${v}%`, t('dashboard.paretoCumulative')]
                : [formatVND(Number(v)), t('dashboard.colRevenue')]
            }
          />
          <Legend iconSize={10} wrapperStyle={{ fontSize: 12 }} />
          <Bar
            yAxisId="left"
            dataKey="revenue"
            name={t('dashboard.colRevenue')}
            fill={token.colorPrimary}
            radius={[4, 4, 0, 0]}
            barSize={26}
          />
          <Line
            yAxisId="right"
            type="monotone"
            dataKey="cumulative"
            name={t('dashboard.paretoCumulative')}
            stroke="#D97706"
            strokeWidth={2}
            dot={false}
          />
          <ReferenceLine
            yAxisId="right"
            y={80}
            stroke={token.colorError}
            strokeDasharray="5 3"
            label={{
              value: '80%',
              fontSize: 11,
              fill: token.colorError,
              position: 'insideTopRight',
            }}
          />
        </ComposedChart>
      </ResponsiveContainer>
    );
  };

  // Single-hall cinema: hall ranking is meaningless (one row, always 100%) -
  // hide the hall cards and let films go full width.
  const singleHall = data.halls.length <= 1;

  return (
    <>
      <TimeFilter value={filter} onChange={setFilter} />
      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} lg={singleHall ? 24 : 12}>
          <Card title={t('dashboard.paretoMovies')} style={{ boxShadow: token.boxShadowTertiary }}>
            {movieBars.length === 0 ? (
              <Empty description={t('common.noData')} />
            ) : (
              pareto(movieBars)
            )}
          </Card>
        </Col>
        {singleHall ? null : (
          <Col xs={24} lg={12}>
            <Card title={t('dashboard.paretoHalls')} style={{ boxShadow: token.boxShadowTertiary }}>
              {hallBars.length === 0 ? (
                <Empty description={t('common.noData')} />
              ) : (
                pareto(hallBars)
              )}
            </Card>
          </Col>
        )}
      </Row>
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={singleHall ? 24 : 12}>
          <Card title={t('dashboard.topMovies')} style={{ boxShadow: token.boxShadowTertiary }}>
            {data.movies.length === 0 ? (
              <Empty description={t('common.noData')} />
            ) : (
              <Table
                rowKey="movie_id"
                size="small"
                pagination={false}
                dataSource={data.movies}
                columns={[
                  {
                    title: t('dashboard.colRank'),
                    key: 'rank',
                    width: 52,
                    render: (_, __, i) => <RankPill rank={i + 1} />,
                  },
                  {
                    title: t('dashboard.colMovie'),
                    dataIndex: 'title',
                    key: 'title',
                    render: (v: string, r) => (
                      <div>
                        <div style={{ fontWeight: 600 }}>{v}</div>
                        <div
                          style={{
                            height: 4,
                            borderRadius: 2,
                            background: token.colorFillSecondary,
                            marginTop: 4,
                          }}
                        >
                          <div
                            style={{
                              width: `${Math.max(3, Math.round((r.revenue / maxMovie) * 100))}%`,
                              height: 4,
                              borderRadius: 2,
                              background: token.colorPrimary,
                            }}
                          />
                        </div>
                      </div>
                    ),
                  },
                  {
                    title: t('dashboard.colTickets'),
                    dataIndex: 'tickets',
                    key: 'tickets',
                    align: 'right',
                    render: (v: number) => formatNumber(v),
                  },
                  {
                    title: t('dashboard.colRevenue'),
                    dataIndex: 'revenue',
                    key: 'revenue',
                    align: 'right',
                    render: (v: number) => formatVND(v),
                  },
                ]}
              />
            )}
          </Card>
        </Col>
        {singleHall ? null : (
          <Col xs={24} lg={12}>
            <Card title={t('dashboard.topHalls')} style={{ boxShadow: token.boxShadowTertiary }}>
              {data.halls.length === 0 ? (
                <Empty description={t('common.noData')} />
              ) : (
                <Table
                  rowKey="hall_id"
                  size="small"
                  pagination={false}
                  dataSource={data.halls}
                  columns={[
                    {
                      title: t('dashboard.colRank'),
                      key: 'rank',
                      width: 52,
                      render: (_, __, i) => <RankPill rank={i + 1} />,
                    },
                    {
                      title: t('dashboard.colHall'),
                      dataIndex: 'name',
                      key: 'name',
                      render: (v: string, r) => (
                        <div>
                          <div style={{ fontWeight: 600 }}>{v}</div>
                          <div
                            style={{
                              height: 4,
                              borderRadius: 2,
                              background: token.colorFillSecondary,
                              marginTop: 4,
                            }}
                          >
                            <div
                              style={{
                                width: `${Math.max(3, Math.round((r.revenue / maxHall) * 100))}%`,
                                height: 4,
                                borderRadius: 2,
                                background: token.colorPrimary,
                              }}
                            />
                          </div>
                        </div>
                      ),
                    },
                    {
                      title: t('dashboard.colTickets'),
                      dataIndex: 'tickets',
                      key: 'tickets',
                      align: 'right',
                      render: (v: number) => formatNumber(v),
                    },
                    {
                      title: t('dashboard.colRevenue'),
                      dataIndex: 'revenue',
                      key: 'revenue',
                      align: 'right',
                      render: (v: number) => formatVND(v),
                    },
                  ]}
                />
              )}
            </Card>
          </Col>
        )}
      </Row>
      <Title level={5} type="secondary" style={{ marginTop: 16, fontWeight: 400, fontSize: 12 }}>
        {current.from} → {current.to}
      </Title>
    </>
  );
};

export default AnalyticsTab;
