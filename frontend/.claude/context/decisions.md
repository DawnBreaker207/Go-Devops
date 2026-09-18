# Open conventions — PROPOSED defaults awaiting the owner's approval

FrontEnd-CP is one commit old. For each item below the repo either has no precedent or has **two conflicting**
ones, so a session that guesses will fork the codebase. Each entry states what the code does today, the proposed
default, and why.

**Status of every entry: PROPOSED (2026-09-18) — not yet approved by the repo owner.**
Until an entry is marked APPROVED, follow the proposal but say in your report that you did, so it can be
corrected cheaply. If a task's requirement contradicts a proposal, ask instead of silently diverging.

---

## 1. Where errors are shown

**Today** two patterns: `MoviesPage` calls `message.error(...)` in a try/catch around `mutateAsync`;
`LoginPage` renders a local `<Alert type="error">` from `useState`.
**Proposed** Mutation and action failures -> `message.error` toast. Failures that block the whole screen
(a failed list query, a failed login) -> inline `<Alert>` so the user can read and retry it.
**Why** A toast is wrong for something the user must act on, and an Alert is noise for a transient action.
Both existing usages then become correct instead of one of them being wrong.

## 2. Which layer owns the toast

**Today** `MoviesPage` does it in the component; `useMovies.ts` hooks only invalidate.
**Proposed** The **component** owns user feedback; hooks stay silent and only handle cache invalidation.
**Why** It matches the one precedent, and a hook reused by two screens should not hardcode one screen's copy.

## 3. Query keys

**Today** one string const per hook file (`MOVIE_QUERY_KEY = 'movies'`), list key `[KEY, query]`, blanket
`invalidateQueries({ queryKey: [KEY] })`.
**Proposed** Keep it. Add a detail key as `[KEY, 'detail', id]` when detail routes arrive. No key-factory object
until a third level of nesting actually exists; keep the blanket invalidate.
**Why** A factory for four hooks is ceremony. Blanket invalidation is correct and cheap at this data size.

## 4. Form validation messages and backend field errors

**Today** `LoginPage` supplies i18n messages in its `rules`; `MovieFormModal` relies on antd's default locale text.
**Proposed** Every `rules` entry carries an explicit i18n `message`. On a `40001`, map `ApiError.details` onto
the form with `form.setFields([{ name, errors: [msg] }])` — add one shared helper the first time it is needed.
**Why** antd's default text is not translated consistently and ignores the vi/en switch. `details` keys are
already the exact field names, so the mapping is one line.

## 5. Barrel files

**Today** `src/api|components|stores|utils/index.ts` exist and **nothing imports them**; only `@/types` is used.
**Proposed** Keep `@/types` as a barrel. Leave the other four files untouched but do not add to them and do not
import them; delete them only in a dedicated cleanup commit.
**Why** They are dead weight, but deleting them is unrelated churn inside a feature task.

## 6. Permission model

**Today** none. `ProtectedRoute` checks the token only; `User.role` is read nowhere.
**Proposed** Add `<RequireRole roles={['admin']}>` as a layout route wrapper for route-level gating, plus a
`useHasRole(...)` selector for hiding menu items and disabling buttons. No generic `<Can>`/policy engine.
**Why** The backend gates by route and by role with **no hierarchy**, so a route-level wrapper mirrors it
exactly. Today a customer can open an admin screen and only learns of the 403 on submit.

## 7. Paging and filter state

**Today** component `useState` per page; state is lost on reload and links are not shareable.
**Proposed** Move page / page_size / search into URL search params with `useSearchParams`, via one shared
`useListQuery()` helper introduced with the second list screen.
**Why** Admin tables get deep-linked and refreshed constantly. Doing it once in a helper is cheaper than
retrofitting four screens later.

## 8. Naming for non-page feature files

**Today** only `features/movie/components/MovieFormModal.tsx` exists.
**Proposed** `<Domain><Thing><Kind>.tsx` with the kind as the suffix: `MovieFormModal`, `MovieTable`,
`MovieFilterBar`, `ShowtimeDetailDrawer`. Feature-local non-components go in `features/<domain>/constants.ts`
and `features/<domain>/utils.ts`.
**Why** Extends the single existing name instead of inventing a scheme beside it.

## 9. `src/hooks/`

**Today** an empty, untracked directory. README documents it; git does not contain it.
**Proposed** Reserve it for hooks used by **two or more** features (`useListQuery`, `useHasRole`). Create it with
the first real file. Anything feature-specific stays in `features/<domain>/hooks/`.
**Why** Matches how `src/components/` vs `features/*/components/` already splits.

## 10. `src/assets/` vs `public/`

**Today** both empty; `index.html` links `/vite.svg`, which does not exist, so the favicon 404s.
**Proposed** Imported, hashed, bundled assets -> `src/assets/`. Files served verbatim at a fixed URL (favicon,
`robots.txt`) -> `public/`. Fix the favicon 404 as part of the first UI polish task.
**Why** Standard Vite split; the broken favicon is the first concrete case.

