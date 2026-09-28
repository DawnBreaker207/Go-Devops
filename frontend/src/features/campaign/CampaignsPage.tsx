import { useState } from 'react';
import {
  Alert,
  App,
  Button,
  Input,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CheckCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  LinkOutlined,
  PlusOutlined,
  StopOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import TableCard from '@/components/TableCard';
import CampaignFormModal from './components/CampaignFormModal';
import CampaignLinksModal from './components/CampaignLinksModal';
import {
  useCampaignList,
  useCreateCampaign,
  useDeleteCampaign,
  useUpdateCampaign,
} from './hooks/useCampaigns';
import type { Campaign, CreateCampaignPayload, UpdateCampaignPayload } from '@/types';
import { useListQuery } from '@/hooks/useListQuery';
import { errorMessage } from '@/utils/error';
import { formatDateTime } from '@/utils/format';

/** Backend takes a Go pointer, so "no filter" must OMIT the field, not send false. */
type ActiveFilter = 'all' | 'active' | 'inactive';

/** Holiday campaigns (PLAN_CAMPAIGN.md): a name + window bundling a discount
 *  code, a seasonal combo and a promotion article. ADMIN-ONLY, like discounts
 *  and pricing: this moves revenue. */
export const CampaignsPage = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  const { query, page, pageSize, search, setPage, setSearch } = useListQuery();
  const [activeFilter, setActiveFilter] = useState<ActiveFilter>('all');
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Campaign | null>(null);
  const [linksFor, setLinksFor] = useState<string | null>(null);
  const [pendingId, setPendingId] = useState<string | null>(null);

  const { data, isFetching, error } = useCampaignList({
    ...query,
    active: activeFilter === 'all' ? undefined : activeFilter === 'active',
  });

  const createCampaign = useCreateCampaign();
  const updateCampaign = useUpdateCampaign();
  const deleteCampaign = useDeleteCampaign();

  // Don't catch here: the modal needs the raw error to bind it to the inputs.
  const handleSubmit = async (payload: CreateCampaignPayload | UpdateCampaignPayload) => {
    if (editing) {
      await updateCampaign.mutateAsync({ id: editing.id, payload });
      message.success(t('common.updateSuccess'));
    } else {
      await createCampaign.mutateAsync(payload as CreateCampaignPayload);
      message.success(t('campaign.createSuccess'));
    }
    setFormOpen(false);
    setEditing(null);
  };

  const toggleActive = async (row: Campaign, next: boolean) => {
    setPendingId(row.id);
    try {
      await updateCampaign.mutateAsync({ id: row.id, payload: { active: next } });
      message.success(t('common.updateSuccess'));
    } catch (err) {
      message.error(errorMessage(err, t('common.somethingWrong')));
    } finally {
      setPendingId(null);
    }
  };

  const handleDelete = async (row: Campaign) => {
    setPendingId(row.id);
    try {
      await deleteCampaign.mutateAsync(row.id);
      message.success(t('campaign.deleteSuccess'));
    } catch (err) {
      message.error(errorMessage(err, t('common.somethingWrong')));
    } finally {
      setPendingId(null);
    }
  };

  const columns: ColumnsType<Campaign> = [
    {
      title: t('campaign.name'),
      key: 'name',
      ellipsis: true,
      render: (_, record) => (
        <Space direction="vertical" size={0}>
          <Typography.Text strong>{record.name}</Typography.Text>
          {record.description ? (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {record.description}
            </Typography.Text>
          ) : null}
        </Space>
      ),
    },
    {
      title: t('campaign.window'),
      key: 'window',
      width: 220,
      render: (_, record) => (
        <Typography.Text style={{ fontSize: 12 }}>
          {`${formatDateTime(record.starts_at)} → ${formatDateTime(record.ends_at)}`}
        </Typography.Text>
      ),
    },
    {
      title: t('campaign.perUserLimit'),
      dataIndex: 'per_user_limit',
      key: 'per_user_limit',
      width: 100,
      align: 'right',
    },
    {
      title: t('campaign.status'),
      dataIndex: 'active',
      key: 'active',
      width: 120,
      render: (value: boolean) => (
        <Tag color={value ? 'green' : 'default'} bordered={false}>
          {t(value ? 'campaign.enabled' : 'campaign.disabled')}
        </Tag>
      ),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 200,
      render: (_, record) => (
        <Space size={4}>
          <Tooltip title={record.active ? t('campaign.markOff') : t('campaign.markOn')}>
            <Popconfirm
              title={record.active ? t('campaign.offConfirm') : t('campaign.onConfirm')}
              okText={t('common.confirm')}
              cancelText={t('common.cancel')}
              onConfirm={() => void toggleActive(record, !record.active)}
            >
              <Button
                type="text"
                icon={record.active ? <StopOutlined /> : <CheckCircleOutlined />}
                aria-label={`toggle-active-${record.id}`}
                disabled={pendingId === record.id}
                loading={pendingId === record.id}
              />
            </Popconfirm>
          </Tooltip>
          <Tooltip title={t('campaign.linksTitle')}>
            <Button
              type="text"
              icon={<LinkOutlined />}
              aria-label={`links-${record.id}`}
              onClick={() => setLinksFor(record.id)}
            />
          </Tooltip>
          <Tooltip title={t('common.edit')}>
            <Button
              type="text"
              icon={<EditOutlined />}
              aria-label={`edit-${record.id}`}
              onClick={() => {
                setEditing(record);
                setFormOpen(true);
              }}
            />
          </Tooltip>
          <Tooltip title={t('common.delete')}>
            <Popconfirm
              title={t('campaign.deleteConfirm')}
              description={t('campaign.deleteHint')}
              okText={t('common.delete')}
              cancelText={t('common.cancel')}
              okButtonProps={{ danger: true }}
              onConfirm={() => void handleDelete(record)}
            >
              <Button
                danger
                type="text"
                icon={<DeleteOutlined />}
                aria-label={`delete-${record.id}`}
                disabled={pendingId === record.id}
              />
            </Popconfirm>
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <>
      <Space wrap style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          defaultValue={search}
          placeholder={t('campaign.searchPlaceholder')}
          style={{ width: 280 }}
          onSearch={setSearch}
        />
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setEditing(null);
            setFormOpen(true);
          }}
        >
          {t('common.create')}
        </Button>
        <Select<ActiveFilter>
          style={{ width: 200 }}
          value={activeFilter}
          onChange={(value) => {
            setActiveFilter(value);
            setPage(1);
          }}
          options={[
            { value: 'all', label: t('campaign.filterAll') },
            { value: 'active', label: t('campaign.enabled') },
            { value: 'inactive', label: t('campaign.disabled') },
          ]}
        />
      </Space>

      {/* Same reminder as discounts/pricing: a campaign bundles a real discount code. */}
      <Alert
        type="warning"
        showIcon
        closable
        style={{ marginBottom: 16 }}
        message={t('campaign.realMoneyHint')}
      />

      {error ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={errorMessage(error, t('common.somethingWrong'))}
        />
      ) : null}

      <TableCard>
        <Table<Campaign>
          rowKey="id"
          columns={columns}
          dataSource={data?.items ?? []}
          loading={isFetching}
          scroll={{ x: 900 }}
          pagination={{
            current: data?.meta.page ?? page,
            pageSize: data?.meta.page_size ?? pageSize,
            total: data?.meta.total ?? 0,
            showSizeChanger: true,
            showTotal: (total) => t('common.totalItems', { total }),
            onChange: (nextPage, nextSize) => setPage(nextPage, nextSize),
          }}
        />
      </TableCard>

      <CampaignFormModal
        open={formOpen}
        editing={editing}
        confirmLoading={createCampaign.isPending || updateCampaign.isPending}
        onCancel={() => {
          setFormOpen(false);
          setEditing(null);
        }}
        onSubmit={handleSubmit}
      />

      <CampaignLinksModal
        open={linksFor !== null}
        campaignId={linksFor}
        onCancel={() => setLinksFor(null)}
      />
    </>
  );
};

export default CampaignsPage;
