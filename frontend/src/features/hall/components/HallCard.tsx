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
import { CopyOutlined, DeleteOutlined, PoweroffOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { Hall } from '@/types';

interface HallCardProps {
  hall: Hall;
  selected: boolean;
  onSelect: () => void;
  onClone: () => void;
  onToggleActive: () => void;
  onDelete: () => void;
}

/** Hall card selecting on click, with actions isolated from selection. */
export const HallCard = ({
  hall,
  selected,
  onSelect,
  onClone,
  onToggleActive,
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
            <Tooltip title={t('hall.clone')}>
              <Button type="text" size="small" icon={<CopyOutlined />} onClick={onClone} />
            </Tooltip>
            <Tooltip title={t(hall.active ? 'hall.deactivate' : 'hall.activate')}>
              <Button
                type="text"
                size="small"
                danger={hall.active}
                icon={<PoweroffOutlined />}
                onClick={onToggleActive}
              />
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
          <Tag color={hall.active ? 'green' : 'default'} bordered={false}>
            {t(hall.active ? 'hall.activeYes' : 'hall.activeNo')}
          </Tag>
        </Space>
      </Space>
    </Card>
  );
};

export default HallCard;
