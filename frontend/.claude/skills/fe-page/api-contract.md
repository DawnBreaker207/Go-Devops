# The backend contract, from the frontend's side

What `BackEnd-CP` actually puts on the wire and what this app must do about it. Verified 2026-09-18.
The authoritative Go-side version is `../../../../BackEnd-CP/.claude/context/contract.md`; the full route table
with guards is `../../../../BackEnd-CP/.claude/skills/be-endpoint/endpoints.md`.

## Base

- `API_BASE_URL` = `import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1'`. The `/api/v1` prefix is
  hardcoded server-side, so api-module paths are always relative to it.
- Vite dev runs on **:3000 with no proxy** -> the call is cross-origin and depends on the backend CORS allowlist.
  `BackEnd-CP/.env` **does** set `CORS_ALLOWED_ORIGINS`, which overrides the `:3000` + `:5173` pair in
  `config.yaml`, so a CORS failure in dev is that one line — ask the owner rather than editing the backend.
- `/health` and `/healthz` sit **outside** `/api/v1`. Poster files are served from `/media/...` (also outside),
  and only when the backend runs the local storage driver.

## Envelope — already handled by `unwrap()`

```jsonc
{ "code": 0, "message": "success", "data": { } }        // success
{ "code": 40001, "message": "validation failed",
  "details": { "email": "must be a valid email address" } }   // error, no data
```

- `code: 0` is success. Any other code is a 5-digit business code. `unwrap()` returns `response.data.data`.
- `data` is **omitted** when there is no payload, so `unwrap()` can legitimately return `undefined`.
- `details` is a **flat** `Record<string, string>` keyed by the **json field name** — exactly the name to feed to
  antd `form.setFields([{ name, errors: [msg] }])`.

### Responses that are NOT the envelope

| Endpoint                         | What comes back                                                          | What the FE must do                                                           |
| -------------------------------- | ------------------------------------------------------------------------ | ----------------------------------------------------------------------------- |
| `DELETE /admin/halls/:id`        | **204, empty body**                                                      | do not `unwrap()`, do not `res.json()`                                        |
| `DELETE /users/me`               | **204, empty body**                                                      | same                                                                          |
| `DELETE /movies/:id`             | 200 `{code,message}`, no `data`                                          | type `ApiResponse<void>`, ignore the result                                   |
| `DELETE /admin/showtimes/:id`    | 200 `{code,message}`, no `data`                                          | same                                                                          |
| `PUT /movies/:id`                | full replace                                                             | send every field — omitting `trailer_url`, `cast` or `age_rating` erases them |
| `GET /payments/:provider/return` | envelope today, **303 redirect** if `payment.return_redirect_url` is set | do not assume a body                                                          |
| `GET                             | POST /payments/:provider/ipn`                                            | provider shape, `code` is a **string**                                        | gateway-only, never call it |
| SSE stream                       | `text/event-stream`                                                      | use `EventSource`, not axios                                                  |

## Status codes the UI must branch on

| Code    | HTTP | What it means for the UI                                                                                                                                                                                                 |
| ------- | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `40001` | 400  | Validation. Render `details` per field; the message is already human-readable.                                                                                                                                           |
| `40100` | 401  | Bad credentials / invalid or replayed refresh token. The client already force-logs-out.                                                                                                                                  |
| `40101` | 401  | Access token expired. **`client.ts` already refreshes and replays** — do not handle it in a screen.                                                                                                                      |
| `40300` | 403  | Wrong role, locked account, or someone else's order. Show a permission message, not a retry.                                                                                                                             |
| `40400` | 404  | Not found, **and also a wrong HTTP verb** (there is no 405 here).                                                                                                                                                        |
| `40900` | 409  | State conflict: seats taken, hold expired, duplicate email, edit lock. Refetch, then tell the user.                                                                                                                      |
| `41300` | 413  | Body too large.                                                                                                                                                                                                          |
| `42800` | 428  | **Cannot currently happen.** The terms gate is unreachable: `account.terms_version` never reaches the backend config, so the gate condition is always false. Do not build an accept-terms screen — see the gotchas file. |
| `42900` | 429  | Rate limit or login lockout. `details.retry_after_seconds` is present for those two — but **absent** for the SSE stream limit, so read it defensively.                                                                   |
| `50000` | 500  | Genuine server error **or** a malformed request body / non-numeric `?page`. Do not auto-retry blindly.                                                                                                                   |
| `50300` | 503  | The DB guard tripped; every `/api/v1` route answers this. Retryable.                                                                                                                                                     |

