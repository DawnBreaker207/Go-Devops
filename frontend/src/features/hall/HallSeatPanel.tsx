import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  App,
  Button,
  Descriptions,
  Dropdown,
  Flex,
  Space,
  Spin,
  Tag,
  Typography,
} from 'antd';
import {
  EditOutlined,
  EyeOutlined,
  MoreOutlined,
  ReloadOutlined,
  SaveOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import SeatGrid from './components/SeatGrid';
import SelectionPill from './components/SelectionPill';
import LayoutRegenerateModal from './components/LayoutRegenerateModal';
import { hallApi } from '@/api/hall.api';
import { useHall, useHallPrices, useHallSeats } from './hooks/useHalls';
import { MAX_ROWS, SEAT_TYPE_STYLE } from './constants';
import {
  areAdjacentSeats,
  buildPendingRow,
  buildSeatChangeBatches,
  diffChangedSeats,
  isPendingSeatId,
  pendingRowIndexOf,
  rowLabelFromIndex,
  summarizeGrid,
} from './seatGrid';
import type { Seat } from '@/types';
import { SEAT_TYPES } from '@/types';
import { errorMessage } from '@/utils/error';
import { formatNumber } from '@/utils/format';

interface HallSeatPanelProps {
  hallId: string;
  /** Let HallsPage know the panel is dirty, to confirm before switching halls. */
  onDirtyChange: (dirty: boolean) => void;
}

/** One hall's seat grid, expanded inline below the cards in HallsPage. Mounted with `key={hallId}`, so switching halls remounts and draft/editing reset by themselves. */
export const HallSeatPanel = ({ hallId, onDirtyChange }: HallSeatPanelProps) => {
  const { t } = useTranslation();
  const { message, modal } = App.useApp();

  const hall = useHall(hallId);
  const seatsQuery = useHallSeats(hallId);
  const prices = useHallPrices(hallId);

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [regenerateOpen, setRegenerateOpen] = useState(false);
  const [saving, setSaving] = useState(false);

  /** Single click no longer selects (`selected` is now row-pick for bulk via SelectionPill). Two transient states instead: `activeSeatId` (open quick-type popover) and `pendingMergeSeatId` (double-clicked, awaiting an adjacent second click). */
  const [activeSeatId, setActiveSeatId] = useState<string | null>(null);
  const [pendingMergeSeatId, setPendingMergeSeatId] = useState<string | null>(null);

  /** Edit mode is one session: every op touches only the local `draft`; `savedSnapshot` is the last server-confirmed copy for dirty-check and revert. */
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState<Seat[]>([]);
  const [savedSnapshot, setSavedSnapshot] = useState<Seat[]>([]);
  const [initialized, setInitialized] = useState(false);
  if (seatsQuery.data && !initialized) {
    setEditing(seatsQuery.data.length === 0);
    setDraft(seatsQuery.data);
    setSavedSnapshot(seatsQuery.data);
    setInitialized(true);
  }

  const changedGroups = useMemo(
    () => diffChangedSeats(savedSnapshot, draft),
    [savedSnapshot, draft]
  );
  const pendingRowCount = useMemo(
    () =>
      new Set(draft.filter((s) => isPendingSeatId(s.id)).map((s) => pendingRowIndexOf(s.id))).size,
    [draft]
  );
  const dirty = pendingRowCount > 0 || changedGroups.length > 0;

  useEffect(() => {
    onDirtyChange(dirty);
  }, [dirty, onDirtyChange]);

  // View mode has no selection/popover/pending-merge: clear on every path into it (state-during-render on `editing`, same as HallsPage first-hall preselect).
  const [editingAtLastSelectionClear, setEditingAtLastSelectionClear] = useState(editing);
  if (editingAtLastSelectionClear !== editing) {
    setEditingAtLastSelectionClear(editing);
    if (!editing) {
      setSelected(new Set());
      setActiveSeatId(null);
      setPendingMergeSeatId(null);
    }
  }

  // Leaving with unsaved changes asks via beforeunload (reload/close tab); in-app hall switches confirm separately in HallsPage via onDirtyChange.
  useEffect(() => {
    if (!dirty) return;
    const handler = (e: BeforeUnloadEvent) => e.preventDefault();
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
  }, [dirty]);

  // Escape closes the type popover and cancels pending merge.
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return;
      setActiveSeatId(null);
      setPendingMergeSeatId(null);
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

  const seatById = useMemo(() => new Map(draft.map((s) => [s.id, s])), [draft]);
  const summary = hall.data ? summarizeGrid(hall.data, draft) : null;

  // Gated on `editing`, not just a cleared `selected`: View never has a selection even if stale state survives (HMR, races).
  const selectedSeats: Seat[] = useMemo(
    () =>
      editing
        ? [...selected].map((sid) => seatById.get(sid)).filter((s): s is Seat => Boolean(s))
        : [],
    [editing, selected, seatById]
  );
  const activeSeat = editing && activeSeatId ? (seatById.get(activeSeatId) ?? null) : null;

  const toggleRow = (rowLabel: string) =>
    setSelected((current) => {
      const rowSeats = draft.filter((s) => s.row_label === rowLabel);
      const allSelected = rowSeats.every((s) => current.has(s.id));
      const next = new Set(current);
      rowSeats.forEach((s) => (allSelected ? next.delete(s.id) : next.add(s.id)));
      return next;
    });

  /** Apply one patch to the whole selection, draft only. */
  const applyPatchToSelection = (patch: Partial<Pick<Seat, 'seat_type' | 'is_gap'>>) => {
    if (selectedSeats.length === 0) return;
    const ids = new Set(selectedSeats.map((s) => s.id));
    setDraft((current) => current.map((s) => (ids.has(s.id) ? { ...s, ...patch } : s)));
    setSelected(new Set());
  };

  /** Gap seat: one click fills it as standard immediately, no pre-select. */
  const fillGap = (seat: Seat) => {
    setDraft((current) =>
      current.map((s) => (s.id === seat.id ? { ...s, is_gap: false, seat_type: 'standard' } : s))
    );
  };

  /** Click completes a pending merge (double-clicked seat): adjacent merges, non-adjacent (incl. re-click) cancels with a hint. Otherwise opens the quick-type popover for this one seat; re-click toggles it shut. */
  const handleSeatClick = (seat: Seat) => {
    if (pendingMergeSeatId) {
      const pending = seatById.get(pendingMergeSeatId);
      setPendingMergeSeatId(null);
      if (
        pending &&
        pending.id !== seat.id &&
        !isPendingSeatId(seat.id) &&
        areAdjacentSeats(pending, seat)
      ) {
        handleMergeCouple(pending, seat);
        return;
      }
      message.info(t('hall.mergeCoupleNotAdjacent'));
      return;
    }
    setActiveSeatId((current) => (current === seat.id ? null : seat.id));
  };

  /** Double-click enters pending-merge. Couples can't re-merge; unsaved "ghost" rows or other dirty edits must save first because merge/split OVERWRITES the draft from fresh server data (see runSeatAction). */
  const handleSeatDoubleClick = (seat: Seat) => {
    if (seat.col_span === 2) return;
    if (dirty || isPendingSeatId(seat.id)) {
      message.info(t('hall.saveBeforeMergeSplit'));
      return;
    }
    setActiveSeatId(null);
    setPendingMergeSeatId(seat.id);
  };

  /** Corner "x": marks a gap on the draft immediately, no confirm. */
  const handleQuickGap = (seat: Seat) => {
    setDraft((current) => current.map((s) => (s.id === seat.id ? { ...s, is_gap: true } : s)));
    setActiveSeatId((current) => (current === seat.id ? null : current));
    setPendingMergeSeatId((current) => (current === seat.id ? null : current));
  };

  const handleAddRow = () => {
    if (!hall.data) return;
    const rowLabel = rowLabelFromIndex(hall.data.rows + pendingRowCount + 1);
    const newRow = buildPendingRow(pendingRowCount, hall.data.seats_per_row, rowLabel, hallId);
    setDraft((current) => [...current, ...newRow]);
  };

  const discardDraft = () => {
    setDraft(savedSnapshot);
    setSelected(new Set());
    setActiveSeatId(null);
    setPendingMergeSeatId(null);
  };

  const confirmDiscard = (onDiscard: () => void) => {
    modal.confirm({
      title: t('hall.discardTitle'),
      content: t('hall.discardBody'),
      okText: t('hall.discardConfirm'),
      okButtonProps: { danger: true },
      cancelText: t('hall.stayEditing'),
      onOk: onDiscard,
    });
  };

  const handleToggleEditing = () => {
    if (editing && dirty) {
      confirmDiscard(() => {
        discardDraft();
        setEditing(false);
      });
      return;
    }
    setEditing((current) => !current);
    setSelected(new Set());
  };

  const handleDiscardClick = () => confirmDiscard(discardDraft);

  /** Save the whole session in one go: pending rows first, in added order (one AddRow call each; backend only appends "next row"). Draft is updated after each row so a mid-save failure never double-creates on retry. Type/gap edits on real seats (incl. just-created) are then batched into a single final PATCH. */
  const handleSave = async () => {
    if (!hall.data) return;
    setSaving(true);
    try {
      let working = draft;
      const pendingIndexes = [
        ...new Set(
          working.filter((s) => isPendingSeatId(s.id)).map((s) => pendingRowIndexOf(s.id))
        ),
      ].sort((a, b) => a - b);

      for (const rowIndex of pendingIndexes) {
        const localRow = working
          .filter((s) => isPendingSeatId(s.id) && pendingRowIndexOf(s.id) === rowIndex)
          .sort((a, b) => a.col_number - b.col_number);
        const created = await hallApi.addRow(hallId);
        const merged = created.map((real, i) => ({
          ...real,
          seat_type: localRow[i]?.seat_type ?? real.seat_type,
          is_gap: localRow[i]?.is_gap ?? real.is_gap,
        }));
        working = [
          ...working.filter(
            (s) => !(isPendingSeatId(s.id) && pendingRowIndexOf(s.id) === rowIndex)
          ),
          ...merged,
        ];
        setDraft(working);
      }

      // New-row seats aren't in savedSnapshot, so diffing would silently skip their overrides; seed a (standard, non-gap) default per new seat first.
      const savedIds = new Set(savedSnapshot.map((s) => s.id));
      const newSeatDefaults: Seat[] = working
        .filter((s) => !savedIds.has(s.id))
        .map((s) => ({ ...s, seat_type: 'standard', is_gap: false }));
      const patchGroups = diffChangedSeats([...savedSnapshot, ...newSeatDefaults], working);
      const batches = buildSeatChangeBatches(patchGroups);
      for (const changes of batches) {
        await hallApi.bulkUpdateSeats(hallId, { changes });
      }

      const [seatsResult] = await Promise.all([seatsQuery.refetch(), hall.refetch()]);
      const fresh = seatsResult.data ?? working;
      setDraft(fresh);
      setSavedSnapshot(fresh);
      setSelected(new Set());
      setActiveSeatId(null);
      setPendingMergeSeatId(null);
      message.success(t('hall.saveSuccess'));
    } catch (error) {
      // 409 "hall layout can not be changed when it has bookings" is the common case here. Keep the draft for retry; rows already created were merged above, so re-save won't duplicate.
      message.error(errorMessage(error, t('common.somethingWrong')));
      void hall.refetch();
    } finally {
      setSaving(false);
    }
  };

  /** Delete-row/merge/split act immediately on real seats, outside the draft session: confirm -> API -> refetch -> reload draft/snapshot from fresh server data, one flow for all three. */
  const runSeatAction = (
    confirm: { title: string; body: string },
    action: () => Promise<unknown>,
    successMessage: string
  ) => {
    modal.confirm({
      title: confirm.title,
      content: confirm.body,
      okText: t('common.confirm'),
      okButtonProps: { danger: true },
      cancelText: t('common.cancel'),
      onOk: async () => {
        try {
          await action();
          const [seatsResult] = await Promise.all([seatsQuery.refetch(), hall.refetch()]);
          if (seatsResult.data) {
            setDraft(seatsResult.data);
            setSavedSnapshot(seatsResult.data);
          }
          setSelected(new Set());
          setActiveSeatId(null);
          setPendingMergeSeatId(null);
          message.success(successMessage);
        } catch (error) {
          message.error(errorMessage(error, t('common.somethingWrong')));
        }
      },
    });
  };

  /** Deletes ANY row (backend renumbers followers). Pending (unsaved) rows drop from the draft with no API/confirm; saved rows are immediate + destructive, with confirm. */
  const handleDeleteRow = (rowLabel: string) => {
    const rowSeats = draft.filter((s) => s.row_label === rowLabel);
    if (rowSeats.length === 0) return;
    const isPending = rowSeats.every((s) => isPendingSeatId(s.id));

    if (isPending) {
      // Nothing persisted yet; drop silently, no confirm.
      setDraft((current) => current.filter((s) => s.row_label !== rowLabel));
      return;
    }
    runSeatAction(
      { title: t('hall.deleteRowConfirmTitle'), body: t('hall.deleteRowConfirmBody') },
      () => hallApi.deleteRow(hallId, rowLabel),
      t('hall.deleteRowSuccess', { row: rowLabel })
    );
  };

  const handleMergeCouple = (a: Seat, b: Seat) => {
    const [left, right] = a.col_number < b.col_number ? [a, b] : [b, a];
    runSeatAction(
      { title: t('hall.mergeCoupleConfirmTitle'), body: t('hall.mergeCoupleConfirmBody') },
      () => hallApi.mergeSeats(hallId, { left_label: left.label, right_label: right.label }),
      t('hall.mergeCoupleSuccess')
    );
  };

  const handleSplitCouple = () => {
    if (!activeSeat) return;
    setActiveSeatId(null);
    runSeatAction(
      { title: t('hall.splitCoupleConfirmTitle'), body: t('hall.splitCoupleConfirmBody') },
      () => hallApi.splitSeat(hallId, { label: activeSeat.label }),
      t('hall.splitCoupleSuccess')
    );
  };

  if (hall.isLoading || seatsQuery.isLoading) {
    return (
      <Flex justify="center" style={{ padding: 24 }}>
        <Spin />
      </Flex>
    );
  }

  if (hall.error || seatsQuery.error) {
    return (
      <Alert
        type="error"
        showIcon
        message={errorMessage(hall.error ?? seatsQuery.error, t('common.somethingWrong'))}
      />
    );
  }

  if (!hall.data) return null;

  return (
    <>
      <Flex justify="space-between" align="center" wrap style={{ marginBottom: 16 }} gap={8}>
        <Typography.Title level={5} style={{ margin: 0 }}>
          {t('hall.seatsTitle', { name: hall.data.name })}
        </Typography.Title>
        <Space wrap>
          {editing && dirty ? (
            <>
              <Tag color="warning">{t('hall.unsavedChanges')}</Tag>
              <Button onClick={handleDiscardClick} disabled={saving}>
                {t('hall.discardChanges')}
              </Button>
              <Button
                type="primary"
                icon={<SaveOutlined />}
                onClick={() => void handleSave()}
                loading={saving}
              >
                {t('hall.saveChanges')}
              </Button>
            </>
          ) : (
            <Button
              icon={editing ? <EyeOutlined /> : <EditOutlined />}
              onClick={handleToggleEditing}
            >
              {t(editing ? 'hall.viewMode' : 'hall.editMode')}
            </Button>
          )}
          <Dropdown
            menu={{
              items: [
                {
                  key: 'regenerate',
                  danger: true,
                  icon: <ReloadOutlined />,
                  label: t('hall.regenerate'),
                  disabled: dirty,
                  onClick: () => setRegenerateOpen(true),
                },
              ],
            }}
          >
            <Button icon={<MoreOutlined />} />
          </Dropdown>
        </Space>
      </Flex>

      <Descriptions size="small" column={{ xs: 1, sm: 2, lg: 4 }} style={{ marginBottom: 8 }}>
        <Descriptions.Item label={t('hall.grid')}>
          {hall.data.rows} × {hall.data.seats_per_row}
        </Descriptions.Item>
        <Descriptions.Item label={t('hall.screenPosition')}>
          {t(`hall.screen_${hall.data.screen_position}`)}
        </Descriptions.Item>
        <Descriptions.Item label={t('hall.aisles')}>
          {hall.data.aisle_after_cols.length > 0
            ? hall.data.aisle_after_cols.join(', ')
            : t('hall.noAisles')}
        </Descriptions.Item>
        <Descriptions.Item label={t('hall.active')}>
          <Tag color={hall.data.active ? 'green' : 'default'} bordered={false}>
            {t(hall.data.active ? 'hall.activeYes' : 'hall.activeNo')}
          </Tag>
        </Descriptions.Item>
      </Descriptions>

      {summary?.declaredMismatch ? (
        // Invariant, not input error: DB doesn't force halls.rows to match seats. Grid already renders from real seats, but the operator must be told, not left silent.
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 8 }}
          message={t('hall.gridMismatch', {
            declaredRows: hall.data.rows,
            declaredCols: hall.data.seats_per_row,
            actualRows: summary.actualRows,
            actualCols: summary.actualSeatsPerRow,
          })}
        />
      ) : null}

      {summary ? (
        // Cells vs seats legitimately differ: couples swallow a column, and sellable further excludes gaps.
        <Typography.Text
          type="secondary"
          style={{ fontSize: 12, display: 'block', marginBottom: 12 }}
        >
          {t('hall.gridSummary', {
            cells: formatNumber(summary.gridCells),
            seats: formatNumber(summary.seatCount),
            sellable: formatNumber(summary.sellable),
            gaps: formatNumber(summary.gaps),
            doubles: formatNumber(summary.doubleSeats),
          })}
        </Typography.Text>
      ) : null}

      <Flex wrap align="center" justify="space-between" gap={8} style={{ marginBottom: 8 }}>
        {editing ? (
          <Button
            size="small"
            type="link"
            style={{ paddingInline: 0 }}
            onClick={() => setSelected(new Set(draft.map((s) => s.id)))}
          >
            {t('hall.selectAll')}
          </Button>
        ) : (
          <span />
        )}
        <Flex wrap align="center" gap={8}>
          {SEAT_TYPES.map((type) => (
            <Tag
              key={type}
              bordered={false}
              style={{
                background: SEAT_TYPE_STYLE[type].bg,
                color: SEAT_TYPE_STYLE[type].fg,
                marginInlineEnd: 0,
              }}
            >
              {t(`hall.seatType_${type}`)}
            </Tag>
          ))}
        </Flex>
      </Flex>

      <Spin spinning={saving}>
        <SeatGrid
          hall={hall.data}
          seats={draft}
          selected={selected}
          onToggleRow={toggleRow}
          readOnly={!editing}
          onSeatClick={handleSeatClick}
          onSeatDoubleClick={handleSeatDoubleClick}
          activeSeatId={activeSeat?.id ?? null}
          pendingMergeSeatId={pendingMergeSeatId}
          seatPopoverContent={
            activeSeat ? (
              activeSeat.col_span === 2 ? (
                <Button
                  size="small"
                  block
                  disabled={dirty || isPendingSeatId(activeSeat.id)}
                  onClick={handleSplitCouple}
                >
                  {t('hall.splitCouple')}
                </Button>
              ) : (
                <Space wrap size={4}>
                  {SEAT_TYPES.map((type) => (
                    <Button
                      key={type}
                      size="small"
                      type={activeSeat.seat_type === type ? 'primary' : 'default'}
                      onClick={() => {
                        setDraft((current) =>
                          current.map((s) =>
                            s.id === activeSeat.id ? { ...s, seat_type: type } : s
                          )
                        );
                        setActiveSeatId(null);
                      }}
                    >
                      {t(`hall.seatType_${type}`)}
                    </Button>
                  ))}
                </Space>
              )
            ) : undefined
          }
          onQuickGap={handleQuickGap}
          onFillGap={fillGap}
          onAddRow={
            editing && hall.data.rows + pendingRowCount < MAX_ROWS ? handleAddRow : undefined
          }
          onDeleteRow={editing && draft.length > 0 ? handleDeleteRow : undefined}
        />
      </Spin>

      {/* Fixed bottom pill replaces the static top toolbar; see SelectionPill. Spacer below keeps it off the last seat row. Edit-only; per-seat popover is independent of row `selected`, so no display clash. */}
      <div style={{ height: editing && selectedSeats.length > 0 ? 76 : 0 }} />
      <SelectionPill
        count={editing ? selectedSeats.length : 0}
        disabled={saving}
        onSetSeatType={(type) => applyPatchToSelection({ seat_type: type })}
        onMarkGap={() => applyPatchToSelection({ is_gap: true })}
        onMarkSeat={() => applyPatchToSelection({ is_gap: false })}
        onClear={() => setSelected(new Set())}
      />

      <LayoutRegenerateModal
        open={regenerateOpen}
        hall={hall.data}
        prices={prices.data ?? []}
        onCancel={() => setRegenerateOpen(false)}
        onDone={() => {
          setRegenerateOpen(false);
          setSelected(new Set());
          setActiveSeatId(null);
          setPendingMergeSeatId(null);
          // Regenerate wipes and rebuilds the grid server-side; local draft/snapshot must reload from scratch, not follow queries.
          void seatsQuery.refetch().then((result) => {
            if (result.data) {
              setDraft(result.data);
              setSavedSnapshot(result.data);
            }
          });
        }}
      />
    </>
  );
};

export default HallSeatPanel;
