---
paths:
  - '**/src/features/**/*.tsx'
  - '**/src/components/**/*.tsx'
  - '**/src/layouts/**/*.tsx'
  - '**/src/pages/**/*.tsx'
---

# Page + component rules

## Structure

- Feature-first: a **singular** folder with a **plural** page — `src/features/movie/MoviesPage.tsx`,
  `src/features/showtime/ShowtimesPage.tsx` — plus `components/`, `hooks/` and `__tests__/` underneath it. `src/pages/` is only for non-feature standalone screens (currently just `NotFoundPage`).
- Cross-feature presentational components go in `src/components/`.
- Arrow-function const + a local `interface <Name>Props` right above it. Export a named const **and** a default.
- Reuse what exists: `<PageHeader title extra? />` at the top of every content page (`LoginPage` and
  `NotFoundPage` are deliberate exceptions), `<Loading fullscreen? tip? />`,
  and the app-level `<ErrorBoundary>`. `ErrorBoundary` is the only class component — keep it that way.

## Data

- Server data comes from a react-query hook in `src/features/<domain>/hooks/use<Domain>s.ts`; never call an api
  module directly from a component. Never keep server lists in zustand.
- Query key: `[<DOMAIN>_QUERY_KEY, query]` with one string const per hook file. Mutations are
  `use<Verb><Domain>` and invalidate `[<DOMAIN>_QUERY_KEY]` on success.
- `placeholderData: (previous) => previous` on list queries; antd `Table loading={isFetching}`.
- Select from a store with a single-field selector: `useAuthStore((s) => s.user)`.

## Every screen needs four states

`loading` (Table `loading` / `<Loading/>`), `error` (see `.claude/context/decisions.md` — do not invent a third
pattern), `empty` (antd `<Empty/>`), and the happy path. A screen that only renders the happy path is incomplete.

## Tables

antd `Table<T>` with `rowKey="id"`, a `ColumnsType<T>` built inline in the component body, titles from `t()`, a
fixed `width` per column, `scroll={{ x: 900 }}`, row actions as text-type icon `Button`s with
an `aria-label` of the form `edit-${record.id}`, and delete wrapped in `Popconfirm`.

## Forms

antd `Form` + `Form.useForm<FormValues>()`, `layout="vertical"`, `rules` for validation, every rule carrying an
explicit i18n `message`. **There is no schema validation library** — do not import zod/yup/react-hook-form.
Modal props are `{ open, entity | null, confirmLoading, onCancel, onSubmit }` with `destroyOnHidden` and
`preserve={false}`. antd 5.29 deprecated `destroyOnClose`; using it logs a console error.

**Seed the form with `initialValues`, never with `setFieldsValue` inside a `useEffect`.** Verified at runtime on
2026-09-18: the effect form silently produces an **empty modal**. `main.tsx` mounts under `StrictMode`, which
runs mount effects, then cleanups, then mount effects again; antd's `preserve={false}` field cleanup deletes the
value from the store and lands after the effect's `setFieldsValue`. `destroyOnHidden` gives each open a fresh
`<Form>`, so `initialValues` are re-applied every time and nothing depends on effect ordering.

`onSubmit` returns a promise and **throws on failure**. The modal catches it, calls
`applyApiFieldErrors(form, error)` from `src/utils/form.ts` so a 400/40001 `details` map lands on the right
inputs, and toasts only what is left with `errorMessage(error, fallback)`. A page that try/catches the mutation
itself breaks this — let the error out.

Before adding a form field, check the backend binding: an over-permissive form produces a 400 with a `details`
map, and a form that omits a field the backend **overwrites** silently destroys data — `PUT /movies/:id` is a
full replace, which is exactly how `trailer_url` and `cast` used to get wiped and `age_rating` reset to `P`
(fixed in `MovieFormModal` on 2026-09-18 by carrying all three). A `details` key with no matching `Form.Item`
is accepted by antd and drawn nowhere, so the form must cover every field its endpoint validates.

## Routing and i18n

- Paths only from `PATHS`. Lazy-load pages in `src/routes/index.tsx`; the layouts own `<Suspense>`.
- A new top-level route needs a `menu.<slug>` key or the breadcrumb shows the raw key.
- Every user-facing string through `t()`, with the key in **both** `vi.json` and `en.json`.
- Toasts via `App.useApp()`, never the static `message` import.

## Permissions

`ProtectedRoute` checks the token only; the role gate is `<RequireRole roles={...}/>` on the route and
`useHasRole(...)` on a control. The roles constants live in `src/routes/navigation.tsx` and are shared by the
sider and the router, so the menu can never offer a link that 403s.

Mirror the backend exactly, because `RequireRoles` there is a case-sensitive map lookup with **no hierarchy**:
`admin|staff` for movie writes and most of `/admin/*` (hall-templates, halls, showtimes, uploads); admin-only for
`/admin/users*`, `/admin/reports/daily`, `/admin/overview`, `/admin/stats`, `/admin/orders`, `/admin/batch/jobs*`
and `/admin/audit-logs`. An admin-only query on a screen that staff can also open must be
`enabled: useHasRole('admin')` rather than firing a request that is certain to 403.

## Styling

Inline `style={{...}}` + `antdTheme.useToken()` tokens (`token.colorBgContainer`, `token.borderRadiusLG`).
Theme changes belong in `src/theme.ts` only. Prettier: single quotes, semicolons, 100 columns.
