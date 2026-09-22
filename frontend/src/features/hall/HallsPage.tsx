import { useState } from 'react';
import { Alert, App, Button, Card, Col, Empty, Flex, Input, Pagination, Row, Space } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import HallCard from './components/HallCard';
import HallFormModal from './components/HallFormModal';
import HallPriceModal from './components/HallPriceModal';
import CloneHallModal from './components/CloneHallModal';
import HallSeatPanel from './HallSeatPanel';
import {
  useCreateHall,
  useDeleteHall,
  useHallList,
  useHallPriceStatuses,
  useUpdateHall,
} from './hooks/useHalls';
import type { Hall, HallPayload, UpdateHallPayload } from '@/types';
import { useListQuery } from '@/hooks/useListQuery';
import { errorMessage } from '@/utils/error';

/** Hall cards + inline seat grid on one page: clicking a card expands that hall's grid below, no child route. Single cinema (no chain), so the hall picker IS the main screen. */
export const HallsPage = () => {
  const { t } = useTranslation();
  const { message, modal } = App.useApp();

  const { query, page, pageSize, search, setPage, setSearch } = useListQuery();
  const { data, isFetching, error } = useHallList(query);

  const halls = data?.items ?? [];
  const priceStatuses = useHallPriceStatuses(halls);

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Hall | null>(null);
  const [pricingHall, setPricingHall] = useState<Hall | null>(null);
  const [cloningHall, setCloningHall] = useState<Hall | null>(null);

  const [selectedHallId, setSelectedHallId] = useState<string | null>(null);
  // Reported by HallSeatPanel: unsaved edits exist; confirm before switching cards (a `key` change discards input, see selectHall).
  const [panelDirty, setPanelDirty] = useState(false);

  // Preselect the first hall once. One-shot (autoSelected guard) so paging/search never jumps halls.
  const [autoSelected, setAutoSelected] = useState(false);
  if (!autoSelected && data && halls.length > 0) {
    setAutoSelected(true);
    // Keep a just-created selection; don't override it here.
    if (selectedHallId === null) setSelectedHallId(halls[0].id);
  }

  const createHall = useCreateHall();
  const updateHall = useUpdateHall();
  const deleteHall = useDeleteHall();

  const openCreate = () => {
    setEditing(null);
    setFormOpen(true);
  };

  const openEdit = (hall: Hall) => {
    setEditing(hall);
    setFormOpen(true);
  };

  const selectHall = (id: string) => {
    if (id === selectedHallId) return;
    const doSelect = () => setSelectedHallId(id);
    if (panelDirty) {
      modal.confirm({
        title: t('hall.discardTitle'),
        content: t('hall.discardBody'),
        okText: t('hall.discardConfirm'),
        okButtonProps: { danger: true },
        cancelText: t('hall.stayEditing'),
        // Switching `key` remounts HallSeatPanel, resetting its local state; no child cancel call needed.
        onOk: doSelect,
      });
      return;
    }
    doSelect();
  };

  // Don't catch here: the modal needs the raw error for input binding.
  const handleCreate = async (payload: HallPayload) => {
    const created = await createHall.mutateAsync(payload);
    message.success(t('common.createSuccess'));
    setFormOpen(false);
    // New card selected directly; its seats are unclassified, so seat arranging is almost surely next.
    setSelectedHallId(created.id);
  };

  const handleUpdate = async (payload: UpdateHallPayload) => {
    if (!editing) return;
    await updateHall.mutateAsync({ id: editing.id, payload });
    message.success(t('common.updateSuccess'));
    setFormOpen(false);
    setEditing(null);
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteHall.mutateAsync(id);
      message.success(t('common.deleteSuccess'));
      if (selectedHallId === id) setSelectedHallId(null);
    } catch (err) {
      // Backend message is English but more specific than any canned sentence here.
      message.error(errorMessage(err, t('common.somethingWrong')));
    }
  };

  return (
    <>
      <PageHeader
        title={t('hall.title')}
        extra={
          <Space>
            <Input.Search
              allowClear
              defaultValue={search}
              placeholder={t('hall.searchPlaceholder')}
              style={{ width: 280 }}
              onSearch={setSearch}
            />
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              {t('common.create')}
            </Button>
          </Space>
        }
      />

      {error ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={errorMessage(error, t('common.somethingWrong'))}
        />
      ) : null}

      {!isFetching && halls.length === 0 ? (
        <Empty description={t('common.noData')} />
      ) : (
        <Row gutter={[16, 16]}>
          {halls.map((hall) => (
            <Col key={hall.id} xs={24} sm={12} lg={8}>
              <HallCard
                hall={hall}
                priceRows={priceStatuses.byHallId[hall.id]}
                selected={selectedHallId === hall.id}
                onSelect={() => selectHall(hall.id)}
                onEdit={() => openEdit(hall)}
                onPrice={() => setPricingHall(hall)}
                onClone={() => setCloningHall(hall)}
                onDelete={() => handleDelete(hall.id)}
              />
            </Col>
          ))}
        </Row>
      )}

      {(data?.meta.total_pages ?? 0) > 1 ? (
        <Flex justify="center" style={{ marginTop: 16 }}>
          <Pagination
            current={data?.meta.page ?? page}
            pageSize={data?.meta.page_size ?? pageSize}
            total={data?.meta.total ?? 0}
            showSizeChanger
            showTotal={(total) => t('common.totalItems', { total })}
            onChange={setPage}
          />
        </Flex>
      ) : null}

      {selectedHallId ? (
        <Card size="small" style={{ marginTop: 20 }}>
          <HallSeatPanel
            key={selectedHallId}
            hallId={selectedHallId}
            onDirtyChange={setPanelDirty}
          />
        </Card>
      ) : null}

      <HallFormModal
        open={formOpen}
        hall={editing}
        confirmLoading={createHall.isPending || updateHall.isPending}
        onCancel={() => {
          setFormOpen(false);
          setEditing(null);
        }}
        onCreate={handleCreate}
        onUpdate={handleUpdate}
      />

      <HallPriceModal
        open={pricingHall !== null}
        hall={pricingHall}
        onCancel={() => setPricingHall(null)}
        onDone={() => setPricingHall(null)}
      />

      <CloneHallModal
        open={cloningHall !== null}
        hall={cloningHall}
        onCancel={() => setCloningHall(null)}
        onDone={() => setCloningHall(null)}
      />
    </>
  );
};

export default HallsPage;
