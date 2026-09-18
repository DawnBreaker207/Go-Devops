---
paths:
  - '**/src/api/**/*.ts'
  - '**/src/types/**/*.ts'
---

# API layer + types rules

## The client is already built — reuse it, do not rebuild it

`src/api/client.ts` owns: the axios instance, the Bearer request interceptor, a **single-flight 401 refresh with
a pending queue**, a separate `refreshClient` (so a refresh failure cannot loop), `normalizeError` and `unwrap`.

- Never import `axios` anywhere else. Never set `Authorization` by hand. Never add a second interceptor chain.
- On session death the client clears tokens and dispatches the `cp:unauthorized` window event; `App.tsx` handles
  the redirect. **Never navigate from inside api code.**
- Pass `{ skipAuthRefresh: true }` for a call that must not trigger the refresh flow (login, register).
- A rejected request always rejects with `ApiError` (`{code, message, details?}`), never an `AxiosError`.

## Endpoint modules

```ts
// src/api/movie.api.ts — the shape to copy
export const movieApi = {
  list: (query: PageQuery) =>
    apiClient.get<ApiResponse<PagedData<Movie>>>('/movies', { params: query }).then(unwrap),
  detail: (id: string) => apiClient.get<ApiResponse<Movie>>(`/movies/${id}`).then(unwrap),
  create: (payload: MoviePayload) =>
    // Movie writes live at /movies (RequireRoles admin|staff) — there is NO /admin/movies route.
    apiClient.post<ApiResponse<Movie>>('/movies', payload).then(unwrap),
};
```

- One file per domain: `src/api/<domain>.api.ts`, one exported object literal `<domain>Api`, arrow methods.
- Paths are relative to the `/api/v1` base already in `API_BASE_URL` — never repeat the prefix.
- **Verify the path exists** in `BackEnd-CP/internal/router/router.go` before adding a method. `docs/swagger.json`
  is 19 operations stale.
- A paged endpoint is `ApiResponse<PagedData<T>>`; a bare-array endpoint is `ApiResponse<T[]>`. Check per
  endpoint — `GET /showtimes`, `GET /movies/:id/showtimes`, `GET /halls/:id/seats`, `GET /admin/halls/:id/seats`,
  `GET /admin/halls/:id/prices`, `GET /admin/hall-templates`, `GET /payments/providers` and
  `GET /staff/showtimes/:id/tickets` all return a bare array.
- Two endpoints answer **204 with an empty body** (`DELETE /admin/halls/:id`, `DELETE /users/me`). Do not type
  them `ApiResponse<...>` and do not call `unwrap()` on them.
- A payload-less 200 (`DELETE /movies/:id`, `DELETE /admin/showtimes/:id`) has **no `data` field** — type it
  `ApiResponse<void>` and ignore the result.

## Types mirror the Go DTOs by hand

- Put shared types in `src/types/<domain>.ts` and re-export from `src/types/index.ts`. Entities are `<Entity>`,
  write models are `<Entity>Payload`.
- Read the Go source, not swagger: `BackEnd-CP/internal/dto/<domain>.go` for shapes,
  `BackEnd-CP/internal/models/<domain>.go` for enum literals.
- A Go field **without** `omitempty` is always present -> required here. **With** `omitempty` -> optional (`?`).
- Keep `snake_case` field names. There is no mapping layer; do not add one.
- `int64` money -> `number` (whole VND, no decimals, no currency field). UUID -> `string`.
  `time.Time` -> `string` (RFC3339 `+07:00`). The deliberate date-only fields (`release_date`, report
  `date`/`from`/`to`) are `string` in `YYYY-MM-DD`.
- A Go `oneof=` enum becomes a string union plus a `Record<Union, string>` label map. No TS `enum`.
- `ApiResponse<T>` currently declares `data: T` as non-optional even though the backend marks it `omitempty` —
  that is a known inaccuracy; do not "fix" it by making every call site optional without checking
  `.claude/context/decisions.md` first.
