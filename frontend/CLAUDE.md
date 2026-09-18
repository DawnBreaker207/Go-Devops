# FrontEnd-CP — Claude context

Auto-loaded when a session opens here, and lazily when a session at `CinemaProject/` reads a file in this repo.
Keep <200 lines. Deep dives: `.claude/skills/fe-page/api-contract.md` (what the backend really returns) and
`.claude/context/decisions.md` (the conventions this repo had NOT decided yet — read it before inventing one).
This file cites symbol + file, not line numbers — line numbers drift, grep the symbol.

## Project overview

Admin/staff web UI for `BackEnd-CP` (cinema booking). **One commit old and mostly a scaffold**: the plumbing is
real and should be reused, but only the `movie` feature is actually built. `ShowtimesPage` and `BookingsPage` are
15-line `<Empty/>` stubs — and both are **blocked on missing backend endpoints**, see
`../.claude/context/cross-repo-gotchas.md` before starting either.

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
  `hooks/`, `__tests__/`.
- `src/components/` — cross-feature: `Loading`, `PageHeader`, `ErrorBoundary` (the only class component).
- `src/layouts/` — `MainLayout` and `AuthLayout`. The sidebar is a **hardcoded four-entry `menuItems` array**
  inside `MainLayout` (dashboard, movies, showtimes, bookings), each with its own icon; both the menu and the
  breadcrumb read from it, and any path missing from it falls back to the dashboard key. A new top-level page
  must be added there by hand — it is not derived from `PATHS`.
- `src/routes/` — `index.tsx` exports `router` from `createBrowserRouter`, `paths.ts` exports `PATHS`,
  `ProtectedRoute.tsx` is a layout route.
- `src/stores/` — `authStore` (user/isAuthenticated/isBootstrapping + login/logout/bootstrap),
  `appStore` (theme/language/siderCollapsed).
- `src/types/` — hand-written mirrors of the Go DTOs, re-exported from `index.ts`.
- `src/utils/` — `storage.ts` (`tokenStorage`, all keys prefixed `cp_`), `format.ts` (date/duration helpers).
- `src/locales/` — `i18n.ts` + `vi.json` + `en.json`. `src/theme.ts` — `buildTheme(mode)` for antd tokens.

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
the `<Suspense>` boundary already lives in the layouts, so route entries add none. A new top-level route needs a
matching `menu.<slug>` i18n key or the breadcrumb renders the raw key.

**Forms.** antd `Form` + `Form.useForm<FormValues>()`, `layout="vertical"`, validation through antd `rules`.
Modals take `{ open, entity | null, confirmLoading, onCancel, onSubmit }`, use `destroyOnClose` +
`preserve={false}`, and seed defaults in a `useEffect` keyed on `[open, entity, form]`. For dates, model the form
value as `dayjs.Dayjs` and submit `.format('YYYY-MM-DD')`.

**i18n.** Every user-facing string goes through `t('<namespace>.<key>')`, and the key must be added to **both**
`vi.json` and `en.json` (vi is default and fallback). Dynamic key shapes in use: `movie.status<Capitalized>` and
`menu.<path without slash>`. Some keys already exist unused (`common.search`, `common.edit`, `common.delete`,
`common.loading`, `error.boundaryTitle`, `error.reload`) — wire those up instead of adding near-duplicates.

**Toasts.** `const { message } = App.useApp()` — **never** the static `message` import.

**Styling.** Inline `style={{...}}` plus antd tokens via `antdTheme.useToken()`. Customise antd only through
`buildTheme(mode)` in `src/theme.ts`. `src/index.css` is a 26-line reset. No CSS modules, no Tailwind, no
styled-components.

**TypeScript.** `strict` + `noUnusedLocals` + `noUnusedParameters`, so an unused import **fails the build**.
Use `import type` for type-only imports. Model enums as string unions (`'draft' | 'showing' | 'ended'`) with a
`Record<Union, string>` for labels — no TS `enum` anywhere.

**Comments.** Un-accented Vietnamese, sparingly, only for non-obvious intent. Match the existing style.

## Never

- Never build a screen against an endpoint you have not found in `BackEnd-CP/internal/router/router.go`.
  `docs/swagger.json` is 19 operations stale.
- Never assume the response is `{items, meta}` — several list endpoints return a bare array.
- Never send `ApiResponse<...>` through `unwrap()` for `DELETE /admin/halls/:id` or `DELETE /users/me`: both
  answer **204 with an empty body**.
- Never gate an admin action on `isAuthenticated` alone — the backend also checks the role, and `ProtectedRoute`
  currently does not.
- Never put a secret in a `VITE_*` variable; only `VITE_*` reaches the client. Declare every new one in **both**
  `.env.example` and the `ImportMetaEnv` interface in `src/vite-env.d.ts`.
- Never add a third error-display pattern. The repo already has two (`message.error` in `MoviesPage`, a local
  `<Alert>` in `LoginPage`) — pick one from `.claude/context/decisions.md` first.
- Never copy the three known-bad spots as a pattern: hardcoded Vietnamese in `ErrorBoundary`, the hardcoded
  `'Users'` card title in `DashboardPage`, the hardcoded `(phút)` suffix in `MovieFormModal`.
- Never assume `src/hooks/` or `src/assets/` exist — both are empty and therefore untracked by git.

## Build & dev commands (verified 2026-09-18)

| Purpose          | Command                                          | Notes                                                                                                                      |
| ---------------- | ------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| Dev server       | `npm run dev`                                    | :3000, no proxy — the backend must allow CORS from :3000                                                                   |
| Typecheck only   | `npx tsc --noEmit -p tsconfig.app.json`          | **passes today**; fastest correctness gate                                                                                 |
| Lint             | `npm run lint` / `npm run lint:fix`              | **passes today**                                                                                                           |
| Format           | `npm run format`                                 | `src/` only. Root config files are still covered by lint-staged, which basename-matches `*.{ts,tsx}` and `*.{css,json,md}` |
| Tests            | `npm test` (`vitest run`) / `npm run test:watch` | **1 file / 3 tests pass today**; no coverage provider installed                                                            |
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
