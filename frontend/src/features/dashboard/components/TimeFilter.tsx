import { DatePicker, Segmented, Select } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import { useTranslation } from 'react-i18next';
import type { DashboardPreset, FilterSelection } from './dashboardRange';

const QUARTERS = [1, 2, 3, 4];

/** Period type + anchor pick (any date for week/month/year; explicit Q+year). */
export const TimeFilter = ({
  value,
  onChange,
}: {
  value: FilterSelection;
  onChange: (next: FilterSelection) => void;
}) => {
  const { t } = useTranslation();
  const now = new Date();
  const thisYear = now.getFullYear();
  const years = [thisYear - 5, thisYear - 4, thisYear - 3, thisYear - 2, thisYear - 1, thisYear];
  const anchorDate =
    value.type === 'quarter' ? new Date(value.year, (value.quarter - 1) * 3, 1) : value.date;

  const pick = (date: Dayjs | null) => {
    if (!date || !date.isValid()) return;
    onChange({ type: value.type, date: date.toDate() } as FilterSelection);
  };

  return (
    <div
      style={{ display: 'flex', gap: 12, flexWrap: 'wrap', alignItems: 'center', marginBottom: 16 }}
    >
      <Segmented
        value={value.type}
        onChange={(v) => {
          const type = v as DashboardPreset;
          onChange(
            type === 'quarter'
              ? {
                  type,
                  quarter: Math.floor(anchorDate.getMonth() / 3) + 1,
                  year: anchorDate.getFullYear(),
                }
              : { type, date: anchorDate }
          );
        }}
        options={[
          { value: 'week', label: t('dashboard.presetWeek') },
          { value: 'month', label: t('dashboard.presetMonth') },
          { value: 'quarter', label: t('dashboard.presetQuarter') },
          { value: 'year', label: t('dashboard.presetYear') },
        ]}
      />
      {value.type === 'quarter' ? (
        <>
          <Select
            value={value.quarter}
            onChange={(q) => onChange({ type: 'quarter', quarter: q, year: value.year })}
            options={QUARTERS.map((q) => ({ value: q, label: `Q${q}` }))}
            style={{ width: 90 }}
            aria-label={t('dashboard.quarterLabel')}
          />
          <Select
            value={value.year}
            onChange={(y) => onChange({ type: 'quarter', quarter: value.quarter, year: y })}
            options={years.map((y) => ({ value: y, label: y }))}
            style={{ width: 100 }}
            aria-label={t('dashboard.yearLabel')}
          />
        </>
      ) : (
        <DatePicker
          picker={value.type}
          value={dayjs(anchorDate)}
          onChange={pick}
          allowClear={false}
          format={
            value.type === 'week' ? 'DD/MM/YYYY' : value.type === 'month' ? 'MM/YYYY' : 'YYYY'
          }
        />
      )}
    </div>
  );
};

export default TimeFilter;
