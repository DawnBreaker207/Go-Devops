# BackEnd-CP endpoint reference

Generated 2026-09-18, **re-verified against the source 2026-09-22** after a large feature pull. The **Guard
column is parsed directly from `internal/router/router.go`** (group nesting + per-route middleware), not from
prose. Request/response/notes come from the handlers and DTOs.

**108 `/api/v1` operations** as of 2026-09-22, up from 78. `docs/swagger.json` documents **102** of them, so
**6 are still undocumented** — they remain the `@Router`-less aliases listed at the bottom of this file.
Treat `router.go` as the only complete list, always.

**13 of those 108 were added on 2026-09-22 after the pull** and are marked `NEW 09-22b` below: five for the
concession catalogue (`/admin/concessions`, operator scope) and seven for discount codes (five admin-only under
`/admin/discounts`, plus `POST` and `DELETE /orders/:id/discount` for customers), and the public
`GET /pricing`.

The 2026-09-22 pull added **17 operations** that this file did not describe; they are marked `NEW 09-22` in
their sections below. Four new schema migrations came with them (`000006_session_devices`,
`000007_catalog_lifecycle`, `000008_combo`, `000009_notification_preferences`), so a database created before
that pull is **four migrations behind** and every one of those endpoints will fail against it.

## Re-verify these numbers before trusting them

Every count in the config is reproducible. Run these from the `BackEnd-CP` root; the expected values held on
**2026-09-22**. If one disagrees, this file is stale — fix it before building on it.

```bash
# Full operation list, group prefixes resolved. This is the check that matters.
python3 - <<'EOF'
import re
src = open('internal/router/router.go').read()
pre = {'v1':'/api/v1','auth':'/api/v1/auth','public':'/api/v1','protected':'/api/v1',
       'movies':'/api/v1/movies','catalog':'/api/v1/admin','orders':'/api/v1/orders',
       'tickets':'/api/v1/tickets','comboOrders':'/api/v1/combo-orders','staff':'/api/v1/staff',
       'halls':'/api/v1/halls','admin':'/api/v1/admin'}
pat = re.compile(r'\b(\w+)\.(GET|POST|PUT|PATCH|DELETE|Match)\(\s*(\[\][^)]*?\}\s*,\s*)?"([^"]*)"')
ops = set()
for m in pat.finditer(src):
    g, v, path = m.group(1), m.group(2), m.group(4)
    if g not in pre: continue
    full = (pre[g] + path).replace('//', '/').rstrip('/') or pre[g]
    ops |= {('GET', full), ('POST', full)} if v == 'Match' else {(v, full)}
print(len([o for o in ops if o[1].startswith('/api/v1')]))   # 95 on 2026-09-22
EOF
find . -name '*.go' -not -path './docs/*' | wc -l                      # 153 Go files
grep -cE '^\s*Err[A-Za-z0-9]+\s*=\s*(BadRequest|Validation|Unauthorized|TokenExpired|Forbidden|NotFound|Conflict|PayloadTooLarge|PreconditionRequired|TooManyRequests|Internal|BadGateway|ServiceUnavailable)\(' pkg/errors/errors.go   # 52
python3 -c "import json;d=json.load(open('docs/swagger.json'));print(sum(len([k for k in v if k in ('get','post','put','patch','delete')]) for v in d['paths'].values()))"   # 89 documented vs 95 real -> 6 aliases
ls migrations/schema/*.up.sql | wc -l                                  # 9 migrations
```

The Guard column is produced by walking the group nesting in `router.go` and unioning each group's `.Use(...)`
middleware with the route's own arguments. Do that, not a prose reading — an earlier hand-classified version of
this table labelled 22 admin routes `public`.

## How to read the Guard column

- Every `/api/v1/**` route ALSO inherits, in order: `RequestID -> Recovery -> SecurityHeaders -> BodyLimit ->
  Logger -> CORS`, then `DBGuard(db,500ms) -> NoStore`. The column lists only what is specific to the route.
- `public` = no credential. `optional-jwt` = `OptionalAuth`: no header passes as a guest, a present-but-invalid
  header still 401s. `jwt` = `middleware.Auth`, **any** role.
- `jwt+admin/staff` = `RequireRoles(RoleAdmin, RoleStaff)`. Exact, case-sensitive, **no hierarchy** — `admin`
  does NOT imply `staff`. `jwt+staff/admin` is the same set in the router's own argument order.
- `stream-token` = no JWT; the `?token=` realtime token validated by `sse.TokenStore` is the whole credential.
- `sig` = the handler verifies the provider's HMAC signature instead of a JWT.
- `rl:auth|hold|public|events` = the token bucket that applies. `audit:<action>` = `middleware.Audit` writes a
  failure row for any status >= 400; the service writes the success row with the **same** action string.
- Caveat: on the `/staff` and `/admin` groups the role check is a group-level `.Use(...)`, so it runs BEFORE the
  route-level `Audit` — a 403 on those two groups is therefore NOT audited, unlike everywhere else.

## outside /api/v1  (5)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `GET` | `/health` | public | - | handlers.HealthStatus {status, service, database} | 200 503 |
| `GET` | `/healthz` | public | - | - (data omitted; response.OK(c, nil)) | 200 503 |
| `GET\|HEAD` | `/media/*filepath` | public | path (served from the local media dir) | - raw image bytes served from disk | 200 404 |
| `ANY` | `/mock-gateway/*path` | public | HTML form: GET /mock-gateway/checkout?ref=... and POST /mock-gateway/checkout/{ref} with form field mode=pay\|pay_no_ipn\|pay_wrong_amount\|decline\|cancel | text/html checkout page, not JSON | 200 404 |
| `GET` | `/swagger/*any` | public | - | Swagger UI HTML/JSON assets | 200 404 |

Notes:

- `GET /health` — Unversioned probe; 2s DB ping timeout. Response is not Cache-Control: no-store (NoStore only on /api/v1).
- `GET /healthz` — Liveness/readiness probe for infra; body carries no payload.
- `GET|HEAD /media/*filepath` — Only mounted when the local disk driver is configured (LocalDir non-empty); with the cloudinary driver poster URLs point at Cloudinary instead.
- `ANY /mock-gateway/*path` — Registered per Simulator provider; path is "/"+provider name+"-gateway" . In-memory only, lost on restart.
- `GET /swagger/*any` — Registered only when cfg.App.Env != "production" . Spec comes from the generated docs package.

## /api/v1/health  (1)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `GET` | `/api/v1/health` | public | - | handlers.HealthStatus | 200 503 \| 50300 |

Notes:

- `GET /api/v1/health` — Two different 503 shapes: DBGuard envelope code 50300 vs handler code 503. Sends Cache-Control: no-store.

## /api/v1/healthz  (1)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `GET` | `/api/v1/healthz` | public | - | - (data omitted) | 200 503 \| 50300 |

Notes:

- `GET /api/v1/healthz` — Versioned twin of /healthz; DBGuard can short-circuit before the handler ever pings.

