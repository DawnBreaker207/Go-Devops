import { expect, test, type Page } from '@playwright/test';

const FE = 'http://localhost:3000';
const API = 'http://localhost:8080/api/v1';
const EMAIL = 'customer@cinema.local';
const PASSWORD = 'customer123';

/** Real-API login, tokens into localStorage (fast, stable). */
async function seedAuth(page: Page): Promise<string> {
  const res = await page.request.post(`${API}/auth/login`, {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(res.ok()).toBeTruthy();
  const body = await res.json();
  await page.addInitScript(
    ({ access, refresh }) => {
      localStorage.setItem('cp_access_token', access);
      localStorage.setItem('cp_refresh_token', refresh);
    },
    { access: body.data.access_token, refresh: body.data.refresh_token }
  );
  return body.data.access_token as string;
}

/** One open showtime in the next 14 days (live backend, seed DB). */
async function findOpenShowtime(page: Page, token: string): Promise<string> {
  const headers = { Authorization: `Bearer ${token}` };
  const today = new Date();
  for (let i = 0; i < 14; i++) {
    const d = new Date(today);
    d.setDate(d.getDate() + i);
    const date = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
    const res = await page.request.get(`${API}/showtimes?date=${date}`, { headers });
    if (!res.ok()) continue;
    const body = await res.json();
    const items = body.data?.items ?? body.data ?? [];
    const open = items.find((s: { status?: string }) => (s.status ?? 'open') === 'open');
    if (open) return open.id as string;
  }
  throw new Error('no open showtime in 14 days');
}

/** Free seats from the live seatmap (each test gets its own pair). */
async function takeAvailableLabels(
  page: Page,
  token: string,
  showtimeId: string,
  count: number
): Promise<string[]> {
  const res = await page.request.get(`${API}/shows/${showtimeId}/seats`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(res.ok()).toBeTruthy();
  const body = await res.json();
  const seats = body.data?.seats ?? [];
  return seats
    .filter(
      (s: { status?: string; is_gap?: boolean; showtime_seat_id?: string }) =>
        s.status === 'available' && !s.is_gap && s.showtime_seat_id
    )
    .slice(0, count)
    .map((s: { label: string }) => s.label);
}

const sidebar = (page: Page) => page.locator('aside').first();

test.describe.serial('luong dat ve khach', () => {
  test('dang nhap UI bang tai khoan seed', async ({ page }) => {
    await page.goto(`${FE}/customer-login`);
    await page.locator('#cp-login-page-email').fill(EMAIL);
    await page.locator('#cp-login-page-password').fill(PASSWORD);
    await page.getByRole('button', { name: /đăng nhập/i }).click();
    await expect
      .poll(async () => page.evaluate(() => localStorage.getItem('cp_access_token')), {
        timeout: 15_000,
      })
      .not.toBeNull();
    await expect(page).not.toHaveURL(/customer-login/);
  });

  test('happy path: giu ghe -> combo (bo qua) -> pay mock -> success, sidebar xuyen suot', async ({
    page,
  }) => {
    const token = await seedAuth(page);
    const showtimeId = await findOpenShowtime(page, token);
    const labels = await takeAvailableLabels(page, token, showtimeId, 2);
    expect(labels.length).toBe(2);

    await page.goto(`${FE}/select-seat/${showtimeId}`);
    // Step 0: map + sidebar (no countdown pre-hold).
    await expect(sidebar(page)).toBeVisible();
    for (const label of labels) {
      await page.getByRole('button', { name: label, exact: true }).click();
    }
    // Sidebar shows the picked labels.
    await expect(sidebar(page).getByText(labels[0])).toBeVisible();
    // Countdown starts right at seat step (order created ~0.5s in).
    await expect(
      sidebar(page)
        .getByText(/\d{2}:\d{2}/)
        .first()
    ).toBeVisible({ timeout: 20_000 });
    // Continue = hold seats -> combo step (URL unchanged).
    await sidebar(page).getByRole('button').last().click();
    await expect(page.getByRole('heading', { name: /chọn combo/i })).toBeVisible({
      timeout: 20_000,
    });
    expect(page.url()).toContain('/select-seat/');
    // Sidebar persists with running countdown (pending order).
    await expect(sidebar(page)).toBeVisible();
    await expect(
      sidebar(page)
        .getByText(/\d{2}:\d{2}/)
        .first()
    ).toBeVisible();
    for (const label of labels) {
      await expect(sidebar(page).getByText(label, { exact: true })).toBeVisible();
    }

    // Skip combo = sidebar Continue with nothing picked.
    await sidebar(page).getByRole('button').last().click();
    await expect(page.getByRole('heading', { name: /phương thức thanh toán/i })).toBeVisible({
      timeout: 20_000,
    });

    // Sidebar pay -> SAME-TAB redirect to the mock gateway (no popup).
    await sidebar(page).getByRole('button').last().click();
    // Mock gateway checkout (BE): pay, gateway IPNs + 303s to
    // /payment-result, which verifies + confirms into success.
    await expect(page).toHaveURL(/checkout/, { timeout: 20_000 });
    await page.locator('button[value="pay"]').click();
    await expect(page).toHaveURL(/\/payment-result/, { timeout: 20_000 });
    // SEPARATE success page after confirm.
    await expect(page.getByRole('heading', { name: /thanh toán thành công/i })).toBeVisible({
      timeout: 30_000,
    });
    expect(page.url()).toContain('/booking-success/');
    // Success wipes the old run's store: a new run shows no stale data.
    const flowState = await page.evaluate(() => sessionStorage.getItem('cp-booking-flow'));
    expect(flowState == null || flowState.includes('"bookingId":null')).toBeTruthy();
    // "My tickets" link -> personal profile with the tickets tab open.
    await page.getByRole('link', { name: /vé của tôi/i }).click();
    await expect(page).toHaveURL(/\/account\?tab=tickets/);
    const accountTabs = page.getByRole('tablist').first();
    await expect(accountTabs.getByRole('tab', { selected: true })).toContainText(/vé của tôi/i);
  });

  test('stepper lui duoc giua cac buoc', async ({ page }) => {
    const token = await seedAuth(page);
    const showtimeId = await findOpenShowtime(page, token);
    const labels = await takeAvailableLabels(page, token, showtimeId, 2);
    await page.goto(`${FE}/select-seat/${showtimeId}`);
    for (const label of labels) {
      await page.getByRole('button', { name: label, exact: true }).click();
    }
    await sidebar(page).getByRole('button').last().click();
    await expect(page.getByRole('heading', { name: /chọn combo/i })).toBeVisible({
      timeout: 20_000,
    });

    // Back to seat step via stepper.
    await page.getByRole('button', { name: /vị trí ngồi/i }).click();
    await expect(page.getByRole('button', { name: labels[0], exact: true })).toBeVisible({
      timeout: 15_000,
    });
  });

  test('F5 giua checkout van o checkout + sidebar', async ({ page }) => {
    const token = await seedAuth(page);
    const showtimeId = await findOpenShowtime(page, token);
    const labels = await takeAvailableLabels(page, token, showtimeId, 2);
    await page.goto(`${FE}/select-seat/${showtimeId}`);
    for (const label of labels) {
      await page.getByRole('button', { name: label, exact: true }).click();
    }
    await sidebar(page).getByRole('button').last().click();
    await expect(page.getByRole('heading', { name: /chọn combo/i })).toBeVisible({
      timeout: 20_000,
    });
    await sidebar(page).getByRole('button').last().click();
    await expect(page.getByRole('heading', { name: /phương thức thanh toán/i })).toBeVisible({
      timeout: 20_000,
    });

    await page.reload();
    await expect(page.getByRole('heading', { name: /phương thức thanh toán/i })).toBeVisible({
      timeout: 20_000,
    });
    await expect(sidebar(page)).toBeVisible();
  });

  test('checkout khong con nut huy don (da don theo yeu cau)', async ({ page }) => {
    const token = await seedAuth(page);
    const showtimeId = await findOpenShowtime(page, token);
    const labels = await takeAvailableLabels(page, token, showtimeId, 2);
    await page.goto(`${FE}/select-seat/${showtimeId}`);
    for (const label of labels) {
      await page.getByRole('button', { name: label, exact: true }).click();
    }
    await sidebar(page).getByRole('button').last().click();
    await expect(page.getByRole('heading', { name: /chọn combo/i })).toBeVisible({
      timeout: 20_000,
    });
    await sidebar(page).getByRole('button').last().click();
    await expect(page.getByRole('heading', { name: /phương thức thanh toán/i })).toBeVisible({
      timeout: 20_000,
    });

    // Pay button lives in the sidebar; Cancel is gone - content has neither.
    await expect(sidebar(page).getByRole('button').last()).toBeVisible();
    await expect(page.getByRole('button', { name: /huỷ đơn|hủy đơn/i })).toHaveCount(0);
  });

  test('voucher mock: nhap ma -> hien giam -> pay van success du tien', async ({ page }) => {
    const token = await seedAuth(page);
    const showtimeId = await findOpenShowtime(page, token);
    const labels = await takeAvailableLabels(page, token, showtimeId, 2);
    await page.goto(`${FE}/select-seat/${showtimeId}`);
    for (const label of labels) {
      await page.getByRole('button', { name: label, exact: true }).click();
    }
    await sidebar(page).getByRole('button').last().click();
    await expect(page.getByRole('heading', { name: /chọn combo/i })).toBeVisible({
      timeout: 20_000,
    });
    await sidebar(page).getByRole('button').last().click();
    await expect(page.getByRole('heading', { name: /phương thức thanh toán/i })).toBeVisible({
      timeout: 20_000,
    });

    // Bad code -> error, no discount.
    await page.locator('#checkout-voucher').fill('KHONGCO');
    await page.getByRole('button', { name: /áp dụng/i }).click();
    await expect(page.getByText(/không hợp lệ/i)).toBeVisible();
    // Good code -> 10k off + payable shown.
    await page.locator('#checkout-voucher').fill('WELCOME10');
    await page.getByRole('button', { name: /áp dụng/i }).click();
    await expect(page.getByText('WELCOME10')).toBeVisible();
    await expect(page.getByText(/số tiền cần thanh toán/i)).toBeVisible();

    // Sidebar pay (same-tab redirect) still settles the full original amount via the mock gateway.
    await sidebar(page).getByRole('button').last().click();
    await expect(page).toHaveURL(/checkout/, { timeout: 20_000 });
    await page.locator('button[value="pay"]').click();
    await expect(page).toHaveURL(/\/payment-result/, { timeout: 20_000 });
    await expect(page.getByRole('heading', { name: /thanh toán thành công/i })).toBeVisible({
      timeout: 30_000,
    });
  });
});
