import { describe, expect, it } from 'vitest';
import { diffAuditRows } from '../auditDiff';

describe('diffAuditRows', () => {
  it('bung prices thanh 4 dong con, dinh dang VND', () => {
    const rows = diffAuditRows(undefined, {
      prices: { standard: 60000, vip: 80000, couple: 120000, recliner: 150000 },
    });
    expect(rows).toHaveLength(4);
    const standard = rows.find((r) => r.key === 'prices.standard');
    expect(standard?.before).toBe('—');
    expect(standard?.after).toContain('₫');
  });

  it('khoa user: active true -> false hien thi Co -> Khong', () => {
    const rows = diffAuditRows({ active: true }, { active: false });
    expect(rows).toEqual([
      { key: 'active', label: expect.any(String), before: 'Có', after: 'Không' },
    ]);
  });

  it('ap ma giam gia: payable la so tien dinh dang VND', () => {
    const rows = diffAuditRows(undefined, { code: 'SALE10', discount: 10000, payable: 90000 });
    const payable = rows.find((r) => r.key === 'payable');
    expect(payable?.after).toBe('90.000 ₫');
    const discount = rows.find((r) => r.key === 'discount');
    expect(discount?.after).toBe('10.000 ₫');
  });

  it('gop ghe: before {left,right}, after {seat_type,col_span}', () => {
    const rows = diffAuditRows({ left: 'A1', right: 'A2' }, { seat_type: 'couple', col_span: 2 });
    const keys = rows.map((r) => r.key);
    expect(keys).toEqual(expect.arrayContaining(['left', 'right', 'seat_type', 'col_span']));
    const left = rows.find((r) => r.key === 'left');
    expect(left?.after).toBe('—');
  });

  it('xoa: after {deleted:true} tra ve dung 1 dong', () => {
    const rows = diffAuditRows(undefined, { deleted: true });
    expect(rows).toHaveLength(1);
    expect(rows[0]).toMatchObject({ key: 'deleted', after: 'Có' });
  });

  it('that bai voi before/after rong: 0 dong, khong throw', () => {
    expect(() => diffAuditRows({}, {})).not.toThrow();
    expect(diffAuditRows({}, {})).toHaveLength(0);
    expect(diffAuditRows(undefined, undefined)).toHaveLength(0);
  });

  it('key la/rac van khong throw va co nhan fallback humanize', () => {
    expect(() => diffAuditRows(null, 'not json{{{')).not.toThrow();
    const rows = diffAuditRows({ some_weird_key: 1 }, { some_weird_key: 2 });
    expect(rows[0].label).toBe('Some Weird Key');
  });

  it('nested object co truong name: lay .name thay vi JSON', () => {
    const rows = diffAuditRows(
      { movie: { id: 'm1', name: 'Old title' } },
      { movie: { id: 'm1', name: 'New title' } }
    );
    expect(rows).toHaveLength(1);
    expect(rows[0].before).toBe('Old title');
    expect(rows[0].after).toBe('New title');
  });

  it('gia tri ISO date dinh dang kieu Viet DD/MM/YYYY HH:mm', () => {
    const rows = diffAuditRows(
      { starts_at: '2026-01-01T10:00:00Z' },
      { starts_at: '2026-01-02T12:30:00Z' }
    );
    expect(rows).toHaveLength(1);
    expect(rows[0].before).toMatch(/^\d{2}\/\d{2}\/2026 \d{2}:\d{2}$/);
  });

  it('an key nhay cam (password) du hai phia khac nhau', () => {
    const rows = diffAuditRows({ password: 'old-secret' }, { password: 'new-secret' });
    expect(rows).toHaveLength(0);
  });

  it('an generic *_at (updated_at) nhung giu starts_at/ends_at', () => {
    const rows = diffAuditRows(
      { updated_at: '2026-01-01T00:00:00Z', ends_at: '2026-01-01T00:00:00Z' },
      { updated_at: '2026-01-02T00:00:00Z', ends_at: '2026-01-02T00:00:00Z' }
    );
    const keys = rows.map((r) => r.key);
    expect(keys).not.toContain('updated_at');
    expect(keys).toContain('ends_at');
  });

  it('an id va token', () => {
    const rows = diffAuditRows({ id: 'a', token: 'x' }, { id: 'b', token: 'y' });
    expect(rows).toHaveLength(0);
  });

  it('bo qua dong khong thay doi (so canonical, khong quan tam thu tu key)', () => {
    const rows = diffAuditRows({ a: { x: 1, y: 2 } }, { a: { y: 2, x: 1 } });
    expect(rows).toHaveLength(0);
  });

  it('chuoi bi cat ngan qua 70 ky tu', () => {
    const long = 'a'.repeat(100);
    const rows = diffAuditRows(undefined, { note: long });
    expect(rows[0].after.length).toBeLessThanOrEqual(71);
    expect(rows[0].after.endsWith('…')).toBe(true);
  });

  it('mang gia tri hien thi dang "N muc"', () => {
    const rows = diffAuditRows(undefined, { seats: ['A1', 'A2', 'A3'] });
    expect(rows[0].after).toBe('3 mục');
  });

  it('chuoi JSON tho (before_json la string) van parse duoc', () => {
    const rows = diffAuditRows('{"status":"active"}', '{"status":"locked"}');
    expect(rows).toEqual([
      { key: 'status', label: expect.any(String), before: 'active', after: 'locked' },
    ]);
  });
});
