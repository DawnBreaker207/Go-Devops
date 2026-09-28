import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, App, Button, Dropdown, Flex, Input, Space, Spin, Tag, Typography } from 'antd';
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
import { useHall, useHallSeats, useUpdateHall } from './hooks/useHalls';
import { MAX_ROWS, MAX_SEATS_PER_ROW, SEAT_TYPE_STYLE } from './constants';
import {
  areAdjacentSeats,
  buildPendingColumn,
  buildPendingRow,
  buildSeatChangeBatches,
  countSeatsByType,
  declaredGridMismatch,
  diffChangedSeats,
  diffMergeSplitOps,
  groupSeatsByRow,
  isPendingColSeatId,
  isPendingSeatId,
  isUnsavedSeatId,
  makeSplitSeatId,
  parseAisles,
  pendingColIndexOf,
  pendingRowIndexOf,
  rowLabelFromIndex,
  summarizeGrid,
  widestColumn,
} from './seatGrid';
import type { Seat, SeatType } from '@/types';
import { SEAT_TYPES } from '@/types';
import { errorMessage } from '@/utils/error';
import { formatNumber } from '@/utils/format';

/** Click-cycle order; couple excluded since that's a merge, not a type change. */
const SEAT_TYPE_CYCLE: SeatType[] = SEAT_TYPES.filter((type) => type !== 'couple');

/** Undo window for cycle-on-double-click: browsers always fire click before dblclick. */
const SEAT_CLICK_CYCLE_DELAY_MS = 250;

interface HallSeatPanelProps {
  hallId: string;
  onDirtyChange: (dirty: boolean) => void;
}

