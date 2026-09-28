import { useState } from 'react';
import { Alert, App, Button, Card, Col, Empty, Flex, Input, Pagination, Row, Space } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import HallCard from './components/HallCard';
import CloneHallModal from './components/CloneHallModal';
import HallSeatPanel from './HallSeatPanel';
import { useCreateHall, useDeleteHall, useHallList, useUpdateHall } from './hooks/useHalls';
import { QUICK_CREATE_SEATS_PER_ROW } from './constants';
import type { Hall } from '@/types';
import { useListQuery } from '@/hooks/useListQuery';
import { errorMessage } from '@/utils/error';

export const HallsPage = () => {
  const { t } = useTranslation();
  const { message, modal } = App.useApp();

  const { query, page, pageSize, search, setPage, setSearch } = useListQuery();
  const { data, isFetching, error } = useHallList(query);

  const halls = data?.items ?? [];

  const [cloningHall, setCloningHall] = useState<Hall | null>(null);

  const [selectedHallId, setSelectedHallId] = useState<string | null>(null);
  // The panel warns about unsaved edits; switching keys drops inputs, so ask before switching cards.
  const [panelDirty, setPanelDirty] = useState(false);

  // Preselect the first hall once; paging/search never jumps halls.
  const [autoSelected, setAutoSelected] = useState(false);
  if (!autoSelected && data && halls.length > 0) {
    setAutoSelected(true);
    if (selectedHallId === null) setSelectedHallId(halls[0].id);
  }

  const createHall = useCreateHall();
  const updateHall = useUpdateHall();
  const deleteHall = useDeleteHall();

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
        // Key change remounts the panel, so the child's cancel never needs calling.
        onOk: doSelect,
      });
      return;
    }
    doSelect();
  };

  // "+" creates an empty hall, inactive until finished.
  const handleQuickCreate = async () => {
    try {
      const created = await createHall.mutateAsync({
        name: `${t('hall.newHallDefaultName')} ${Date.now().toString().slice(-5)}`,
        rows: 1,
        seats_per_row: QUICK_CREATE_SEATS_PER_ROW,
        screen_position: 'front',
        aisle_after_cols: [],
        active: false,
      });
      message.success(t('common.createSuccess'));
      setSelectedHallId(created.id);
    } catch (err) {
      message.error(errorMessage(err, t('common.somethingWrong')));
    }
  };

  // Single click needs no confirm: the backend already blocks halls with live showtimes.
  const handleToggleActive = (hall: Hall) => {
    updateHall.mutate(
      { id: hall.id, payload: { active: !hall.active } },
      { onError: (err) => message.error(errorMessage(err, t('common.somethingWrong'))) }
    );
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteHall.mutateAsync(id);
      message.success(t('common.deleteSuccess'));
      if (selectedHallId === id) setSelectedHallId(null);
    } catch (err) {
      // Keep the specific backend sentence over the generic one.
      message.error(errorMessage(err, t('common.somethingWrong')));
    }
  };

  return (
    <>
      <Space wrap style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          defaultValue={search}
          placeholder={t('hall.searchPlaceholder')}
          style={{ width: 280 }}
          onSearch={setSearch}
        />
        <Button
          type="primary"
          icon={<PlusOutlined />}
          loading={createHall.isPending}
          onClick={() => void handleQuickCreate()}
        >
          {t('common.create')}
        </Button>
      </Space>

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
                selected={selectedHallId === hall.id}
                onSelect={() => selectHall(hall.id)}
                onClone={() => setCloningHall(hall)}
                onToggleActive={() => handleToggleActive(hall)}
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
