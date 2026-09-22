import { Button, Select, Typography, theme as antdTheme } from 'antd';
import { useTranslation } from 'react-i18next';
import type { SeatType } from '@/types';
import { SEAT_TYPES } from '@/types';

interface SelectionPillProps {
  count: number;
  disabled: boolean;
  onSetSeatType: (type: SeatType) => void;
  onMarkGap: () => void;
  onMarkSeat: () => void;
  onClear: () => void;
}

/** Bottom control bar for the current seat selection. */
export const SelectionPill = ({
  count,
  disabled,
  onSetSeatType,
  onMarkGap,
  onMarkSeat,
  onClear,
}: SelectionPillProps) => {
  const { t } = useTranslation();
  const { token } = antdTheme.useToken();

  const visible = count > 0;

  return (
    <div
      role="toolbar"
      aria-hidden={!visible}
      style={{
        position: 'fixed',
        insetInline: 0,
        bottom: 0,
        zIndex: 20,
        display: 'flex',
        flexWrap: 'wrap',
        alignItems: 'center',
        gap: 12,
        padding: '12px 20px',
        background: token.colorBgElevated,
        borderTop: `1px solid ${token.colorBorderSecondary}`,
        boxShadow: token.boxShadowSecondary,
        transform: visible ? 'translateY(0)' : 'translateY(100%)',
        opacity: visible ? 1 : 0,
        pointerEvents: visible ? 'auto' : 'none',
        transition:
          'transform var(--motion-duration-base) var(--motion-ease-out), ' +
          'opacity var(--motion-duration-base) var(--motion-ease-out)',
      }}
    >
      <Typography.Text strong className="tabular-nums">
        {t('hall.selectedCount', { count })}
      </Typography.Text>

      <Select<SeatType>
        style={{ width: 200 }}
        placeholder={t('hall.setSeatType')}
        value={null}
        disabled={disabled}
        onChange={onSetSeatType}
        options={SEAT_TYPES.map((type) => ({ value: type, label: t(`hall.seatType_${type}`) }))}
      />

      {/* Two COMMAND buttons, not state toggles (a Segmented would light up the
          first item and mislead). */}
      <Button disabled={disabled} onClick={onMarkGap}>
        {t('hall.markAsGap')}
      </Button>
      <Button disabled={disabled} onClick={onMarkSeat}>
        {t('hall.markAsSeat')}
      </Button>

      <Button type="link" onClick={onClear} style={{ marginLeft: 'auto' }}>
        {t('hall.clearSelection')}
      </Button>
    </div>
  );
};

export default SelectionPill;