## 11. In-page loading UX

**Today** `<Loading/>` for Suspense and bootstrap; `Table loading={isFetching}` in `MoviesPage`.
**Proposed** Tables and lists -> the component's own `loading` prop. First paint of a detail/form screen ->
antd `Skeleton`. Full-screen only for route-level Suspense and app bootstrap, which already works that way.
**Why** Swapping a whole table for a spinner loses the header and the user's scroll position.

## 12. Test scope

**Today** one test (`LoginPage`), no network mocking, no shared render helper, and it asserts **Vietnamese** copy.
It also does `useAuthStore.setState({ login: loginSpy })`, which permanently mutates the real store for any
later test.
**Proposed** (a) Add a shared `renderWithProviders()` test helper the second time providers are needed.
(b) Keep asserting Vietnamese copy, since `vi` is the default and fallback locale. (c) Mock at the **api-module**
boundary with `vi.mock('@/api/<domain>.api')` — do not add MSW yet. (d) Reset zustand stores in an `afterEach`.
(e) Worth a test: anything with real logic (seat selection, price math, the 401 refresh queue, `RequireRole`).
Not worth one: pure layout.
**Why** MSW is a large dependency for a repo with one test; api-module mocking gives the same isolation. The
store-leak is a real bug waiting to bite the second test file.

## 13. Commits and branches

**Today** a single commit, `chore: initialize project base`.
**Proposed** Conventional Commits with an optional scope, matching BackEnd-CP's 45-commit history:
`feat|fix|chore|test|refactor|perf|docs(<scope>): <subject in English>`. Branches use the `feat/…` prefix, the
only one the sibling repo has ever used, cut off `main`. Scopes worth using here: `auth`, `movie`, `showtime`,
`booking`, `api`, `ui`, `i18n`.
**Why** One sample is not a convention, but the sibling repo's is unambiguous and the two are developed together.

## 14. Dashboard data source

**Today** `DashboardPage` abuses `useMovieList({page:1,page_size:1})` to read `meta.total` and hardcodes `0` for
the other three cards.
**Proposed** Do not extend the hack. Either (a) ask the backend for a stats endpoint, or (b) reduce the dashboard
to the tiles that have a real source. Pick one with the owner before touching the page.
**Why** Three tiles currently lie. `GET /admin/overview` exists but returns today's aggregate + upcoming
showtimes + alerts, not these counts, and is admin-only.

## 15. CSS at scale

**Today** inline `style={{...}}` + antd tokens; `src/index.css` is a 26-line reset.
**Proposed** Keep inline styles and tokens. Introduce a spacing scale const in `src/theme.ts` when a third
screen repeats the same magic numbers. Do not add Tailwind, CSS modules or styled-components without asking.
**Why** antd already owns the design tokens; a second styling system would fragment theming and dark mode.

## 16. Realtime (SSE) ownership — no precedent at all

**Today** nothing in the app consumes SSE, so there is no pattern for where an `EventSource` lives, who owns its
lifecycle, or how a `seats` event reaches react-query.
**Proposed** One hook per stream, `src/features/<domain>/hooks/use<Thing>Stream.ts`: it owns the `EventSource`,
opens it in a `useEffect` keyed on the entity id, closes it on unmount, re-tokens on error with a backoff, and
applies events with `queryClient.setQueryData` on the seat-map key (not `invalidateQueries`, which would refetch
the whole map on every seat change). `EventSource` is the only sanctioned bypass of `apiClient`.
**Why** The seat map is the next real screen, the stream is force-closed after ~30 min so reconnection is not
optional, and a blanket invalidate per event would hammer the API during a busy show.

---

## Also known-broken, not conventions (fix when nearby)

- `index.html` links a favicon that does not exist -> 404 in dev and in the nginx image.
- `ErrorBoundary` hardcodes Vietnamese text while `error.boundaryTitle` / `error.reload` already exist in both
  locale files. `DashboardPage` hardcodes `'Users'`. `MovieFormModal` hardcodes the `(phút)` suffix.
- `appStore.ts` defines `LANG_KEY = 'cp_language'` while `locales/i18n.ts` re-reads the same key as an inline
  literal — two sources of truth. The theme/language keys also bypass `src/utils/storage.ts`, which is supposed
  to own localStorage.
- `VITE_APP_NAME` is declared in `.env.example` and `vite-env.d.ts` but read nowhere; the title comes from the
  `common.appName` i18n key and a hardcoded `<title>` in `index.html`.
- `MoviesPage` renders `showTotal: (total) => ${total}` — a bare number with no label in either language.
- `movieApi.detail(id)` and `authApi.register(...)` are dead code: no caller, and no `/movies/:id` or `/register`
  route exists.
- `coverage/` is git-ignored but no coverage provider is installed, so `vitest --coverage` fails.
- No CI: lint/build/test are enforced only by the local husky pre-commit hook.