## /api/v1/auth  (7)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `POST` | `/api/v1/auth/forgot-password` | public rl:auth audit:auth.forgot_password | dto.ForgotPasswordRequest (json body: email) | - (response.OK(c, nil), data omitted) | 200 400 429 500 503 \| 40001 42900 50000 |
| `POST` | `/api/v1/auth/login` | public rl:auth audit:auth.login | dto.LoginRequest (json body: email, password min 6; ClientIP filled server-side) | dto.LoginResponse = dto.TokenResponse{access_token, refresh_token, token_type, expires_in} + user dto.UserResponse | 200 400 401 403 428 429 500 503 \| 40001 40100 40300 42800 42900 |
| `POST` | `/api/v1/auth/logout` | public rl:auth audit:auth.logout | dto.LogoutRequest (json body: refresh_token) | - (response.OK(c, nil), data omitted) | 200 400 401 429 500 503 \| 40001 40100 40101 42900 |
| `POST` | `/api/v1/auth/refresh` | public rl:auth audit:auth.refresh | dto.RefreshRequest (json body: refresh_token) | dto.TokenResponse | 200 400 401 403 429 500 503 \| 40001 40100 40101 40300 42900 |
| `POST` | `/api/v1/auth/register` | public rl:auth audit:auth.register | dto.RegisterRequest (json body: email, password 6-72, full_name 2-255) | dto.UserResponse | 201 400 409 413 429 500 503 \| 40001 40900 41300 42900 50000 50300 |
| `POST` | `/api/v1/auth/reset-password` | public rl:auth audit:auth.reset_password | dto.ResetPasswordRequest (json body: token, new_password 6-72) | - (response.OK(c, nil), data omitted) | 200 400 403 429 500 503 \| 40000 40001 40300 42900 |
| `POST` | `/api/v1/auth/terms-accept` | public rl:auth audit:auth.accept_terms | dto.LoginRequest (json body: email + password) | dto.LoginResponse | 200 400 401 403 429 500 503 \| 40001 40100 40300 42900 |

Notes:

- `POST /api/v1/auth/forgot-password` — Enumeration-safe: burns a bcrypt compare for unknown emails. Mail failure only logs; reset link TTL from auth resetTTL.
- `POST /api/v1/auth/login` — The listed 428 is **unreachable as wired** (`account.terms_version` never reaches the config), so no client needs an /auth/terms-accept branch today. The 429 Retry-After header mirrors `details.retry_after_seconds`; the lockout is 5 wrong passwords per email+IP for 5m.
- `POST /api/v1/auth/logout` — Revokes the whole token family; an already-revoked or unknown token still answers 200 .
- `POST /api/v1/auth/refresh` — Rotating refresh: reuse of a used token revokes the whole family and answers 401 .
- `POST /api/v1/auth/register` — Always creates role=customer, active=true . Email lowercased/trimmed. 429 sets Retry-After header.
- `POST /api/v1/auth/reset-password` — Single-use token (sha256 of lowercased value); on success revokes every refresh token of the user .
- `POST /api/v1/auth/terms-accept` — Re-verifies credentials, stores accepted terms version, then issues a token pair. Swagger's documented 428 is never returned.

## /api/v1/users  (9)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `DELETE` | `/api/v1/users/me` | jwt+customer audit:users.delete_me | dto.DeleteAccountRequest (json body: password) | - (204 No Content, empty body via c.Status) | 204 400 401 403 404 409 500 503 \| 40001 40100 40300 40400 40900 |
| `GET` | `/api/v1/users/me` | jwt | - | dto.UserResponse | 200 401 403 404 500 503 \| 40100 40101 40300 40400 50300 |
| `PUT` | `/api/v1/users/me` | jwt audit:users.update_profile | dto.UpdateProfileRequest (json body: full_name 2-255 required, phone optional max 20) | dto.UserResponse | 200 400 401 403 404 413 500 503 \| 40001 40100 40101 40300 40400 41300 |
| `PUT` | `/api/v1/users/me/password` | jwt audit:users.change_password | dto.ChangePasswordRequest (json body: current_password, new_password 6-72) | dto.TokenResponse | 200 400 401 403 404 413 500 503 \| 40001 40100 40101 40300 40400 41300 50000 |
| `GET` | `/api/v1/users/me/sessions` | jwt — **NEW 09-22** | dto.SessionListQuery (query: device_id omitempty max=255) | `[]dto.SessionResponse` — **BARE ARRAY** | 200 400 401 403 500 503 \| 40001 40100 40101 40300 |
| `DELETE` | `/api/v1/users/me/sessions/:id` | jwt audit:users.revoke_session — **NEW 09-22** | - (path param id) | - | 200 401 403 404 500 503 \| 40100 40101 40300 40400 |
| `GET` | `/api/v1/users/me/transactions` | jwt — **NEW 09-22** | dto.PageQuery | response.Paged{items=[]dto.TransactionResponse} | 200 400 401 403 500 503 \| 40001 40100 40101 40300 |
| `GET` | `/api/v1/users/me/notification-preferences` | jwt — **NEW 09-22** | - | dto.NotificationPreferenceResponse (booking_reminders, promo_offers — both plain bool, both REQUIRED) | 200 401 403 500 503 \| 40100 40101 40300 |
| `PUT` | `/api/v1/users/me/notification-preferences` | jwt audit:users.update_notification_preferences — **NEW 09-22** | dto.UpdateNotificationPreferenceRequest — replaces BOTH flags at once | dto.NotificationPreferenceResponse | 200 400 401 403 413 500 503 \| 40001 40100 40101 40300 41300 |

Notes:

- `DELETE /api/v1/users/me` — Irreversible: scrubs PII, sets deleted_at, expires pending holds, deletes refresh + reset tokens. Staff/admin get 403.
- `GET /api/v1/users/me` — Auth re-reads account status (30s cache); a role change since the token was minted answers 401.
- `PUT /api/v1/users/me` — Only full_name and phone are writable; empty phone clears it. Audit row written inside the transaction .
- `PUT /api/v1/users/me/password` — Revokes every other session and returns a fresh token pair; the client must replace both stored tokens.
- `GET /api/v1/users/me/sessions` — **Bare array, not paged.** `is_current` is true ONLY when the caller passes
  its own `device_id` and it matches; omit the param and every row reads `is_current: false`. `user_agent` and
  `last_used_at` are omitempty.
- `DELETE /api/v1/users/me/sessions/:id` — Revokes one device's refresh-token family. Revoking your OWN session
  does not invalidate the access token you are holding; it dies at its next refresh (TTL 900s).
- `GET /api/v1/users/me/transactions` — Payment history scoped to the caller. Carries `booking_id` and
  `showtime_id` per row, so it links back without a second call. `paid_amount` / `paid_at` / `refunded_at` are
  omitempty pointers — absent, not zero.
- `PUT /api/v1/users/me/notification-preferences` — Replaces both flags; there is no partial update. Both false
  is a legal state ("send me nothing").

