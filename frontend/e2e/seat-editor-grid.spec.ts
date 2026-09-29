import { expect, test } from '@playwright/test';

// Regression: a seat split off a couple is appended at the END of the draft
// array. With sparse grid auto-flow the browser refused to backtrack the row
// cursor and dropped it a row below (right X, wrong Y). SeatGrid pins
// gridRow: 1 on every cell so array order never decides the row.
const FE = 'http://localhost:3000';
const API = 'http://localhost:8080/api/v1';

test('tach ghe doi: ghe phai ve dung o, khong rot xuong dong duoi', async ({ page, request }) => {
  const login = await page.request.post(`${API}/auth/login`, {
    data: { email: 'admin@cinema.local', password: 'admin123' },
  });
  expect(login.ok()).toBeTruthy();
  const body = await login.json();
  await page.addInitScript(
    ({ access, refresh }) => {
      localStorage.setItem('cp_access_token', access);
      localStorage.setItem('cp_refresh_token', refresh);
    },
    { access: body.data.access_token, refresh: body.data.refresh_token }
  );
  const headers = { Authorization: `Bearer ${body.data.access_token as string}` };

  const created = await request.post(`${API}/admin/halls`, {
    headers,
    data: {
      name: `ZZ Grid ${Date.now()}`,
      rows: 2,
      seats_per_row: 6,
      screen_position: 'front',
      aisle_after_cols: [],
      active: false,
    },
  });
  expect(created.ok()).toBeTruthy();
  const hall = ((await created.json()).data ?? {}) as { id: string; name: string };

  try {
    await page.goto(`${FE}/catalog`);
    await page.getByRole('tab', { name: /phòng chiếu/i }).click();
    await page.getByText(hall.name, { exact: true }).first().click();
    await page.getByRole('button', { name: /sửa sơ đồ/i }).click();

    await page.getByRole('button', { name: 'A1', exact: true }).dblclick();
    await page.getByRole('button', { name: 'A2', exact: true }).click();
    await page.getByRole('button', { name: 'A1', exact: true }).click();
    await page.getByRole('button', { name: /xác nhận/i }).click();
    await page.waitForSelector('.ant-modal', { state: 'detached', timeout: 10000 });

    const pos = await page.evaluate(() => {
      const cell = (label: string) =>
        document.querySelector(`button[aria-label="${label}"]`)?.closest('.seat-cell');
      const r = (label: string) => cell(label)?.getBoundingClientRect();
      return { a1: r('A1'), a2: r('A2') };
    });
    expect(pos.a1).toBeTruthy();
    expect(pos.a2).toBeTruthy();
    // Same row (allow 1px sub-pixel drift), A2 exactly one cell to the right.
    expect(Math.abs((pos.a2?.top ?? 0) - (pos.a1?.top ?? 0))).toBeLessThanOrEqual(1);
    expect((pos.a2?.left ?? 0) - (pos.a1?.left ?? 0)).toBeGreaterThan(30);
  } finally {
    await request.delete(`${API}/admin/halls/${hall.id}`, { headers });
  }
});