There is **no 422** anywhere.

## Pagination

Request `{ page, page_size, search }` (defaults 1 / 10, `page_size` max **100**) as axios `params`.
Paged response: `data: { items: T[], meta: { page, page_size, total, total_pages } }`.

**Not every list is paged.** These return a bare array as `data`: `GET /showtimes`,
`GET /movies/:id/showtimes`, `GET /halls/:id/seats`, `GET /admin/halls/:id/seats`, `GET /admin/halls/:id/prices`,
`GET /admin/hall-templates`, `GET /payments/providers`, `GET /staff/showtimes/:id/tickets`.

`page_size` over 100 is a **400 / 40001**, not a silent clamp — `useListQuery()` already caps it.

## The three operator endpoints added 2026-09-18

| Endpoint                   | Roles          | Shape                                                                                                                                                                                                                                                                                                                                                        |
| -------------------------- | -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `GET /admin/showtimes`     | admin + staff  | paged `dto.ShowtimeResponse`. Filters `movie_id`, `hall_id`, `status`, `date`, `from`, `to`, `search` (movie title or hall name), `sort` = `start_at\|created_at`, `order`. Unlike `GET /showtimes` it keeps closed showtimes, draft/ended movies, past dates and halls with no full price set.                                                              |
| `GET /admin/showtimes/:id` | admin + staff  | one `dto.ShowtimeResponse`; does not require the showtime to be on sale.                                                                                                                                                                                                                                                                                     |
| `GET /admin/orders`        | **admin only** | paged `dto.AdminOrderListItem` = the customer's own order shape plus `sold_via`, `seats` and `customer`. Filters `status`, `payment_status`, `sold_via`, `showtime_id`, `movie_id`, `user_id`, `date`, `from`, `to`, `sort` = `created_at\|paid_at\|total_amount\|start_at`, `order`; `search` covers booking id, customer email/name/phone and movie title. |
| `GET /admin/stats`         | **admin only** | `{movies, showtimes, bookings, users}` — plain counts, NOT `/admin/overview`'s daily aggregate.                                                                                                                                                                                                                                                              |

Two traps in `/admin/orders`, both seen at runtime:

- A **counter sale has no account**, so `customer` carries only `full_name` and `phone`, with no `user_id`.
  Type `customer` as all-optional or a counter row will read as corrupt.
- `payment` is the attempt the order already **carries**. An unpaid hold with an open checkout has none, and
  `payment_status` will never match it. For live reconciliation use `GET /staff/orders/:id`.

Staff get **403** on `/admin/orders` and `/admin/stats`. They read one order at a time via
`GET /staff/orders/:id`, which admin can call too — so a detail link from the admin list can point there.

## Seats — the id trap that will cost you an afternoon

`SeatMapSeat` carries **two** ids: `id` is the hall seat, `showtime_seat_id` is the row for this showtime.
`POST /orders/hold` takes **`showtime_seat_id` values** in `seat_ids`, and the ids in an SSE `seats[]` event are
also showtime-seat ids. Sending `id` answers 400/40001 with `details.seat_id` — "seat does not belong to this
showtime".

Also true of the seat map and the hold:

- A booking is capped at **10 seats** (`max_seats_per_booking`); over that is `ErrSeatLimitExceeded` with
  `details.max`.