/** One seat grid per hall, mounted with key={hallId} so switching halls remounts and resets the draft. */
export const HallSeatPanel = ({ hallId, onDirtyChange }: HallSeatPanelProps) => {
  const { t } = useTranslation();
  const { message, modal } = App.useApp();

  const hall = useHall(hallId);
  const seatsQuery = useHallSeats(hallId);
  /** Hall metadata saves immediately, separate from the seat draft session. */
  const updateHall = useUpdateHall();

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [regenerateOpen, setRegenerateOpen] = useState(false);
  /** Custom editor instead of antd editable, for a live preview right on the grid. */
  const [aislesEditing, setAislesEditing] = useState(false);
  const [aislesDraft, setAislesDraft] = useState('');
  const [saving, setSaving] = useState(false);

  const [pendingMergeSeatId, setPendingMergeSeatId] = useState<string | null>(null);

  /** Pre-cycle type per seat for double-click undo; other seats never clear it. */
  const cycleRevertRef = useRef<{
    seatId: string;
    originalType: SeatType;
    timer: ReturnType<typeof setTimeout>;
  } | null>(null);
  const clearCycleRevert = () => {
    if (cycleRevertRef.current) {
      clearTimeout(cycleRevertRef.current.timer);
      cycleRevertRef.current = null;
    }
  };
  useEffect(() => clearCycleRevert, []);

  /** One edit session: every op touches the draft only; savedSnapshot diffs dirty and reverts. */
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
  const pendingColCount = useMemo(
    () =>
      new Set(draft.filter((s) => isPendingColSeatId(s.id)).map((s) => pendingColIndexOf(s.id)))
        .size,
    [draft]
  );
  const dirty = pendingRowCount > 0 || pendingColCount > 0 || changedGroups.length > 0;

  useEffect(() => {
    onDirtyChange(dirty);
  }, [dirty, onDirtyChange]);

  // View mode keeps no selection; reset on every view entry via state-during-render.
  const [editingAtLastSelectionClear, setEditingAtLastSelectionClear] = useState(editing);
  if (editingAtLastSelectionClear !== editing) {
    setEditingAtLastSelectionClear(editing);
    if (!editing) {
      setSelected(new Set());
      setPendingMergeSeatId(null);
    }
  }

  // Never clear refs during render (unsafe); dedicated effect instead.
  useEffect(() => {
    if (!editing) clearCycleRevert();
  }, [editing]);

  // Keep beforeunload for F5/closing tabs; in-app hall switches ask via HallsPage instead.
  useEffect(() => {
    if (!dirty) return;
    const handler = (e: BeforeUnloadEvent) => e.preventDefault();
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
  }, [dirty]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return;
      if (cycleRevertRef.current) {
        const { seatId, originalType } = cycleRevertRef.current;
        clearCycleRevert();
        setDraft((current) =>
          current.map((s) => (s.id === seatId ? { ...s, seat_type: originalType } : s))
        );
      }
      setPendingMergeSeatId(null);
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

  const seatById = useMemo(() => new Map(draft.map((s) => [s.id, s])), [draft]);
  const summary = hall.data ? summarizeGrid(hall.data, draft) : null;
  /** Compared against savedSnapshot: pending rows intentionally overshoot rows. */
  const mismatch = hall.data ? declaredGridMismatch(hall.data, savedSnapshot) : null;
  const typeCounts = useMemo(() => countSeatsByType(draft), [draft]);

  // Selection exists only while editing, so no stale state survives HMR or races.
  const selectedSeats: Seat[] = useMemo(
    () =>
      editing
        ? [...selected].map((sid) => seatById.get(sid)).filter((s): s is Seat => Boolean(s))
        : [],
    [editing, selected, seatById]
  );
  const toggleRow = (rowLabel: string) =>
    setSelected((current) => {
      const rowSeats = draft.filter((s) => s.row_label === rowLabel);
      const allSelected = rowSeats.every((s) => current.has(s.id));
      const next = new Set(current);
      rowSeats.forEach((s) => (allSelected ? next.delete(s.id) : next.add(s.id)));
      return next;
    });

  const handleToggleSelectAll = (selectAll: boolean) =>
    setSelected(selectAll ? new Set(draft.map((s) => s.id)) : new Set());

  const applyPatchToSelection = (patch: Partial<Pick<Seat, 'seat_type' | 'is_gap'>>) => {
    if (selectedSeats.length === 0) return;
    const ids = new Set(selectedSeats.map((s) => s.id));
    setDraft((current) => current.map((s) => (ids.has(s.id) ? { ...s, ...patch } : s)));
    setSelected(new Set());
  };

  const fillGap = (seat: Seat) => {
    setDraft((current) =>
      current.map((s) => (s.id === seat.id ? { ...s, is_gap: false, seat_type: 'standard' } : s))
    );
  };

  /** Merge/split mutate the draft, so no clean grid is needed first - same as AddRow. */
  const handleSeatClick = (seat: Seat) => {
    if (pendingMergeSeatId) {
      const pending = seatById.get(pendingMergeSeatId);
      setPendingMergeSeatId(null);
      if (pending && pending.id !== seat.id && areAdjacentSeats(pending, seat)) {
        handleMergeCouple(pending, seat);
        return;
      }
      message.info(t('hall.mergeCoupleNotAdjacent'));
      return;
    }
    if (seat.col_span === 2) {
      handleSplitCouple(seat);
      return;
    }
    // Reuse the saved originalType: this seat already cycled once.
    const originalType =
      cycleRevertRef.current?.seatId === seat.id
        ? cycleRevertRef.current.originalType
        : seat.seat_type;
    clearCycleRevert();
    const currentIndex = SEAT_TYPE_CYCLE.indexOf(seat.seat_type);
    const nextType = SEAT_TYPE_CYCLE[(currentIndex + 1) % SEAT_TYPE_CYCLE.length];
    setDraft((current) =>
      current.map((s) => (s.id === seat.id ? { ...s, seat_type: nextType } : s))
    );
    cycleRevertRef.current = {
      seatId: seat.id,
      originalType,
      timer: setTimeout(clearCycleRevert, SEAT_CLICK_CYCLE_DELAY_MS),
    };
  };

  /** Undo the cycle that two click events caused before pending-merge; never marks dirty. */
  const handleSeatDoubleClick = (seat: Seat) => {
    if (cycleRevertRef.current?.seatId === seat.id) {
      const { originalType } = cycleRevertRef.current;
      clearCycleRevert();
      setDraft((current) =>
        current.map((s) => (s.id === seat.id ? { ...s, seat_type: originalType } : s))
      );
    }
    if (seat.col_span === 2) return;
    setPendingMergeSeatId(seat.id);
  };

  const handleQuickGap = (seat: Seat) => {
    clearCycleRevert();
    setDraft((current) => current.map((s) => (s.id === seat.id ? { ...s, is_gap: true } : s)));
    setPendingMergeSeatId((current) => (current === seat.id ? null : current));
  };

  const handleAddRow = () => {
    if (!hall.data) return;
    const rowLabel = rowLabelFromIndex(hall.data.rows + pendingRowCount + 1);
    // Draw gaps for pending columns already in the draft instead of leaving holes.
    const totalWidth = Math.max(widestColumn(draft), hall.data.seats_per_row);
    const newRow = buildPendingRow(
      pendingRowCount,
      hall.data.seats_per_row,
      rowLabel,
      hallId,
      totalWidth
    );
    setDraft((current) => [...current, ...newRow]);
  };

  /** Adding a column stays draft until Save, like AddRow but along the other axis. */
  const handleAddColumn = () => {
    if (!hall.data) return;
    const rowLabels = groupSeatsByRow(draft).map((row) => row.rowLabel);
    const nextCol = Math.max(widestColumn(draft), hall.data.seats_per_row) + 1;
    const newCol = buildPendingColumn(pendingColCount, rowLabels, nextCol, hallId);
    setDraft((current) => [...current, ...newCol]);
  };

  const discardDraft = () => {
    clearCycleRevert();
    setDraft(savedSnapshot);
    setSelected(new Set());
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

  /** Save in fixed order: rows, then columns, then merge/split, finally type/gap patches. */
  const handleSave = async () => {
    if (!hall.data) return;
    clearCycleRevert();
    setSaving(true);
    try {
      let working = draft;
      // Creation sources disagree on defaults, so each seat needs its own baseline.
      let newSeatDefaults: Seat[] = [];

      const pendingRowIndexes = [
        ...new Set(
          working.filter((s) => isPendingSeatId(s.id)).map((s) => pendingRowIndexOf(s.id))
        ),
      ].sort((a, b) => a - b);

      for (const rowIndex of pendingRowIndexes) {
        const localRow = working.filter(
          (s) => isPendingSeatId(s.id) && pendingRowIndexOf(s.id) === rowIndex
        );
        const created = await hallApi.addRow(hallId);
        // Match by col_number: a local merge can drop a column.
        const localByCol = new Map(localRow.map((s) => [s.col_number, s]));
        const merged = created.map((real) => {
          const local = localByCol.get(real.col_number);
          return {
            ...real,
            seat_type: local?.seat_type ?? real.seat_type,
            is_gap: local?.is_gap ?? real.is_gap,
            col_span: local?.col_span ?? real.col_span,
          };
        });
        working = [
          ...working.filter(
            (s) => !(isPendingSeatId(s.id) && pendingRowIndexOf(s.id) === rowIndex)
          ),
          ...merged,
        ];
        newSeatDefaults = [...newSeatDefaults, ...created];
        setDraft(working);
      }

      const pendingColIndexes = [
        ...new Set(
          working.filter((s) => isPendingColSeatId(s.id)).map((s) => pendingColIndexOf(s.id))
        ),
      ].sort((a, b) => a - b);

      for (const colIndex of pendingColIndexes) {
        const localCol = working.filter(
          (s) => isPendingColSeatId(s.id) && pendingColIndexOf(s.id) === colIndex
        );
        const byRow = new Map(localCol.map((s) => [s.row_label, s]));
        const created = await hallApi.addColumn(hallId);
        const merged = created.map((real) => {
          const local = byRow.get(real.row_label);
          return {
            ...real,
            seat_type: local?.seat_type ?? real.seat_type,
            is_gap: local?.is_gap ?? real.is_gap,
            col_span: local?.col_span ?? real.col_span,
          };
        });
        working = [
          ...working.filter(
            (s) => !(isPendingColSeatId(s.id) && pendingColIndexOf(s.id) === colIndex)
          ),
          ...merged,
        ];
        newSeatDefaults = [...newSeatDefaults, ...created];
        setDraft(working);
      }

      // Session merges/splits are draft-only, so reconcile against the server by label here.
      const { merges, splits } = diffMergeSplitOps([...savedSnapshot, ...newSeatDefaults], working);
      // Drop finished merges/splits from the shared patch to avoid resending couple types.
      const mergeSplitSeatIds = new Set<string>();

      for (const mergeOp of merges) {
        const updatedLeft = await hallApi.mergeSeats(hallId, {
          left_label: mergeOp.leftLabel,
          right_label: mergeOp.rightLabel,
        });
        mergeSplitSeatIds.add(updatedLeft.id);
        working = [
          ...working.filter((s) => s.label !== mergeOp.leftLabel && s.label !== mergeOp.rightLabel),
          updatedLeft,
        ];
        setDraft(working);
      }

      for (const splitOp of splits) {
        const updated = await hallApi.splitSeat(hallId, { label: splitOp.label });
        updated.forEach((s) => mergeSplitSeatIds.add(s.id));
        const updatedLabels = new Set(updated.map((s) => s.label));
        working = [...working.filter((s) => !updatedLabels.has(s.label)), ...updated];
        setDraft(working);
      }

      const patchGroups = diffChangedSeats(
        [...savedSnapshot, ...newSeatDefaults],
        working.filter((s) => !mergeSplitSeatIds.has(s.id))
      );
      const batches = buildSeatChangeBatches(patchGroups);
      for (const changes of batches) {
        await hallApi.bulkUpdateSeats(hallId, { changes });
      }

      const [seatsResult] = await Promise.all([seatsQuery.refetch(), hall.refetch()]);
      const fresh = seatsResult.data ?? working;
      setDraft(fresh);
      setSavedSnapshot(fresh);
      setSelected(new Set());
      setPendingMergeSeatId(null);
      message.success(t('hall.saveSuccess'));
    } catch (error) {
      // The common failure is 409 on halls with bookings; keep the draft to retry without re-adding rows.
      message.error(errorMessage(error, t('common.somethingWrong')));
      void hall.refetch();
    } finally {
      setSaving(false);
    }
  };

  /** Row delete hits the server immediately, outside the draft session unlike local merge/split. */
  const performSeatAction = async (action: () => Promise<unknown>, successMessage: string) => {
    try {
      clearCycleRevert();
      await action();
      const [seatsResult] = await Promise.all([seatsQuery.refetch(), hall.refetch()]);
      if (seatsResult.data) {
        setDraft(seatsResult.data);
        setSavedSnapshot(seatsResult.data);
      }
      setSelected(new Set());
      setPendingMergeSeatId(null);
      message.success(successMessage);
    } catch (error) {
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  /** Row delete is hard to revert, so keep the confirm; local merge/split never takes this path. */
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
      onOk: () => performSeatAction(action, successMessage),
    });
  };

  /** The backend renumbers rows after a delete; unsaved rows just drop from the draft. */
  const handleDeleteRow = (rowLabel: string) => {
    const rowSeats = draft.filter((s) => s.row_label === rowLabel);
    if (rowSeats.length === 0) return;
    const isPending = rowSeats.every((s) => isUnsavedSeatId(s.id));

    if (isPending) {
      setDraft((current) => current.filter((s) => s.row_label !== rowLabel));
      return;
    }
    runSeatAction(
      { title: t('hall.deleteRowConfirmTitle'), body: t('hall.deleteRowConfirmBody') },
      () => hallApi.deleteRow(hallId, rowLabel),
      t('hall.deleteRowSuccess', { row: rowLabel })
    );
  };

  /** Merge needs no confirm: double-click plus an adjacent click is deliberate enough; defer to Save. */
  const handleMergeCouple = (a: Seat, b: Seat) => {
    const [left, right] = a.col_number < b.col_number ? [a, b] : [b, a];
    setDraft((current) =>
      current
        .filter((s) => s.id !== right.id)
        .map((s) =>
          s.id === left.id ? { ...s, col_span: 2 as const, seat_type: 'couple' as SeatType } : s
        )
    );
    message.success(t('hall.mergeCoupleSuccess'));
  };

  /** Split is easy to fat-finger, so keep a light confirm; still draft until Save hits the server. */
  const handleSplitCouple = (seat: Seat) => {
    modal.confirm({
      title: t('hall.splitCoupleConfirmTitle'),
      content: t('hall.splitCoupleConfirmBody'),
      okText: t('common.confirm'),
      okButtonProps: { danger: true },
      cancelText: t('common.cancel'),
      mask: false,
      width: 360,
      onOk: () => {
        const rightSeat: Seat = {
          id: makeSplitSeatId(seat.id),
          hall_id: hallId,
          label: `${seat.row_label}${seat.col_number + 1}`,
          row_label: seat.row_label,
          col_number: seat.col_number + 1,
          seat_type: 'standard',
          is_gap: false,
          col_span: 1,
        };
        setDraft((current) => [
          ...current.map((s) =>
            s.id === seat.id ? { ...s, col_span: 1 as const, seat_type: 'standard' as SeatType } : s
          ),
          rightSeat,
        ]);
        message.success(t('hall.splitCoupleSuccess'));
      },
    });
  };

  /** Hall metadata saves straight through PUT, independent of the seat draft session. */
  const handleRename = (name: string) => {
    const trimmed = name.trim();
    if (!trimmed || trimmed === hall.data?.name) return;
    updateHall.mutate(
      { id: hallId, payload: { name: trimmed } },
      { onError: (error) => message.error(errorMessage(error, t('common.somethingWrong'))) }
    );
  };

  const startEditingAisles = () => {
    setAislesDraft(hall.data ? hall.data.aisle_after_cols.join(', ') : '');
    setAislesEditing(true);
  };

  const cancelEditingAisles = () => setAislesEditing(false);

  const confirmAisles = () => {
    const current = hall.data;
    setAislesEditing(false);
    if (!current) return;
    const cols = parseAisles(aislesDraft);
    const unchanged =
      cols.length === current.aisle_after_cols.length &&
      cols.every((c, i) => c === current.aisle_after_cols[i]);
    if (unchanged) return;
    updateHall.mutate(
      { id: hallId, payload: { aisle_after_cols: cols } },
      { onError: (error) => message.error(errorMessage(error, t('common.somethingWrong'))) }
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
        <Space size={6} align="baseline" style={{ minWidth: 0 }}>
          <Typography.Text type="secondary">{t('hall.seatsTitlePrefix')}</Typography.Text>
          <Typography.Title
            level={5}
            style={{ margin: 0 }}
            editable={{ onChange: handleRename, tooltip: t('common.edit') }}
          >
            {hall.data.name}
          </Typography.Title>
        </Space>
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
                  key: 'aisles',
                  icon: <EditOutlined />,
                  label: t('hall.aisles'),
                  onClick: startEditingAisles,
                },
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

      {/* Active toggle nam tren HallCard nen o day chi hien trang thai. */}
      <Flex align="center" style={{ marginBottom: 8 }}>
        <Tag color={hall.data.active ? 'green' : 'default'} bordered={false}>
          {t(hall.data.active ? 'hall.activeYes' : 'hall.activeNo')}
        </Tag>
      </Flex>

      {aislesEditing ? (
        <Flex align="center" gap={8} style={{ marginBottom: 8 }}>
          <Typography.Text type="secondary">{t('hall.aisles')}</Typography.Text>
          <Input
            size="small"
            autoFocus
            value={aislesDraft}
            placeholder="4, 10"
            style={{ width: 140 }}
            onChange={(e) => setAislesDraft(e.target.value)}
            onPressEnter={confirmAisles}
            onBlur={confirmAisles}
            onKeyDown={(e) => {
              if (e.key === 'Escape') cancelEditingAisles();
            }}
          />
        </Flex>
      ) : null}

      {mismatch ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 8 }}
          message={t('hall.gridMismatch', {
            declaredRows: hall.data.rows,
            declaredCols: hall.data.seats_per_row,
            actualRows: mismatch.actualRows,
            actualCols: mismatch.actualSeatsPerRow,
          })}
        />
      ) : null}

      {summary ? (
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

      <Flex wrap align="center" gap={8} style={{ marginBottom: 8 }}>
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
            {t(`hall.seatType_${type}`)} · {typeCounts[type]}
          </Tag>
        ))}
      </Flex>

      <Spin spinning={saving}>
        <SeatGrid
          hall={hall.data}
          seats={draft}
          selected={selected}
          onToggleRow={toggleRow}
          onToggleSelectAll={handleToggleSelectAll}
          readOnly={!editing}
          onSeatClick={handleSeatClick}
          onSeatDoubleClick={handleSeatDoubleClick}
          pendingMergeSeatId={pendingMergeSeatId}
          onQuickGap={handleQuickGap}
          onFillGap={fillGap}
          onAddRow={
            editing && hall.data.rows + pendingRowCount < MAX_ROWS ? handleAddRow : undefined
          }
          onDeleteRow={editing && draft.length > 0 ? handleDeleteRow : undefined}
          onAddColumn={
            editing && Math.max(widestColumn(draft), hall.data.seats_per_row) < MAX_SEATS_PER_ROW
              ? handleAddColumn
              : undefined
          }
          aisleAfterColsOverride={aislesEditing ? parseAisles(aislesDraft) : undefined}
        />
      </Spin>

      {/* Giu khoang trong de pill duoi khong che hang ghe cuoi. */}
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
        onCancel={() => setRegenerateOpen(false)}
        onDone={() => {
          setRegenerateOpen(false);
          setSelected(new Set());
          setPendingMergeSeatId(null);
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
