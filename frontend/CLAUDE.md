# FrontEnd-CP — Claude context

Auto-loaded when a session opens here, and lazily when a session at `CinemaProject/` reads a file in this repo.
Keep <200 lines. Deep dives: `.claude/skills/fe-page/api-contract.md` (what the backend really returns),
`.claude/context/decisions.md` (the conventions this repo had NOT decided yet — read it before inventing one),
and `.claude/context/figma.md` (the design reference: how to open it, every frame's node-id, per-screen specs).
This file cites symbol + file, not line numbers — line numbers drift, grep the symbol.

## Project overview

Admin/staff web UI for `BackEnd-CP` (cinema booking). **There is no customer-facing UI yet** — every screen here
is for an operator. Built: `movie` (table + form), `showtime` (table + filters + form), `booking` (table +
filters + detail drawer), `hall` (table + create/edit/clone/price modals + a seat-grid editor at
`/halls/:id/seats`), `user` (table + filters + inline role/lock controls + create modal),
`report` (date range + per-movie rollup + per-day table with a per-showtime breakdown), `dashboard`.
**The customer side is complete**: `CustomerLayout` (top nav, no sider, plain CSS, its own visual world),
`browse` (home poster grid + film detail with a day strip and showtime picker), the full `booking-flow`
(seat picker -> hold -> checkout with a countdown -> gateway -> auto-confirm -> tickets -> my tickets), and
the customer account screens (register, forgot password, reset password) on the Figma split layout. Read
`../.claude/context/cross-repo-gotchas.md` for the response-shape traps before adding a screen.

**The visual language comes from a Figma file**; the information architecture does not — it mirrors the
backend. **Before building a screen, open its Figma frame** and follow it: `.claude/context/figma.md` has every
frame's node-id, the only reliable way to open one, and the per-screen specs measured so far.

## Tech stack (versions from package.json + package-lock.json)

React **18.3** (not 19), TypeScript **6.0** (strict), Vite **8.2** (dev port 3000, no proxy), antd **5.29** +
`@ant-design/icons`, `@tanstack/react-query` **v5**, zustand **5**, `react-router-dom` **v6** (data router),
axios **1.20**, i18next **26** + react-i18next (vi default, en secondary), dayjs, vitest **5** + jsdom +
@testing-library/react, eslint flat config + prettier + husky/lint-staged. Package manager is **npm**
(`package-lock.json` is the only lockfile; `npm ci` in the Dockerfile). Node 22 per the Dockerfile; there is
**no `engines` field and no `.nvmrc`**. **No schema-validation library** — no zod, yup or react-hook-form.

## Architecture map (concrete paths only)

- `src/main.tsx` — mounts `<App/>`; side-effect `import '@/locales/i18n'`.
- `src/App.tsx` — provider chain, in this order: `ConfigProvider > AntdApp > ErrorBoundary >
QueryClientProvider > RouterProvider`. The `QueryClient` is created once via `useState(createQueryClient)`
  with `retry: 1`, `refetchOnWindowFocus: false`, `staleTime: 30_000`. It also listens for the
  `cp:unauthorized` window event and runs logout + `queryClient.clear()` + `router.navigate('/login')`.
- `src/api/client.ts` — the single axios instance. Bearer request interceptor, **single-flight 401 refresh with a
  pending queue**, a separate `refreshClient` so refresh cannot loop, `normalizeError` -> `ApiError`,
  `unwrap()` -> `response.data.data`, and the `skipAuthRefresh` config flag.
- `src/api/<domain>.api.ts` — one `<domain>Api` object literal of arrow methods.
- `src/features/<domain>/` — singular folder, **plural** page (`movie/MoviesPage.tsx`), plus `components/`,
  `hooks/`, `constants.ts`, `__tests__/`. Domains: `auth`, `dashboard`, `movie`, `showtime`, `hall`, `booking`, `user`, `report`, `browse`, `booking-flow`.
  `auth` holds BOTH the operator `LoginPage` (an antd card in `AuthLayout`) and the customer account screens
  (`CustomerAuthShell`, a split panel) — they look nothing alike on purpose.
- `src/components/` — cross-feature: `Loading`, `PageHeader`, `ErrorBoundary` (the only class component).
- `src/layouts/` — `MainLayout` and `AuthLayout`. The sidebar and the breadcrumb are both derived from
  `NAV_ITEMS` in `src/routes/navigation.tsx`, filtered by role through `useNavItems()`; `MainLayout` itself
  holds no menu data.
- `src/routes/` — `index.tsx` exports `router` from `createBrowserRouter`, `paths.ts` exports `PATHS`,
  `ProtectedRoute.tsx` (token) and `RequireRole.tsx` (role) are layout routes, and `navigation.tsx` is the
  single source of nav entries + the `ROLES_OPERATOR` / `ROLES_ADMIN` constants the router groups by.
- `src/hooks/` — hooks shared by two or more features: `useHasRole` (+ `useCurrentRole`) and `useListQuery`.
  Feature-specific hooks stay in `features/<domain>/hooks/`.
- `src/test/` — `renderWithProviders` / `createTestQueryClient`, the shared test harness. Never imported by
  app code.
- `src/stores/` — `authStore` (user/isAuthenticated/isBootstrapping + login/logout/bootstrap),
  `appStore` (theme/language/siderCollapsed).
- `src/types/` — hand-written mirrors of the Go DTOs, re-exported from `index.ts`.
- `src/utils/` — `storage.ts` (`tokenStorage`, all keys prefixed `cp_`), `format.ts` (date/duration/money,
  zone pinned to `Asia/Ho_Chi_Minh`), `error.ts` (`isApiError`, `errorMessage`, `fieldErrorsOf`),
  `form.ts` (`applyApiFieldErrors`).
- `src/locales/` — `i18n.ts` + `vi.json` + `en.json`.
- `src/theme/` — `tokens.ts` is the single source of every colour; `index.ts` maps them onto antd via
  `buildTheme(mode)` and re-exports them. Mirrored into `--cp-*` in `src/index.css`, with a parity test.

## Critical conventions

**Imports.** Use the `@/` alias (configured in **both** `vite.config.ts` and `tsconfig.app.json` — keep them in
sync). Import types from the `@/types` barrel; import everything else by **deep path**. The barrels in
`src/api|components|stores|utils/index.ts` exist but nothing imports them — do not start relying on them.

**Data flow.** Server state lives in react-query hooks under `src/features/*/hooks/`. Session and UI state live
in zustand. **Never cache an API list in zustand.** Every HTTP call goes through `apiClient`; never import axios
in a component, hook or store, and never set the `Authorization` header by hand.

**API modules.** `apiClient.<verb><ApiResponse<T>>(path, ...).then(unwrap)`. Wire fields stay **snake_case** on
both sides — there is no camelCase mapping layer and adding one is out of scope. Pass `{ skipAuthRefresh: true }`
for endpoints that must not trigger the refresh flow (login and register already do).

**Hooks.** One `export const <DOMAIN>_QUERY_KEY = '<domain>'` per feature hook file; list key is
`[<DOMAIN>_QUERY_KEY, query]`. Mutations are `use<Verb><Domain>` and invalidate `[<DOMAIN>_QUERY_KEY]` on
success. Keep the previous page visible with `placeholderData: (previous) => previous` and drive antd `Table`
`loading` from `isFetching`, not `isLoading`.

**Components.** Arrow-function consts with a local `interface <Name>Props` declared directly above. Export both
a named const **and** a default (`routes/index.tsx` lazy-imports the default). The four content pages start with
`<PageHeader title=... />`; `LoginPage` (a `<Card>`) and `NotFoundPage` (an antd `<Result>`) deliberately do not.

**Routing.** Data router only — never introduce `<Routes>/<Route>` JSX. Every path is a literal in `PATHS`
(`src/routes/paths.ts`); no path string is hardcoded elsewhere. Lazy-load each page in `src/routes/index.tsx`;
the `<Suspense>` boundary already lives in the layouts, so route entries add none. A new top-level page is
**two lines**: one `NAV_ITEMS` entry in `src/routes/navigation.tsx` and one route in `src/routes/index.tsx`
nested under the `<RequireRole>` group that uses the **same** roles constant — that is what keeps the menu from
offering a link that 403s. It also needs a `menu.<i18nKey>` key in both locale files.

**Permissions.** Mirror the backend: `RequireRoles` there is an exact, case-sensitive lookup with **no
hierarchy**, so `useHasRole('admin')` is false for a staff account and vice versa. Route-level gating is
`<RequireRole roles={...}/>`; hiding a menu entry or disabling a button is `useHasRole(...)`. Never invent a
rank or a `<Can>` policy engine.

**List state.** Page / page size / search live in the URL via `useListQuery()`, not in `useState`, so a table
survives F5 and can be linked. Pass its `query` straight into the API call and the react-query key.

**Forms.** antd `Form` + `Form.useForm<FormValues>()`, `layout="vertical"`, validation through antd `rules`,
every rule carrying an explicit i18n `message`. Modals take
`{ open, entity | null, confirmLoading, onCancel, onSubmit }`, use `destroyOnClose` + `preserve={false}`, and
seed defaults in a `useEffect` keyed on `[open, entity, form]`. For dates, model the form value as `dayjs.Dayjs`
and submit `.format('YYYY-MM-DD')`. `onSubmit` returns a promise and **throws on failure**: the modal catches it,
runs `applyApiFieldErrors(form, error)` so a 400/40001 lands on the right input, and only toasts what is left.
The page must therefore NOT swallow the mutation error.

**i18n.** Every user-facing string goes through `t('<namespace>.<key>')`, and the key must be added to **both**
`vi.json` and `en.json` (vi is default and fallback). Dynamic key shapes in use: `movie.status<Capitalized>`,
`menu.<i18nKey>`, and the snake_case enum shapes `booking.status_<value>` / `booking.payment_<value>` /
`booking.soldVia_<value>` / `booking.reason_<value>` / `booking.seatType_<value>` / `booking.ticket_<value>`.
Some keys exist unused (`common.search`, `common.edit`, `common.delete`, `common.loading`) — wire those up
instead of adding near-duplicates.

antd's own strings (date pickers, pagination, table filters) come from `ConfigProvider locale`, wired in
`App.tsx`. **`antd/locale/<name>` is a CJS re-export and arrives double-`default`-wrapped through Vite**, so it
is unwrapped by `unwrapLocale` there — passing the import straight in silently leaves antd in English. The
DatePicker's month names come from the GLOBAL dayjs locale instead, set in an effect on `language`.

**Backend error messages are English.** A 40900 conflict toast currently shows the server's own sentence inside
a Vietnamese screen. All showtime conflicts share code 40900, so they cannot be told apart by code alone —
`TODO: confirm` with the owner whether to add a message-keyed translation table.

**Toasts.** `const { message } = App.useApp()` — **never** the static `message` import.

**Motion.** All animation follows `.claude/rules/motion.md` and the tokens in `src/motion.ts`; use the
`fe-motion` skill when adding or reviewing it.

**Styling.** Inline `style={{...}}` plus antd tokens via `antdTheme.useToken()`. All colour comes from
`src/theme/tokens.ts` — see `.claude/rules/design-tokens.md`, which is binding. **Never write a hex literal in a
component.** Customer-facing screens use plain CSS with `var(--cp-*)` and the motion tokens, the way
`MovieCard.css` already does; they do not get antd's heavy components. `src/index.css` is a 26-line reset. No CSS modules, no Tailwind, no
styled-components.

**TypeScript.** `strict` + `noUnusedLocals` + `noUnusedParameters`, so an unused import **fails the build**.
Use `import type` for type-only imports. Model enums as string unions (`'draft' | 'showing' | 'ended'`) with a
`Record<Union, string>` for labels — no TS `enum` anywhere.

**Comments.** Un-accented Vietnamese, sparingly, only for non-obvious intent. Match the existing style.

## Never

- Never build a screen against an endpoint you have not found in `BackEnd-CP/internal/router/router.go`.
  `docs/swagger.json` still misses 6 alias operations.
- Never assume the response is `{items, meta}` — several list endpoints return a bare array.
- Never send `ApiResponse<...>` through `unwrap()` for `DELETE /admin/halls/:id` or `DELETE /users/me`: both
  answer **204 with an empty body**.
- Never render a seat grid by array index. `col_span` is 1 or 2, a 2-column seat swallows the next column, and
  **no seat row exists for the swallowed column** — so column numbers have holes. Place each seat at its
  `col_number`. The pure math lives in `src/features/hall/seatGrid.ts` and is unit-tested; reuse it rather than
  re-deriving it for the customer seat picker.
- Never offer a control the backend will refuse. `/admin/users` rejects locking yourself and changing your own
  role with a 409; `UsersPage` disables both controls on the signed-in user's own row instead. The remaining
  guard (the last active admin) cannot be known client-side, so that one is left to the 409.
- Never treat a seat as bookable because `status` says `available`. `showtime_seat_id` is the ONLY omitempty
  field on `SeatMapSeat`, it is what `POST /orders/hold` wants, and a seat without it still reports
  `available` because the SQL is a LEFT JOIN. `is_gap` seats also report `available` and also carry an id —
  holding one is a 400. Check both before letting a seat be picked.
- Never render an order's tickets from `GET /orders`: that list returns `OrderStatusResponse` WITHOUT them.
  `TicketCard` distinguishes `tickets === undefined` (not loaded) from `[]` (loaded, genuinely none) — mixing
  the two made a CONFIRMED order read as "Tickets (0) — awaiting payment".
- Never trust a client-side "I paid". The gateway reports through the IPN; the checkout page polls
  `GET /orders/:id/status` and `POST /orders/:id/confirm` refuses anything the provider has not settled.
- Never read `price: 0` on a seat as free. It means the hall has no `hall_prices` row for that seat type, and
  holding it is a 409.
- Never treat a missing day in `GET /admin/reports/daily` as a zero. `days` holds only the days the `closeDay`
  job has closed; a day with no business IS closed and DOES appear, with zeros. A day that is absent has no
  figures at all. `missingDays()` in `src/features/report/reportRollup.ts` names them.
- Never multiply `occupancy_rate` by 100 — the backend already does (`ROUND(100.0 * seats_sold / capacity)`).
- Never assume `rows * seats_per_row` is capacity. It is the cell count; sellable capacity is
  `seats.filter(s => !s.is_gap).length`, which is how every backend report query counts it.
- Never gate an admin action on `isAuthenticated` alone — `ProtectedRoute` only checks the token. Role gating is
  `<RequireRole>` on the route and `useHasRole(...)` on the control.
- Never put a secret in a `VITE_*` variable; only `VITE_*` reaches the client. Declare every new one in **both**
  `.env.example` and the `ImportMetaEnv` interface in `src/vite-env.d.ts`.
- Never add a third error-display pattern. There are exactly two: `message.error` for a failed action, an inline
  `<Alert>` for a failure that blocks the screen (a failed list query, a failed login). Field-level validation
  goes through `applyApiFieldErrors`, not a toast.
- Never copy `ErrorBoundary`'s hardcoded Vietnamese as a pattern — `error.boundaryTitle` / `error.reload`
  already exist in both locale files.
- Never edit `MainLayout`'s menu by hand; it is derived from `NAV_ITEMS`.
- Never assume `src/assets/` exists — it is empty and therefore untracked by git.
- Never write a colour anywhere but `src/theme/tokens.ts`, and never add Tailwind/CSS modules/styled-components
  — settled in `.claude/context/decisions.md` #15.

## Build & dev commands (verified 2026-09-18)

| Purpose          | Command                                          | Notes                                                                                                                      |
| ---------------- | ------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| Dev server       | `npm run dev`                                    | :3000, no proxy — the backend must allow CORS from :3000                                                                   |
| Typecheck only   | `npx tsc --noEmit -p tsconfig.app.json`          | **passes today**; fastest correctness gate                                                                                 |
| Lint             | `npm run lint` / `npm run lint:fix`              | **passes today**                                                                                                           |
| Format           | `npm run format`                                 | `src/` only. Root config files are still covered by lint-staged, which basename-matches `*.{ts,tsx}` and `*.{css,json,md}` |
| Tests            | `npm test` (`vitest run`) / `npm run test:watch` | **9 files / 121 tests pass today**; no coverage provider installed. Stores reset in `setupTests.ts` afterEach              |
| Production build | `npm run build` (`tsc -b && vite build`)         | typecheck then bundle to `dist/`                                                                                           |
| Preview build    | `npm run preview`                                | serves `dist/`                                                                                                             |

The husky `pre-commit` hook runs `lint-staged`: `eslint --fix` + `prettier --write` on staged `*.{ts,tsx}`, and
`prettier --write` on staged `*.{css,json,md}`. Staged files can therefore be rewritten during a commit.

## When you finish a task

1. `npx tsc --noEmit -p tsconfig.app.json && npm run lint && npm test`.
2. New i18n keys added to **both** `vi.json` and `en.json`? New `VITE_*` var in both `.env.example` and
   `vite-env.d.ts`?
3. If you relied on a backend field, say which endpoint and which DTO — the types here are hand-written.
4. Report what you did NOT do and every `TODO: confirm` you left.
5. Leave `git status` clean except the files you meant to change.

<!-- last verified: 2026-09-18 against 59202fd (main) -->