## /api/v1/movies  (6)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `GET` | `/api/v1/movies` | optional-jwt rl:public | dto.MovieListQuery (query; embeds dto.PageQuery: page, page_size, search + status, genre, sort, order) | response.Paged{items=[]dto.MovieResponse} | 200 400 401 403 429 503 500 \| 40001 40100 40101 40300 42900 50000 50300 |
| `POST` | `/api/v1/movies` | jwt+admin/staff audit:admin.create_movie | dto.MovieRequest (json body) | dto.MovieResponse (201 Created) | 201 400 401 403 413 503 500 \| 40001 40100 40101 40300 41300 50000 50300 |
| `DELETE` | `/api/v1/movies/:id` | jwt+admin/staff audit:admin.delete_movie | - (path param id) | - (200 with {code:0,message:"deleted"}, no data ) | 200 404 409 400 401 403 503 500 \| 40000 40100 40101 40300 40400 40900 50000 50300 |
| `GET` | `/api/v1/movies/:id` | optional-jwt rl:public | - (path param id) | dto.MovieResponse | 200 404 400 401 403 429 503 500 \| 40000 40100 40101 40300 40400 42900 50000 50300 |
| `PUT` | `/api/v1/movies/:id` | jwt+admin/staff audit:admin.update_movie | dto.MovieRequest (json body, full replace) | dto.MovieResponse | 200 400 404 409 401 403 413 503 500 \| 40001 40100 40101 40300 40400 40900 41300 50000 50300 |
| `GET` | `/api/v1/movies/:id/showtimes` | optional-jwt rl:public | - (path param id; optional query date=YYYY-MM-DD read with c.Query) | []dto.ShowtimeListItem | 200 404 400 401 403 429 503 500 \| 40000 40001 40100 40101 40300 40400 42900 50000 50300 |

Notes:

- `GET /api/v1/movies` — Drafts hidden unless admin/staff token ; anonymous results Redis-cached; sort default created_at desc
- `POST /api/v1/movies` — Required: title, genre, duration, director, release_date, status(draft|showing|ended); empty age_rating defaults to "P"; busts catalog cache
- `DELETE /api/v1/movies/:id` — Soft delete; refused while open showtimes are still to come; returns 200 not 204
- `GET /api/v1/movies/:id` — Anonymous read cached in Redis under movie:<id>
- `PUT /api/v1/movies/:id` — All MovieRequest fields required each call; duration change blocked while any showtime (even closed) is still to come
- `GET /api/v1/movies/:id/showtimes` — Movie not "showing" returns empty array ; only open, not-yet-started shows in halls priced for all 4 seat types

## /api/v1/pricing  (1)  — NEW 09-22b

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `GET` | `/api/v1/pricing` | public rl:public optional-auth | - | dto.PublicPriceListResponse (`from_price` + `halls[]`, each with all 4 seat-type prices) | 200 429 500 503 \| 42900 50000 50300 |

Notes: the only ANONYMOUS read of `hall_prices`. Applies the same gate as the customer showtime query —
`active = TRUE` and all four seat types priced above 0 — so a hall nobody can book never appears. An empty
`halls` therefore means "nothing is bookable", not "prices are unconfigured", and `from_price` is 0 there.

## /api/v1/showtimes  (1)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `GET` | `/api/v1/showtimes` | optional-jwt rl:public | - (optional query date=YYYY-MM-DD via c.Query) | []dto.ShowtimeListItem | 200 400 401 403 429 503 500 \| 40001 40100 40101 40300 42900 50000 50300 |

Notes:

- `GET /api/v1/showtimes` — Customer picker, NOT an admin list: requires movie showing + showtime open + start_at >= now + a complete hall price set. Its one param `?date=YYYY-MM-DD` clamps the result to exactly ONE calendar day, defaulting to today. No paging, no hall/movie/status filter.

## /api/v1/shows  (1)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `GET` | `/api/v1/shows/:id/seats` | jwt | - (path param id = showtime id) | dto.SeatMapResponse | 200 404 400 401 403 503 500 \| 40000 40100 40101 40300 40400 50000 50300 |

Notes:

- `GET /api/v1/shows/:id/seats` — Only while open, movie showing and start_at in future; seat status available|held|sold, gaps and col_span included

## /api/v1/halls  (3)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `PUT` | `/api/v1/halls/:id/prices` | jwt+admin/staff audit:admin.set_hall_prices | dto.PriceRequest (json body) | []dto.HallPriceResponse (4 rows, models.AllSeatTypes order) | 200 400 404 401 403 413 503 500 \| 40001 40100 40101 40300 40400 41300 50000 50300 |
| `GET` | `/api/v1/halls/:id/seats` | jwt | - (path param id) | []dto.SeatResponse | 200 404 400 401 403 503 500 \| 40000 40100 40101 40300 40400 50000 50300 |
| `PUT` | `/api/v1/halls/:id/seats/:seatId` | jwt+admin/staff audit:admin.update_hall_seat | dto.SeatUpdateRequest (json body) | dto.SeatResponse | 200 400 404 409 401 403 413 503 500 \| 40001 40100 40101 40300 40400 40900 41300 50000 50300 |

Notes:

- `PUT /api/v1/halls/:id/prices` — Alias of PUT /api/v1/admin/halls/:id/prices, identical guards and handler
- `GET /api/v1/halls/:id/seats` — Contract alias of the admin route, deliberately open to customers ; no audit middleware
- `PUT /api/v1/halls/:id/seats/:seatId` — Alias of PUT /api/v1/admin/halls/:id/seats/:seatId, identical guards and handler

## /api/v1/events  (2)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `GET` | `/api/v1/events/shows/:id` | stream-token | path param id (showtime) + query param token | - text/event-stream: "retry: 3000", event connected {showtime_id, hall_id}, event seats handlers.seatsPayload{showtime_id, hall_id, seats []sse.SeatUpdate} | 200 401 429 503 \| 40100 42900 50300 |
| `GET` | `/api/v1/events/token` | jwt rl:events | query param show_id (read with c.Query, no DTO struct) | gin.H{token string, expires_in int seconds, stream_url string} - anonymous map, no DTO | 200 400 401 403 404 429 500 503 \| 40001 40100 40101 40300 40400 42900 |

Notes:

- `GET /api/v1/events/shows/:id` — Seat updates debounced ~100ms, ": ping" every 15s, per-write 10s deadline, stream force-closed after 30min - re-token and reconnect.
- `GET /api/v1/events/token` — Token TTL 30s , reusable for reconnects, bound to showtime+hall. stream_url is prebuilt with the token.

