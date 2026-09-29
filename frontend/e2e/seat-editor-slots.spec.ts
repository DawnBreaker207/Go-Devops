import { expect, test } from '@playwright/test';

// A row gaining a seat in a new column must offer the other rows a virtual
// fill (+) slot there; clicking it mints a real pending seat. Covers the
// column-trim contract: kept columns keep their fill buttons.
const FE = 'http://localhost:3000';
const API = 'http://localhost:8080/api/v1';

test('hang co cot moi -> hang con lai hien +, bam ra ghe that', async ({ page, request }) => {
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
      name: `ZZ Slots ${Date.now()}`,
      rows: 2,
      seats_per_row: 4,
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

    // Row-end "+" of row A: one real pending seat A5 (gap cells have no
    // .seat-cell class, so the row-end buttons are matched by text).
    await page
      .getByRole('button', { name: /thêm ghế/i })
      .first()
      .click();

    // Row B offers a virtual fill (+) slot at B5.
    await expect(page.getByRole('button', { name: 'B5', exact: true })).toHaveCount(1);

    // Filling it mints a real pending B5 (a .seat-cell appears).
    await page.getByRole('button', { name: 'B5', exact: true }).click();
    const seats = await page.evaluate(
      () =>
        [...document.querySelectorAll('.seat-cell')].filter((c) =>
          ['A5', 'B5'].includes(
            c.querySelector('button[aria-label]')?.getAttribute('aria-label') ?? ''
          )
        ).length
    );
    expect(seats).toBe(2);
  } finally {
    await request.delete(`${API}/admin/halls/${hall.id}`, { headers });
  }
});