- A seat with `is_gap: true` is **not sellable** (`ErrSeatNotSellable`); taken seats answer `ErrSeatTaken`. Both
  list the offending labels in `details.seats`.
- `col_span` exists for couple seats — a seat can occupy more than one column.
- `SeatMapResponse` carries `screen_position` and `aisle_after_cols` (an empty array, not null, when there are no
  aisles) but **no row/column count**: derive the grid from the seats themselves. It also carries `movie_title`,
  `hall_name`, `age_rating`, `start_at`, `end_at`, `status` and `prices` (a **map** of seat_type -> price), so one
  request is enough to render the whole booking screen header.
- Verified shape of one seat: `{id, showtime_seat_id, label, row_label, col_number, seat_type, is_gap, col_span,
status, price}`.

## Formats

- **Time**: RFC3339, but the offset is **not uniform** — verified at runtime 2026-09-18: `GET /showtimes` returns
  `2026-09-18T17:00:00+07:00` while `POST /admin/showtimes` returns the same instant as `2026-09-18T10:00:00Z`.
  So parse every timestamp as an instant; never slice or compare the offset string. Render through
  `src/utils/format.ts`.
  **Trap**: `format.ts` uses plain `dayjs(value)` and the app installs **no** `dayjs/plugin/utc` or
  `/timezone` — so a `19:00+07:00` showtime renders in the _viewer's_ local zone (12:00 on a UTC machine or in
  CI). If a screen must show cinema-local time, that plugin pair has to be added first; say so rather than
  quietly rendering the wrong hour.
- **Date-only strings**: `movie.release_date` and the report/staff `date` / `from` / `to` params are plain
  `YYYY-MM-DD` strings in both directions. Submit `dayjs.format('YYYY-MM-DD')`.
- **Money**: `int64` **whole VND** — not minor units, no decimals, and there is **no currency field**. All
  formatting is the frontend's job.
- **IDs**: UUID strings. Exception: a ticket `code` is a non-UUID varchar(32).

## Enums

**The canonical list of all 24 enums with their exact literals is
`../../../../BackEnd-CP/.claude/context/contract.md`.** Read it there instead of copying values here — a second
copy in this repo drifted within one commit. If the two ever disagree, the Go code wins.

Each enum the UI displays needs a `Record<Union, string>` label map plus an i18n key in **both** locale files.
The dynamic key shape already in use is `movie.status<Capitalized>`.

## Auth, as the client already implements it

- `Authorization: Bearer <access_token>`, **no cookies**. The request interceptor attaches it.
- Refresh: `POST /auth/refresh` with `{ refresh_token }` in the **body**. Rotating — the old one dies, and a
  replay revokes the whole family. Single-flight + queue is already implemented; do not add a second path.
- A role change server-side invalidates the token within ~30s (`40100`), forcing a fresh login.
- `POST /auth/logout` exists and is **currently never called** — see the gotchas file.
- `PUT /users/me/password` **returns a fresh token pair and revokes every other session** — the client must
  replace both stored tokens, or the user is logged out on their next request.

## Realtime seats (SSE)

Two steps: `GET /events/token` mints a short-lived token, then open the stream it points at. Events are
`connected` and `seats`; the `seats` payload is `{ showtime_id, hall_id, seats: [{ id, status }] }` where `id` is
a **showtime_seat id**. The 429 here carries no `retry_after_seconds`.

Two mechanics to get right:

- `stream_url` comes back **relative** (`/api/v1/events/shows/<id>?token=<t>`). Resolve it with
  `new URL(stream_url, API_BASE_URL)` — concatenating it onto `API_BASE_URL`, which already ends in `/api/v1`,
  produces `/api/v1/api/v1/...` and a 404.
- The stream takes **no** `Authorization` header by design (the query token is the credential), so `EventSource`
  is the one sanctioned bypass of `apiClient`. Tokens expire and the stream is force-closed after ~30 min, so
  plan to re-token and reconnect.

Nothing in this app consumes SSE yet.
