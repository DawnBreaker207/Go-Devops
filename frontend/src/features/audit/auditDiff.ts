import i18n from '@/locales/i18n';
import { formatDateTime, formatVND } from '@/utils/format';

export interface AuditDiffRow {
  key: string;
  label: string;
  before: string;
  after: string;
}

// Vietnamese labels for the ~30 keys the backend writes in before/after; unknown keys get a humanized key name.
const FIELD_LABELS: Record<string, string> = {
  status: 'Trạng thái',
  active: 'Hoạt động',
  price: 'Giá',
  prices: 'Giá',
  base_price: 'Giá',
  total_amount: 'Tổng tiền',
  discount: 'Số tiền giảm',
  payable: 'Số tiền phải trả',
  seat_type: 'Loại ghế',
  is_gap: 'Lối đi',
  code: 'Mã',
  deleted: 'Đã xóa',
  adjust_kind: 'Loại điều chỉnh',
  adjust_value: 'Giá trị điều chỉnh',
  day_of_week: 'Ngày trong tuần',
  start_time: 'Giờ bắt đầu',
  end_time: 'Giờ kết thúc',
  starts_at: 'Bắt đầu lúc',
  ends_at: 'Kết thúc lúc',
  specific_date: 'Ngày cụ thể',
  priority: 'Độ ưu tiên',
  standard: 'Giá ghế thường',
  vip: 'Giá ghế VIP',
  couple: 'Giá ghế đôi',
  recliner: 'Giá ghế nằm',
  left: 'Ghế trái',
  right: 'Ghế phải',
  col_span: 'Số cột chiếm',
  role: 'Vai trò',
  name: 'Tên',
  email: 'Email',
  phone: 'Số điện thoại',
  quantity: 'Số lượng',
  amount: 'Số tiền',
  resource_type: 'Loại đối tượng',
};

// Hide noisy id/secret/timestamps, keep only business times like starts/ends.
const HIDDEN_SUBSTRINGS = ['password', 'token', 'secret', 'signature'];
// A business time stays; other bookkeeping _at fields hide.
const ALLOWED_AT_KEYS = new Set(['starts_at', 'ends_at', 'start_time', 'end_time']);

const isHiddenKey = (key: string): boolean => {
  const lower = key.toLowerCase();
  if (key === 'id') return true;
  if (HIDDEN_SUBSTRINGS.some((s) => lower.includes(s))) return true;
  if (lower.endsWith('_at') && !ALLOWED_AT_KEYS.has(key)) return true;
  return false;
};

const MONEY_KEY_RE = /price|amount|discount|payable|total/i;
// ISO formatting goes through formatDateTime.
const ISO_DATE_RE = /^\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}(:\d{2})?(\.\d+)?(Z|[+-]\d{2}:?\d{2})?)?$/;

const truncate = (value: string, max = 70): string =>
  value.length > max ? `${value.slice(0, max)}…` : value;

const safeStringify = (value: unknown): string => {
  try {
    return JSON.stringify(value) ?? String(value);
  } catch {
    return '[unserializable]';
  }
};

const humanize = (key: string): string =>
  key
    .split(/[_.]/)
    .filter(Boolean)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');

const labelFor = (key: string): string => {
  const fallback = FIELD_LABELS[key] ?? humanize(key);
  try {
    return i18n.t(`audit.field_${key}`, { defaultValue: fallback });
  } catch {
    return fallback;
  }
};

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value);

// Objects with name/code/title collapse to that label instead of expanding.
const isReferenceObject = (obj: Record<string, unknown>): boolean =>
  typeof obj.name === 'string' || typeof obj.code === 'string' || typeof obj.title === 'string';

const formatValue = (key: string, value: unknown, parentKey?: string): string => {
  try {
    if (value === undefined || value === null) return '—';
    if (typeof value === 'boolean') return value ? 'Có' : 'Không';
    if (typeof value === 'number') {
      // A money-reading parent (e.g. old prices) money-formats every numeric sub-key too.
      const isMoney =
        MONEY_KEY_RE.test(key) || (parentKey !== undefined && MONEY_KEY_RE.test(parentKey));
      if (isMoney && Number.isFinite(value)) return formatVND(value);
      return String(value);
    }
    if (typeof value === 'string') {
      if (ISO_DATE_RE.test(value)) return formatDateTime(value);
      return truncate(value);
    }
    if (Array.isArray(value)) return `${value.length} mục`;
    if (isPlainObject(value)) {
      for (const k of ['name', 'code', 'title'] as const) {
        const v = value[k];
        if (typeof v === 'string') return truncate(v);
      }
      return truncate(safeStringify(value));
    }
    return truncate(String(value));
  } catch {
    return '—';
  }
};

// Parse strings, coerce to {} to tolerate old/junk rows, never throw.
const toRecord = (value: unknown): Record<string, unknown> => {
  try {
    if (value == null) return {};
    if (typeof value === 'string') {
      const parsed: unknown = JSON.parse(value);
      return isPlainObject(parsed) ? parsed : {};
    }
    if (isPlainObject(value)) return value;
    return {};
  } catch {
    return {};
  }
};

// Key-order-independent deep compare, never throws.
const canonicalEqual = (a: unknown, b: unknown): boolean => {
  const canon = (v: unknown): unknown => {
    if (Array.isArray(v)) return v.map(canon);
    if (isPlainObject(v)) {
      const out: Record<string, unknown> = {};
      for (const k of Object.keys(v).sort()) out[k] = canon(v[k]);
      return out;
    }
    return v;
  };
  try {
    return JSON.stringify(canon(a)) === JSON.stringify(canon(b));
  } catch {
    return a === b;
  }
};

// Diff before/after: expand plain objects by sub-key, collapse reference objects; never throws, it reads old/new DB rows directly.
export const diffAuditRows = (before?: unknown, after?: unknown): AuditDiffRow[] => {
  try {
    const beforeObj = toRecord(before);
    const afterObj = toRecord(after);
    const keys = Array.from(new Set([...Object.keys(beforeObj), ...Object.keys(afterObj)]));

    const rows: AuditDiffRow[] = [];

    for (const key of keys) {
      if (isHiddenKey(key)) continue;

      const bv = beforeObj[key];
      const av = afterObj[key];
      const sample = bv !== undefined ? bv : av;

      if (isPlainObject(sample) && !isReferenceObject(sample)) {
        const bvObj = isPlainObject(bv) ? bv : {};
        const avObj = isPlainObject(av) ? av : {};
        const subKeys = Array.from(new Set([...Object.keys(bvObj), ...Object.keys(avObj)]));
        for (const subKey of subKeys) {
          if (isHiddenKey(subKey)) continue;
          const subBefore = bvObj[subKey];
          const subAfter = avObj[subKey];
          if (canonicalEqual(subBefore, subAfter)) continue;
          rows.push({
            key: `${key}.${subKey}`,
            label: labelFor(subKey),
            before: formatValue(subKey, subBefore, key),
            after: formatValue(subKey, subAfter, key),
          });
        }
        continue;
      }

      if (canonicalEqual(bv, av)) continue;
      rows.push({
        key,
        label: labelFor(key),
        before: formatValue(key, bv),
        after: formatValue(key, av),
      });
    }

    return rows;
  } catch {
    return [];
  }
};
