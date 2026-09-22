import { Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import type { DailyBreakdownShowtime } from '@/types';
import { occupancyPercent } from '../reportRollup';
import { formatDateTime, formatNumber, formatVND } from '@/utils/format';

interface ShowtimeBreakdownTableProps {
  showtimes: DailyBreakdownShowtime[];
}

/** Per-showtime rows of one day using close-out snapshots. */
export const ShowtimeBreakdownTable = ({ showtimes }: ShowtimeBreakdownTableProps) => {
  const { t } = useTranslation();

  const columns: ColumnsType<DailyBreakdownShowtime> = [
    {
      title: t('report.startAt'),
      dataIndex: 'start_at',
      key: 'start_at',
      width: 160,
      render: (value: string) => formatDateTime(value),
    },
    { title: t('report.movie'), dataIndex: 'movie', key: 'movie', ellipsis: true },
    { title: t('report.hall'), dataIndex: 'hall', key: 'hall', width: 170, ellipsis: true },
    {
      title: t('report.seats'),
      key: 'seats',
      width: 130,
      align: 'right',
      render: (_, row) => (
        <span className="tabular-nums">
          {formatNumber(row.seats_sold)} / {formatNumber(row.capacity)}
        </span>
      ),
    },
    {
      title: t('report.occupancy'),
      key: 'occupancy',
      width: 100,
      align: 'right',
      render: (_, row) => (
        <span className="tabular-nums">{occupancyPercent(row.seats_sold, row.capacity)}%</span>
      ),
    },
    {
      title: t('report.checkedIn'),
      dataIndex: 'checked_in',
      key: 'checked_in',
      width: 110,
      align: 'right',
      render: (value: number) => <span className="tabular-nums">{formatNumber(value)}</span>,
    },
    {
      title: t('report.revenue'),
      dataIndex: 'revenue',
      key: 'revenue',
      width: 150,
      align: 'right',
      render: (value: number) => <span className="tabular-nums">{formatVND(value)}</span>,
    },
  ];

  return (
    <Table<DailyBreakdownShowtime>
      rowKey="showtime_id"
      size="small"
      columns={columns}
      dataSource={showtimes}
      pagination={false}
      scroll={{ x: 900 }}
    />
  );
};

export default ShowtimeBreakdownTable;