## /api/v1/orders  (12)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `GET` | `/api/v1/orders` | jwt | dto.PageQuery (query: page, page_size, search) — ShouldBindQuery, then Normalize | response.Paged{items=[]dto.OrderStatusResponse} | 200 400 401 403 413 500 503 \| 40001 40100 40101 41300 50000 50300 |
| `GET` | `/api/v1/orders/:id` | jwt+customer | - (path param id only) | dto.OrderDetailResponse | 200 401 403 404 500 503 \| 40100 40101 40300 50000 50300 |
| `POST` | `/api/v1/orders/:id/cancel` | jwt+customer audit:orders.cancel | - (path param id only) | dto.OrderStatusResponse | 200 401 403 404 409 500 503 \| 40100 40101 40300 50000 50300 |
| `POST` | `/api/v1/orders/:id/confirm` | jwt+customer audit:orders.confirm | - (path param id only; no body read) | dto.OrderDetailResponse (OrderStatusResponse + tickets[] with code for the QR) | 200 401 403 404 409 500 503 \| 40100 40101 40300 50000 50300 |
| `POST` | `/api/v1/orders/:id/pay` | jwt+customer audit:orders.pay | dto.PayRequest (json body: provider omitempty max=32; client_ip is server-filled, json:"-") (; handler sets ClientIP at ) | dto.PayResponse (payment_id, provider, txn_ref, redirect_url, expires_at) | 200 400 401 403 404 409 413 502 500 503 \| 40001 40100 40101 40300 41300 50000 50200 50300 |
| `GET` | `/api/v1/orders/:id/status` | jwt+customer | - (path param id only) | dto.OrderStatusResponse | 200 401 403 404 500 503 \| 40100 40101 40300 50000 50300 |
| `GET` | `/api/v1/orders/:id/tickets` | jwt+customer | - (path param id only) | dto.OrderDetailResponse | 200 401 403 404 500 503 \| 40100 40101 40300 50000 50300 |
| `POST` | `/api/v1/orders/hold` | jwt+customer rl:hold audit:orders.hold | dto.HoldRequest (json body: show_id required, seat_ids required min=1, idempotency_key omitempty min=1 max=128) | dto.HoldResponse | 201 400 401 403 404 409 413 429 500 503 \| 40001 40100 40101 40300 41300 42900 50000 50300 |
| `POST` | `/api/v1/orders/init` | jwt+customer rl:hold audit:orders.init — **NEW 09-22** | dto.InitRequest (json body: show_id required) | dto.InitResponse (booking_id, showtime_id, expires_at, **ttl_seconds**, **reused**) | 200 400 401 403 404 409 429 500 503 \| 40001 40100 40101 40300 40400 40900 42900 |
| `POST` | `/api/v1/orders/:id/refresh` | jwt+customer rl:hold audit:orders.refresh — **NEW 09-22** | - (path param id) | dto.RefreshResponse (the hold's new deadline) | 200 401 403 404 409 429 500 503 \| 40100 40101 40300 40400 40900 42900 |
| `POST` | `/api/v1/orders/:id/discount` | jwt+customer audit:orders.apply_discount — **NEW 09-22b** | dto.ApplyDiscountRequest (json body: code required 1-32, matched UPPERCASE) | dto.DiscountAppliedResponse (booking_id, code, **subtotal**, discount, **payable**) | 200 400 401 403 404 409 500 503 \| 40001 40100 40101 40300 40400 40900 |
| `DELETE` | `/api/v1/orders/:id/discount` | jwt+customer audit:orders.remove_discount — **NEW 09-22b** | - (path param id) | dto.DiscountAppliedResponse (discount 0, payable back to subtotal) | 200 400 401 403 404 409 500 503 \| 40001 40100 40101 40300 40400 40900 |

Notes:

- `GET /api/v1/orders` — Own orders only, by CurrentUserID. Payment summary is always nil here; showtime filled per row .
- `GET /api/v1/orders/:id` — Reconciles then returns tickets[]; each ticket.code is the string the frontend renders as the QR.
- `POST /api/v1/orders/:id/cancel` — Releases seats and broadcasts SSE 'available'. Leaves an open provider checkout alone; late money is refunded.
- `POST /api/v1/orders/:id/confirm` — Idempotent; reconciles with provider first, then issues tickets. A refunded settle answers 409, not 200.
- `POST /api/v1/orders/:id/pay` — Empty provider uses config default ("mock"). Re-pay same provider returns the open checkout; frontend must redirect to redirect_url.
- `GET /api/v1/orders/:id/status` — Reconciles with the provider on every call (may settle the booking); poll this while waiting for payment.
- `GET /api/v1/orders/:id/tickets` — Alias: identical handler and full payload as GET /api/v1/orders/:id, not a tickets-only array.
- `POST /api/v1/orders/hold` — 201 not 200. Idempotency-key body field. One pending hold per user+show; replacing keeps old expires_at. TTL 10min default.
- `POST /api/v1/orders/init` — **Opens a pending order BEFORE any seat is picked**, which is why the booking
  screen can show a live countdown on first paint. It REUSES the caller's existing pending order for the same
  show rather than making a second one (`reused: true` says so), under the same lock order as a hold
  (user-show, user, clock, showtime). Refuses with `ErrShowtimeClosed` when the showtime is not `open`, has
  already started, or its movie is not `showing` — so a stale link cannot open an order.
  `ttl_seconds` exists so the client can run a ticker without doing clock arithmetic against a server
  timestamp, which is the bug this avoids: the same instant arrives with different offsets from different
  endpoints.
- `POST /api/v1/orders/:id/refresh` — Heartbeat that extends a pending hold. **Ownership failure here is 403
  `"this order belongs to another user"`, NOT 404** — the opposite choice from `GET /tickets/:id/qr`, which
  hides existence behind a 404. Do not assume one convention across the API. Also 409 (`ErrBookingNotPending`)
  once the order is paid or settled.

## /api/v1/tickets  (2)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `POST` | `/api/v1/tickets/:id/redeem` | jwt+staff/admin audit:staff.redeem_ticket | dto.RedeemRequestBody (json body: showtime_id required) + path id = ticket id OR QR code | dto.RedeemResponse (status, ticket_id, showtime_id, movie_title, age_rating, hall_name, seat_label, start_at, checkin_opens_at, checkin_closes_at) | 200 400 401 403 413 500 503 \| 40001 40100 40101 40300 41300 50000 50300 |
| `GET` | `/api/v1/tickets/:id/qr` | **jwt only — NO RequireRoles** — **NEW 09-22** | - (path param id = ticket id) | dto.TicketQRResponse (ticket_id, code, **qr_base64** — a base64 PNG, all three plain/REQUIRED) | 200 401 403 404 500 503 \| 40100 40101 40300 40400 50000 |

Notes:

- `POST /api/v1/tickets/:id/redeem` — Check-in gate: refusals are 200 with a verdict, not HTTP errors. Window default -30min/+20min around start_at.
- `GET /api/v1/tickets/:id/qr` — **The route carries no role guard; ownership is enforced in the SERVICE.**
  `bookingService.TicketQR` compares `row.UserID` against the caller and, for a non-staff caller looking at
  someone else's ticket, returns **`ErrTicketNotFound` (404) rather than 403** — deliberately, so the API never
  confirms that another user's ticket exists. Staff and admin may read any ticket. The server renders the PNG
  (`GenerateQRPNG(row.Code)`) and hands back base64, so the client does not need a QR library; `code` is still
  returned for a text fallback.

## /api/v1/combos  (1)  — NEW 09-22

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `GET` | `/api/v1/combos` | **public** (rl:public + OptionalAuth) | - | `[]dto.ComboResponse` — **BARE ARRAY** | 200 401 429 500 503 \| 40100 42900 50300 |

Notes:

- `GET /api/v1/combos` — The concession catalogue behind step 2 of the booking wizard. Bare array, not paged.
  `description` and `image_url` are omitempty; `id`, `name`, `price` (int64 whole VND) and `active` are always
  present.
- **There is NO write path for this catalogue.** It reads the `concession_items` table (created by
  `000008_combo`), for which there is no admin endpoint and no seed file. On a fresh database the array is
  empty, so the combo step renders with nothing to choose, and the only way to populate it today is direct
  SQL. Treat that as a known gap, not a bug to chase in the frontend.

## /api/v1/combo-orders  (2)  — NEW 09-22

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `POST` | `/api/v1/combo-orders` | jwt (**no RequireRoles**) audit:combo_orders.create | dto.CreateComboOrderRequest (json: `booking_id` omitempty uuid, `items` required min=1 dive of {combo_id required uuid, quantity required min=1 max=20}) | the placed order with priced lines (`[]dto.ComboOrderItemResponse`: combo_id, combo_name, quantity, unit_price, subtotal) | 200/201 400 401 403 404 413 500 503 \| 40001 40100 40101 40300 40400 41300 |
| `GET` | `/api/v1/combo-orders/me` | jwt (**no RequireRoles**) | dto.PageQuery | the caller's own combo orders | 200 400 401 403 500 503 \| 40001 40100 40101 40300 |

Notes:

- A combo order is **independent of the ticket booking**. `booking_id` is an optional correlation only ("pick up
  with your tickets"); the DTO comment is explicit that a combo order never touches the booking. So it does not
  extend a hold, does not appear in the booking total, and is not covered by the booking's payment.
- Line prices are resolved server-side from the catalogue; the client never sends a price.
- Neither route is customer-only, unlike the `/orders` tree — any signed-in role can place one.

## /api/v1/payments  (4)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `GET` | `/api/v1/payments/:provider/ipn` | public sig | provider-defined; mock decodes a JSON body capped at 64KiB (txn_ref, amount, status, gateway_txn_id, signature) | provider-shaped, NOT the response.Body envelope: mock writes {"code","message"} | 404 200 401 400 503 \| 40400 50300 |
| `POST` | `/api/v1/payments/:provider/ipn` | public sig | provider-defined; mock decodes a JSON body capped at 64KiB (txn_ref, amount, status, gateway_txn_id, signature) | provider-shaped, NOT the response.Body envelope: mock writes {"code","message"} | 404 200 401 400 503 413 \| 40400 41300 50300 |
| `GET` | `/api/v1/payments/:provider/return` | public sig | query params, provider-defined; mock requires ref, status, sig | dto.PaymentReturnResponse (booking_id, booking_status, booking_reason, payment_id, payment_status) when payment.return_redirect_url is empty; otherwise no body | 200 303 400 401 404 500 503 \| 40001 40100 40400 50000 50300 |
| `GET` | `/api/v1/payments/providers` | jwt | - | []dto.PaymentProviderResponse (name, display_name, default) | 200 401 403 503 \| 40100 40101 50300 |

Notes:

- `GET /api/v1/payments/:provider/ipn` — Gateway-to-server only; never called by the frontend. Registered via v1.Match for GET and POST .
- `POST /api/v1/payments/:provider/ipn` — Idempotent by txn_ref: duplicates ack 02 and change nothing. A DB failure answers 503 so the gateway retries.
- `GET /api/v1/payments/:provider/return` — Browser landing page; 303 only when payment.return_redirect_url is set (config default empty -> JSON). Return is never proof of payment.
- `GET /api/v1/payments/providers` — Always a JSON array, empty if none enabled. Feeds the provider field of POST /orders/:id/pay.

## /api/v1/staff  (9)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `GET` | `/api/v1/staff/boxoffice/day` | jwt+staff/admin | query date (raw c.Query("date")), YYYY-MM-DD, default today | dto.BoxOfficeDayResponse{date, count, total} | 200 400 401 403 500 503 \| 40001 40100 40101 40300 50000 50300 |
| `GET` | `/api/v1/staff/customers` | jwt+staff/admin | dto.PageQuery (query: page, page_size max=100, search max=255); handler forces Role=models.RoleCustomer | response.Paged{items=[]dto.UserResponse} | 200 400 401 403 500 503 \| 40001 40100 40101 40300 50000 50300 |
| `GET` | `/api/v1/staff/customers/:id` | jwt+staff/admin | - (path param id = user id) | dto.UserResponse | 200 401 403 404 500 503 \| 40100 40101 40300 50000 50300 |
| `GET` | `/api/v1/staff/customers/:id/orders` | jwt+staff/admin | path id = user id; dto.PageQuery (query: page, page_size, search) | response.Paged{items=[]dto.OrderStatusResponse} | 200 400 401 403 404 500 503 \| 40001 40100 40101 40300 50000 50300 |
| `GET` | `/api/v1/staff/dashboard` | jwt+staff/admin | query date (raw c.Query("date"), no DTO), local YYYY-MM-DD, default today | dto.StaffBoardResponse{date, showtimes[]dto.StaffShowtimeResponse} | 200 400 401 403 500 503 \| 40001 40100 40101 40300 50000 50300 |
| `POST` | `/api/v1/staff/orders` | jwt+staff/admin audit:orders.counter_sell | dto.CounterSellRequest (json body: show_id required, seat_ids required min=1, customer_name omitempty max=255, customer_phone omitempty max=20) | dto.OrderDetailResponse — booking already confirmed with issued tickets | 200 201 400 401 403 404 409 413 500 503 \| 40001 40100 40101 40300 41300 50000 50300 |
| `GET` | `/api/v1/staff/orders/:id` | jwt+staff/admin | - (path param id = booking id) | dto.OrderDetailResponse | 200 401 403 404 500 503 \| 40100 40101 40300 50000 50300 |
| `GET` | `/api/v1/staff/overview` | jwt+staff/admin | query date (raw c.Query("date")), local YYYY-MM-DD, default today | dto.StaffOverviewResponse{date, showtimes[], counter_sales_count, counter_sales_total, awaiting_checkin} | 200 400 401 403 500 503 \| 40001 40100 40101 40300 50000 50300 |
| `GET` | `/api/v1/staff/showtimes/:id/tickets` | jwt+staff/admin | path id = showtime id; query status (raw c.Query), "" \| issued \| redeemed | []dto.StaffTicketResponse (id, booking_id, seat_label, seat_type, status, updated_at) | 200 400 401 403 404 500 503 \| 40001 40100 40101 40300 50000 50300 |

Notes:

- `GET /api/v1/staff/boxoffice/day` — Walk-in counter sales only (close-day register report). `count` is the NUMBER of counter bookings, `total` is int64 whole VND — there are no minor units anywhere in this API.
- `GET /api/v1/staff/customers` — Role filter is server-forced to customer; staff/admin accounts are never listed here. search matches email, name or phone.
- `GET /api/v1/staff/customers/:id` — Never confirms a staff/admin account exists: a non-customer id answers the same 404 as a missing one.
- `GET /api/v1/staff/customers/:id/orders` — Same payload as the customer's own GET /orders (no payment summary per row); search is accepted but unused.
- `GET /api/v1/staff/dashboard` — No pagination. available = capacity - held - sold, computed server-side. Days are in the configured timezone.
- `POST /api/v1/staff/orders` — Cash sale, no account and no payment row; confirmed immediately, no ticket email. Duplicate seat_ids are rejected.
- `GET /api/v1/staff/orders/:id` — Support lookup: no ownership check (bookingService.AdminOrder), reconciles with the provider before answering.
- `GET /api/v1/staff/overview` — One call replacing /staff/dashboard + /staff/boxoffice/day; awaiting_checkin = sum(sold - checked_in).
- `GET /api/v1/staff/showtimes/:id/tickets` — Plain array, not paginated. status=issued means still waiting at the gate. No ticket code is returned.

## /api/v1/admin  (46)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `GET` | `/api/v1/admin/audit-logs` | jwt+admin | dto.AuditLogListQuery (query; embeds dto.PageQuery) - | response.Paged{items: []dto.AuditLogResponse, meta} | 200 400 401 403 500 503 \| 40001 40100 40101 40300 50000 50300 |
| `GET` | `/api/v1/admin/batch/jobs` | jwt+admin | dto.PageQuery (query: page, page_size, search) - | response.Paged{items: []models.BatchJob, meta} | 200 400 401 403 500 503 \| 40001 40100 40101 40300 50000 50300 |
| `POST` | `/api/v1/admin/batch/jobs/:name/run` | jwt+admin audit:admin.run_job | - (job name is the :name path param; no body read) | gin.H{job, run_id, status:"running", triggered_by:"manual"} | 202 404 409 401 403 500 503 \| 40100 40101 40300 40400 40900 50000 50300 |
| `GET` | `/api/v1/admin/concessions` | jwt+admin/staff — **NEW 09-22b** | dto.AdminComboListQuery (embeds PageQuery; `active` is a Go POINTER, omit for both) | response.Paged{items=[]dto.ComboResponse} — **includes INACTIVE**, unlike public `GET /combos` | 200 400 401 403 500 503 \| 40001 40100 40101 40300 |
| `GET` | `/api/v1/admin/concessions/:id` | jwt+admin/staff — **NEW 09-22b** | - (path param id) | dto.ComboResponse | 200 401 403 404 500 503 \| 40100 40101 40300 40400 |
| `POST` | `/api/v1/admin/concessions` | jwt+admin/staff audit:admin.create_concession — **NEW 09-22b** | dto.CreateComboRequest (name 2-255, price>=0, image_url omitempty url, `active` omitted = TRUE) | dto.ComboResponse (201) | 201 400 401 403 413 500 503 \| 40001 40100 40101 40300 41300 |
| `PATCH` | `/api/v1/admin/concessions/:id` | jwt+admin/staff audit:admin.update_concession — **NEW 09-22b** | dto.UpdateComboRequest — **PARTIAL**, all pointers; a body with no field is 400/40001 "nothing to update" | dto.ComboResponse | 200 400 401 403 404 500 503 \| 40001 40100 40101 40300 40400 |
| `DELETE` | `/api/v1/admin/concessions/:id` | jwt+admin/staff audit:admin.delete_concession — **NEW 09-22b** | - (path param id) | **204, EMPTY BODY, no envelope** (soft delete) | 204 401 403 404 500 503 \| 40100 40101 40300 40400 |
| `GET` | `/api/v1/admin/discounts` | jwt+**admin only** — **NEW 09-22b** | dto.DiscountListQuery (embeds PageQuery; `active` pointer) | response.Paged{items=[]dto.DiscountCodeResponse} | 200 400 401 403 500 503 \| 40001 40100 40101 40300 |
| `GET` | `/api/v1/admin/discounts/:id` | jwt+**admin only** — **NEW 09-22b** | - (path param id) | dto.DiscountCodeResponse | 200 401 403 404 500 503 \| 40100 40101 40300 40400 |
| `POST` | `/api/v1/admin/discounts` | jwt+**admin only** audit:admin.create_discount — **NEW 09-22b** | dto.CreateDiscountRequest (code 3-32 stored UPPERCASE, kind oneof=percent amount, value>=1, max_discount percent-only, min_order, starts_at/ends_at, max_uses) | dto.DiscountCodeResponse (201) | 201 400 401 403 409 413 500 503 \| 40001 40100 40101 40300 40900 41300 |
| `PATCH` | `/api/v1/admin/discounts/:id` | jwt+**admin only** audit:admin.update_discount — **NEW 09-22b** | dto.UpdateDiscountRequest — PARTIAL; **`code` and `kind` are NOT accepted** (changing either rewrites what past orders meant) | dto.DiscountCodeResponse | 200 400 401 403 404 500 503 \| 40001 40100 40101 40300 40400 |
| `DELETE` | `/api/v1/admin/discounts/:id` | jwt+**admin only** audit:admin.delete_discount — **NEW 09-22b** | - (path param id) | **204, EMPTY BODY** (soft delete; the code string becomes reusable) | 204 401 403 404 500 503 \| 40100 40101 40300 40400 |
| `GET` | `/api/v1/admin/hall-templates` | jwt+admin/staff | - | []dto.HallTemplateResponse | 200 401 403 503 500 \| 40100 40101 40300 50000 50300 |
| `GET` | `/api/v1/admin/halls` | jwt+admin/staff | dto.PageQuery (query: page, page_size<=100, search) | response.Paged{items=[]dto.HallResponse} | 200 400 401 403 503 500 \| 40001 40100 40101 40300 50000 50300 |
| `POST` | `/api/v1/admin/halls` | jwt+admin/staff audit:admin.create_hall | dto.HallRequest (json body) | dto.HallResponse (201 Created) | 201 400 409 401 403 413 503 500 \| 40001 40100 40101 40300 40900 41300 50000 50300 |
| `DELETE` | `/api/v1/admin/halls/:id` | jwt+admin/staff audit:admin.delete_hall | - (path param id) | - (204 No Content, empty body ) | 204 404 409 400 401 403 503 500 \| 40000 40100 40101 40300 40400 40900 50000 50300 |
| `GET` | `/api/v1/admin/halls/:id` | jwt+admin/staff | - (path param id) | dto.HallResponse | 200 404 400 401 403 503 500 \| 40000 40100 40101 40300 40400 50000 50300 |
| `PUT` | `/api/v1/admin/halls/:id` | jwt+admin/staff audit:admin.update_hall | dto.UpdateHallRequest (json body, all fields optional pointers) | dto.HallResponse | 200 400 404 409 401 403 413 503 500 \| 40001 40100 40101 40300 40400 40900 41300 50000 50300 |
| `POST` | `/api/v1/admin/halls/:id/clone` | jwt+admin/staff audit:admin.clone_hall | dto.CloneHallRequest (json body: name required, copy_prices bool) | dto.HallResponse (201 Created) | 201 400 404 409 401 403 413 503 500 \| 40000 40001 40100 40101 40300 40400 40900 41300 50000 50300 |
| `PUT` | `/api/v1/admin/halls/:id/layout` | jwt+admin/staff audit:admin.update_hall_layout | dto.HallRequest (json body; name and prices ignored by the service but prices still required by binding) | dto.HallResponse | 200 400 404 409 401 403 413 503 500 \| 40001 40100 40101 40300 40400 40900 41300 50000 50300 |
| `GET` | `/api/v1/admin/halls/:id/prices` | jwt+admin/staff | - (path param id) | []dto.HallPriceResponse | 200 404 400 401 403 503 500 \| 40000 40100 40101 40300 40400 50000 50300 |
| `PUT` | `/api/v1/admin/halls/:id/prices` | jwt+admin/staff audit:admin.set_hall_prices | dto.PriceRequest (json body: prices map[string]int64, required) | []dto.HallPriceResponse (always 4 rows in models.AllSeatTypes order ) | 200 400 404 401 403 413 503 500 \| 40001 40100 40101 40300 40400 41300 50000 50300 |
| `GET` | `/api/v1/admin/halls/:id/seats` | jwt+admin/staff | - (path param id) | []dto.SeatResponse | 200 404 400 401 403 503 500 \| 40000 40100 40101 40300 40400 50000 50300 |
| `PATCH` | `/api/v1/admin/halls/:id/seats` | jwt+admin/staff audit:admin.bulk_update_seats | dto.BulkSeatUpdateRequest (json body: changes[1..50], each with one selector) | []dto.SeatResponse (only the touched seats, sorted by row then column) | 200 400 404 409 401 403 413 503 500 \| 40001 40100 40101 40300 40400 40900 41300 50000 50300 |
| `PUT` | `/api/v1/admin/halls/:id/seats/:seatId` | jwt+admin/staff audit:admin.update_hall_seat | dto.SeatUpdateRequest (json body: seat_type, is_gap; both optional) | dto.SeatResponse | 200 400 404 409 401 403 413 503 500 \| 40001 40100 40101 40300 40400 40900 41300 50000 50300 |
| `GET` | `/api/v1/admin/orders` | jwt+admin | dto.AdminOrderListQuery (query; embeds PageQuery: page, page_size, search on booking id \| customer email/name/phone \| movie title + status, payment_status, sold_via, showtime_id, movie_id, user_id, date, from, to, sort created_at\|paid_at\|total_amount\|start_at, order) | response.Paged{items=[]dto.AdminOrderListItem} | 200 400 401 403 500 503 \| 40001 40100 40101 40300 50000 50300 |
| `GET` | `/api/v1/admin/overview` | jwt+admin | - (no input) | dto.AdminOverviewResponse {today, last_7_days, upcoming_showtimes, alerts{stuck_refunds,failed_jobs,given_up_emails}} | 200 401 403 500 503 \| 40100 40101 40300 50000 50300 |
| `GET` | `/api/v1/admin/reports/daily` | jwt+admin | - (raw c.Query("from"), c.Query("to"), YYYY-MM-DD; no DTO) | dto.DailyReportResponse {from,to,total_revenue,tickets_sold,days[]DailyAggregateResponse} | 200 400 401 403 500 503 \| 40001 40100 40101 40300 50000 50300 |
| `GET` | `/api/v1/admin/showtimes` | jwt+admin/staff | dto.ShowtimeAdminListQuery (query; embeds PageQuery: page, page_size, search on movie title or hall name + movie_id, hall_id, status, date, from, to, sort start_at\|created_at, order) | response.Paged{items=[]dto.ShowtimeResponse} | 200 400 401 403 500 503 \| 40001 40100 40101 40300 50000 50300 |
| `GET` | `/api/v1/admin/showtimes/:id` | jwt+admin/staff | - (path param id) | dto.ShowtimeResponse | 200 401 403 404 500 503 \| 40100 40101 40300 40400 50000 50300 |
| `POST` | `/api/v1/admin/showtimes` | jwt+admin/staff audit:admin.create_showtime | dto.ShowtimeRequest (json body: movie_id uuid, hall_id uuid, start_at RFC3339, status open\|closed optional) | dto.ShowtimeResponse (201 Created) | 201 400 404 409 401 403 413 503 500 \| 40001 40100 40101 40300 40400 40900 41300 50000 50300 |
| `DELETE` | `/api/v1/admin/showtimes/:id` | jwt+admin/staff audit:admin.delete_showtime | - (path param id) | - (200 with {code:0,message:"deleted"} ) | 200 404 409 400 401 403 503 500 \| 40000 40100 40101 40300 40400 40900 50000 50300 |
| `PUT` | `/api/v1/admin/showtimes/:id` | jwt+admin/staff audit:admin.update_showtime | dto.ShowtimeRequest (json body, full replace; status omitted keeps current) | dto.ShowtimeResponse | 200 400 404 409 401 403 413 503 500 \| 40001 40100 40101 40300 40400 40900 41300 50000 50300 |
| `GET` | `/api/v1/admin/stats` | jwt+admin | - (no input) | dto.AdminStatsResponse {movies, showtimes, bookings, users} | 200 401 403 500 503 \| 40100 40101 40300 50000 50300 |
| `POST` | `/api/v1/admin/uploads/poster` | jwt+admin/staff audit:admin.upload_poster | multipart/form-data field "file" (no DTO; c.Request.FormFile) | dto.UploadResponse {url, content_type, size} | 201 400 401 403 502 503 \| 40000 40001 40100 40101 40300 50200 |
| `GET` | `/api/v1/admin/users` | jwt+admin | dto.UserListQuery (query: page, page_size max 100, search max 255, role oneof customer\|staff\|admin, active bool) + | response.Paged{items []dto.UserResponse, meta {page, page_size, total, total_pages}} | 200 400 401 403 500 503 \| 40001 40100 40101 40300 |
| `POST` | `/api/v1/admin/users` | jwt+admin audit:admin.create_user | dto.CreateUserRequest (json body: email, password 6-72, full_name 2-255, role oneof staff\|admin) | dto.UserResponse | 201 400 401 403 409 413 500 503 \| 40001 40100 40101 40300 40900 41300 50000 |
| `PATCH` | `/api/v1/admin/users/:id` | jwt+admin audit:admin.update_user | dto.UpdateUserRequest (json body: active *bool and/or role *string oneof customer\|staff\|admin) | dto.UserResponse | 200 400 401 403 404 409 413 500 503 \| 40001 40100 40101 40300 40400 40900 41300 |
| `PUT` | `/api/v1/admin/users/:id` | jwt+admin audit:admin.update_user | dto.UpdateUserRequest (json body, same as PATCH) | dto.UserResponse | 200 400 401 403 404 409 413 500 503 \| 40001 40100 40101 40300 40400 40900 41300 |
| `GET` | `/api/v1/admin/reports/breakdown` | jwt+admin — **NEW 09-22** | query `from`, `to` (`YYYY-MM-DD`, raw `c.Query`, no binding tag) | dto.BreakdownResponse (from, to, total_revenue, tickets_sold, **days[]**, **movies[]**, **halls[]**, **providers[]**) | 200 400 401 403 500 503 \| 40001 40100 40101 40300 |
| `POST` | `/api/v1/admin/showtimes/:id/cancel` | jwt+admin/staff audit:admin.cancel_showtime — **NEW 09-22** | - (path param id) | dto.ShowtimeCancelResponse (showtime_id, status, **bookings_affected**) | 200 401 403 404 409 500 503 \| 40100 40101 40300 40400 40900 |
| `POST` | `/api/v1/admin/halls/:id/seats/rows` | jwt+admin/staff audit:admin.add_hall_row — **NEW 09-22** | - (**no body**; path param id only) | `[]dto.SeatResponse` — the seats of the row just appended (**201**) | 201 401 403 404 409 500 503 \| 40100 40101 40300 40400 40900 |
| `DELETE` | `/api/v1/admin/halls/:id/seats/rows/:rowLabel` | jwt+admin/staff audit:admin.delete_hall_row — **NEW 09-22** | - (path params id + rowLabel, e.g. `J`) | `[]dto.SeatResponse` — the remaining grid | 200 401 403 404 409 500 503 \| 40100 40101 40300 40400 40900 |
| `POST` | `/api/v1/admin/halls/:id/seats/merge` | jwt+admin/staff audit:admin.merge_hall_seats — **NEW 09-22** | dto.MergeSeatsRequest (json: `left_label` required max=8, `right_label` required max=8, e.g. `D3`+`D4`) | `[]dto.SeatResponse` | 200 400 401 403 404 409 413 500 503 \| 40001 40100 40101 40300 40400 40900 41300 |
| `POST` | `/api/v1/admin/halls/:id/seats/split` | jwt+admin/staff audit:admin.split_hall_seat — **NEW 09-22** | dto.SplitSeatRequest (json: `label` required max=8) | `[]dto.SeatResponse` | 200 400 401 403 404 409 413 500 503 \| 40001 40100 40101 40300 40400 40900 41300 |

Notes:

- `GET /api/v1/admin/audit-logs` — created_at DESC; page_size default 10 max 100; to is inclusive (created_at < to+1 day); booking_id spans one order's lifecycle.
- `GET /api/v1/admin/batch/jobs` — search = job_name ILIKE %..%; started_at DESC; statuses running|success|failed|skipped|stopped; swagger declares Paged without items type.
- `POST /api/v1/admin/batch/jobs/:name/run` — Fire-and-forget: 202 once the RUNNING row exists; poll /admin/batch/jobs. Names: closeDay, sweepExpiredHolds, sendTicketEmails, cleanup.
- `GET /api/v1/admin/hall-templates` — In-memory only, no DB. Sorted ALPHABETICALLY (large, medium, small) by `slices.Sort` in `Templates()`, NOT small|medium|large. `seat_count` excludes gaps and already accounts for col_span=2 seats, so medium is 10x12 but 115 seats, large 14x14 but 190. `seat_count_by_type` omits any type whose count is 0, so it is a partial map, never 4 keys.
- `GET /api/v1/admin/halls` — search is name ILIKE %search% (NOT escaped: a literal % matches everything), ordered by name; includes inactive halls. page_size > 100 is REJECTED 400/40001, not clamped — the `max=100` binding tag fires before Normalize()
- `POST /api/v1/admin/halls` — prices must hold a positive value for all 4 seat types ; template small|medium|large fills unset layout fields
- `DELETE /api/v1/admin/halls/:id` — Soft delete; 204 has no response envelope, so do not parse JSON on success
- `GET /api/v1/admin/halls/:id` — Returns rows, seats_per_row, screen_position, aisle_after_cols, active — no seats or prices
- `PUT /api/v1/admin/halls/:id` — Partial update of name/screen_position/aisle_after_cols/active only; never touches seats; inactive hall takes no new showtimes
- `POST /api/v1/admin/halls/:id/clone` — Copies the live seat grid including manual edits; clone is always active:true; prices copied only when copy_prices true
- `PUT /api/v1/admin/halls/:id/layout` — Destructive: drops showtime_seats+seats, rebuilds grid, re-seeds seat states for upcoming showtimes; only if hall never had a booking
- `GET /api/v1/admin/halls/:id/prices` — Returns only stored rows, so fewer than 4 seat types is possible; unlike PUT which always echoes all 4
- `PUT /api/v1/admin/halls/:id/prices` — Full upsert of standard/vip/couple/recliner; busts the catalog cache because listings show MIN(price) as from_price
- `GET /api/v1/admin/halls/:id/seats` — Label is row_label+col_number (dto.SeatLabel); carries is_gap and col_span; same handler as GET /api/v1/halls/:id/seats
- `PATCH /api/v1/admin/halls/:id/seats` — All-or-nothing transaction; selector is exactly one of labels/rows/cols/range("A1:C4"); a selector matching no seat fails; col_span untouched
- `PUT /api/v1/admin/halls/:id/seats/:seatId` — Blocked once the hall has any booking; omitted fields keep current value; aliased at PUT /api/v1/halls/:id/seats/:seatId
- `GET /api/v1/admin/orders` — The operator order list, added 2026-09-18. Not scoped to the caller, unlike `GET /orders`. Counter sales (user_id NULL) appear next to online ones; `customer` carries the account for an online order and the walk-in name/phone for a counter sale. `payment` is the attempt the order already CARRIES (bookings.payment_id), so an unpaid hold with an open checkout has none and `payment_status` never matches it. Deleted movies/halls/showtimes stay joined in on purpose. Does NOT reconcile with the provider — use `GET /staff/orders/:id` for that. `date` wins over `from`/`to`, resolved in the server timezone.
- `GET /api/v1/admin/overview` — Today computed live, not from closeDay; each alert list capped at 20 (alertListLimit ); stuck refund = 8 attempts.
- `GET /api/v1/admin/reports/daily` — Reads daily_aggregates written by the closeDay job; defaults to to=today, from=to-6 days. No pagination.
- `GET /api/v1/admin/stats` — The four dashboard tiles, added 2026-09-18. movies/showtimes/users are plain counts of non-soft-deleted rows (locked `active=false` accounts still count, matching `GET /admin/users`); bookings counts `confirmed` ONLY, because the bookings table is also the hold table. Deliberately separate from `/admin/overview`, which answers "how is today going" rather than "how big is the catalogue".
- `GET /api/v1/admin/showtimes` — The operator list, added 2026-09-18. Unlike `GET /showtimes` it keeps closed showtimes, draft/ended movies, past dates and halls without a full price set. `date` wins over `from`/`to`; both bounds are resolved in the server timezone. Never cached.
- `GET /api/v1/admin/showtimes/:id` — Operator read; does NOT require the showtime to be on sale, unlike the booking endpoints.
- `POST /api/v1/admin/showtimes` — end_at derived from movie duration; overlap check includes cleanup minutes; seat states created for the hall grid; status forced open
- `DELETE /api/v1/admin/showtimes/:id` — Soft delete; any booking of any status blocks it — close the showtime instead; busts catalog cache
- `PUT /api/v1/admin/showtimes/:id` — Same movie_id+hall_id+start_at is a status-only change (closing always allowed); moving hall rebuilds seat states and needs zero bookings
- `POST /api/v1/admin/uploads/poster` — Limit storage.max_upload_mb, default 5MB ; over-size returns 400, not 413. Content type sniffed server-side. Feed url into movie poster_url.
- `GET /api/v1/admin/users` — Query normalized to page 1 / size 10, hard cap 100 . Erased accounts are filtered out by soft delete.
- `POST /api/v1/admin/users` — role only staff or admin (customers self-register); created active=true. Audit row written in the same transaction.
- `PATCH /api/v1/admin/users/:id` — Side effect: invalidates the account-status cache, so a lock or role change lands within one request .
- `PUT /api/v1/admin/users/:id` — Alias of the PATCH route, same handler and semantics (partial update despite PUT); pick either from the frontend.

### The 2026-09-22 additions, in more detail

- `GET /api/v1/admin/reports/breakdown` — The analytics feed behind the redesigned dashboard. Unlike
  `/admin/reports/daily`, which returns only the `closeDay` day-rollups, this one pre-aggregates FOUR
  dimensions server-side: `days`, `movies`, `halls` and `providers`. That removes the client-side rollup the
  reports screen had to do from the jsonb `breakdown` column — prefer this endpoint for any new chart.
  `from`/`to` are read with a raw `c.Query` and validated in the service, so a bad date is 400/40001 **with no
  `details` map** — show the message, do not try to attach it to a field.
- `POST /api/v1/admin/showtimes/:id/cancel` — **Not the same as `DELETE /admin/showtimes/:id`.** Delete refuses
  while the showtime has unfinished business; cancel goes through with it: it releases seats and pushes every
  affected booking through the normal refund pipeline, reporting how many in `bookings_affected`. Reach for
  this when a screening genuinely will not happen, and expect money to move.
- The four hall seat-editing routes are the **incremental** grid editor, and they are the answer to a gap this
  config recorded earlier: `col_span` used to be changeable only by regenerating the whole layout from a
  template. `merge` turns two neighbours into one 2-column seat, `split` undoes it, and `rows` appends or drops
  a whole row — none of which destroys the rest of the grid the way `PUT /admin/halls/:id/layout` does.
  `POST .../seats/rows` takes **no body** (it appends the next row after the last) and answers **201**;
  the other three answer 200. All four return a `SeatResponse` array, and all four are still subject to the
  `HallHasBookings` guard, so a hall with a live hold or a confirmed order refuses with 409.
