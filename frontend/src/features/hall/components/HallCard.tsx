import {
  Button,
  Card,
  Popconfirm,
  Space,
  Tag,
  Tooltip,
  Typography,
  theme as antdTheme,
} from 'antd';
import { CopyOutlined, DeleteOutlined, DollarOutlined, EditOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { Hall, HallPrice } from '@/types';
import { isPriceSetComplete } from '../hooks/useHalls';

interface HallCardProps {
  hall: Hall;
  /** undefined = prices still loading. */
  priceRows: HallPrice[] | undefined;
  selected: boolean;
  onSelect: () => void;
  onEdit: () => void;
  onPrice: () => void;
  onClone: () => void;
  onDelete: () => void;
}

/** Hall card selecting on click with actions isolated from selection. */
export const HallCard = ({
  hall,
  priceRows,
  selected,
  onSelect,
  onEdit,
  onPrice,
  onClone,
  onDelete,
}: HallCardProps) => {
  const { t } = useTranslation();
  const { token } = antdTheme.useToken();

  return (
    <Card
      hoverable
      size="small"
      onClick={onSelect}
      style={{
        cursor: 'pointer',
        borderColor: selected ? token.colorPrimary : undefined,
        background: selected ? token.colorPrimaryBg : undefined,
      }}
      styles={{ body: { padding: 16 } }}
    >
      <Space direction="vertical" size={4} style={{ width: '100%' }}>
        <Space align="start" style={{ width: '100%', justifyContent: 'space-between' }}>
          <div style={{ minWidth: 0, flex: 1 }}>
            <Typography.Text strong ellipsis style={{ display: 'block' }}>
              {hall.name}
            </Typography.Text>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {hall.rows} × {hall.seats_per_row}
            </Typography.Text>
          </div>
          <Space size={0} onClick={(e) => e.stopPropagation()}>
            <Tooltip title={t('hall.prices')}>
              <Button type="text" size="small" icon={<DollarOutlined />} onClick={onPrice} />
            </Tooltip>
            <Tooltip title={t('hall.clone')}>
              <Button type="text" size="small" icon={<CopyOutlined />} onClick={onClone} />
            </Tooltip>
            <Tooltip title={t('common.edit')}>
              <Button type="text" size="small" icon={<EditOutlined />} onClick={onEdit} />
            </Tooltip>
            <Popconfirm
              title={t('hall.deleteConfirm')}
              okText={t('common.confirm')}
              cancelText={t('common.cancel')}
              onConfirm={onDelete}
            >
              <Button danger type="text" size="small" icon={<DeleteOutlined />} />
            </Popconfirm>
          </Space>
        </Space>

        <Space wrap size={4}>
          {priceRows === undefined ? (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              …
            </Typography.Text>
          ) : isPriceSetComplete(priceRows) ? (
            <Tag color="green" bordered={false}>
              {t('hall.pricesComplete')}
            </Tag>
          ) : (
            <Tooltip title={t('hall.pricesIncompleteTooltip')}>
              <Tag color="warning" bordered={false}>
                {t('hall.pricesIncomplete', { count: priceRows.length })}
              </Tag>
            </Tooltip>
          )}
          <Tag color={hall.active ? 'green' : 'default'} bordered={false}>
            {t(hall.active ? 'hall.activeYes' : 'hall.activeNo')}
          </Tag>
        </Space>
      </Space>
    </Card>
  );
};

export default HallCard;
