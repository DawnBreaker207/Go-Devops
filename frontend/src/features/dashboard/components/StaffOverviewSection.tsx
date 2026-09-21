import {
  Alert,
  Card,
  Col,
  Empty,
  Row,
  Skeleton,
  Statistic,
  Table,
  Typography,
  theme as antdTheme,
} from 'antd';
import { ShopOutlined, SolutionOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useStaffOverview } from '../hooks/useDashboard';
import { formatDateTime, formatNumber, formatVND } from '@/utils/format';
import { errorMessage } from '@/utils/error';

const { Title } = Typography;

/** Staff dashboard (counter sales + check-ins, no revenue detail), from a single GET /staff/overview. */
export const StaffOverviewSection = () => {
  const { t } = useTranslation();
  const { token } = antdTheme.useToken();
  const { data, isFetching, error } = useStaffOverview(true);

  if (error) {
    return (
      <Alert
        type="error"
        showIcon
        style={{ marginBottom: 16 }}
        message={errorMessage(error, t('common.somethingWrong'))}
      />
    );
  }
  if (isFetching || !data) {
    return <Skeleton active paragraph={{ rows: 4 }} style={{ marginBottom: 16 }} />;
  }

  return (
    <>
      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={8}>
          <Card style={{ boxShadow: token.boxShadowTertiary }}>
            <Statistic
              title={t('dashboard.counterCountToday')}
              value={formatNumber(data.counter_sales_count)}
              prefix={<ShopOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card style={{ boxShadow: token.boxShadowTertiary }}>
            <Statistic
              title={t('dashboard.counterToday')}
              value={formatVND(data.counter_sales_total)}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card style={{ boxShadow: token.boxShadowTertiary }}>
            <Statistic
              title={t('dashboard.awaitingCheckin')}
              value={formatNumber(data.awaiting_checkin)}
              prefix={<SolutionOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Title level={5} style={{ marginTop: 8, marginBottom: 12 }}>
        {t('dashboard.staffToday')}
      </Title>
      {data.showtimes.length === 0 ? (
        <Empty description={t('common.noData')} />
      ) : (
        <Table
          rowKey="id"
          size="small"
          pagination={false}
          dataSource={data.showtimes}
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

export default StaffOverviewSection;
