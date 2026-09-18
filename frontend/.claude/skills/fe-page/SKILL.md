---
name: fe-page
description: fe-page - build a FrontEnd-CP screen wired to a real backend endpoint, with loading, error and empty states, i18n keys and a route. Use for any new page or a page rewrite in this repo.
argument-hint: <route and purpose, e.g. "/showtimes danh sach suat chieu cho admin">
---

# /fe-page $ARGUMENTS

Build the screen for **$ARGUMENTS**. What the backend really returns: `api-contract.md` in this directory.
Undecided conventions: `../../context/decisions.md` — read it instead of inventing one.

## 0. Verify the endpoint exists — this is the step that gets skipped

```bash
grep -nE '\.(GET|POST|PUT|PATCH|DELETE|Match)\(' ../BackEnd-CP/internal/router/router.go | grep -i <domain>
```

Then read the DTO in `../BackEnd-CP/internal/dto/<domain>.go`. **`docs/swagger.json` is 19 operations stale.**

If the endpoint does not exist, **STOP and tell the user**. `ShowtimesPage` and `BookingsPage` are blocked on
exactly this — see `../../../../.claude/context/cross-repo-gotchas.md` (the workspace-root one). Do not fake
data and do not build a shell.

Also check the guard: `admin`-only, `staff+admin`, or `customer`-only. Roles have **no hierarchy**, so an admin
token gets 403 on every `/orders/*` route except `GET /orders`, which has no role gate and simply returns that
admin's own (empty) order list. Most of `/admin/*` accepts `staff` too — only `/admin/users*`,
`/admin/reports/daily`, `/admin/overview`, `/admin/batch/jobs*` and `/admin/audit-logs` are admin-only.

## 1. Types — `src/types/<domain>.ts`

Mirror the Go DTO field by field, snake_case, `omitempty` -> optional. Re-export from `src/types/index.ts`.
Enums become string unions plus a `Record<Union, string>` label map.

## 2. API module — `src/api/<domain>.api.ts`

One `<domain>Api` object literal. Copy the shape from `movie.api.ts`. Decide paged vs bare array **from the
endpoint**, not by assumption.

## 3. Hook — `src/features/<domain>/hooks/use<Domain>s.ts`

One `<DOMAIN>_QUERY_KEY` const; list key `[KEY, query]`; `placeholderData: (previous) => previous`; one
`use<Verb><Domain>` per mutation, each invalidating `[KEY]` on success.

## 4. Page — `src/features/<domain>/<Domain>sPage.tsx`

The folder is singular and the page is **plural**: `features/movie/MoviesPage.tsx`, `features/showtime/ShowtimesPage.tsx`.

- `<PageHeader title={t('<domain>.title')} extra={...} />` first.
- All four states: loading (`Table loading={isFetching}` or `<Loading/>`), error, empty (`<Empty/>`), happy path.
- antd `Table<T>` with `rowKey="id"`, `ColumnsType<T>` inline, `scroll={{ x: 900 }}`, `Popconfirm` on delete.
- Format dates and durations through `src/utils/format.ts`, never `dayjs` inline in a renderer. There is **no
  money helper yet** — add `formatVND` to `format.ts` for the first money-bearing screen instead of inlining
  `toLocaleString`. And read the timezone trap in `api-contract.md` before rendering a showtime time.
- Export a named const **and** a default.

## 5. Route + menu + i18n

- Add the path to `PATHS` in `src/routes/paths.ts`.
- Lazy-import the page in `src/routes/index.tsx` under `ProtectedRoute > MainLayout`. Add no `<Suspense>` — the
  layout owns it.
- A top-level route must ALSO be added to the hardcoded `menuItems` array in `MainLayout` (with an icon), or it
  will not appear in the sidebar and the breadcrumb will fall back to the dashboard key.
- It needs a `menu.<slug>` i18n key too, or the breadcrumb renders the raw key.
- Every string through `t()`, with the key added to **both** `src/locales/vi.json` and `en.json`.

## 6. Verify

```bash
npx tsc --noEmit -p tsconfig.app.json && npm run lint && npm test
```

`strict` + `noUnusedLocals` means an unused import fails the build. If you can run the backend, click the screen
once; otherwise say plainly that it was not exercised against a live API.

## 7. Report

In Vietnamese: the route, the endpoints it calls with their guard, which types you added or changed, which i18n
keys you added, which of the four states are real versus placeholder, and every `TODO: confirm` left behind.
