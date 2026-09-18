# BackEnd-CP endpoint reference

Generated 2026-09-18. The **Guard column is parsed directly from `internal/router/router.go`** (group nesting +
per-route middleware), not from prose. Request/response/notes come from the handlers and DTOs.
`docs/swagger.json` was regenerated on 2026-09-18 and now covers 72 of the 78 `/api/v1` operations; the 6
gaps are aliases with no `@Router`. Still treat `router.go` as the only complete list.

**83 distinct METHOD+path rows.** `router.go` holds 82 registration statements; `v1.Match` (GET+POST on the
payment IPN) yields two rows, and `engine.Static` counts once as `GET|HEAD /media/*filepath`.

## Re-verify these numbers before trusting them

Every count in the config is reproducible. Run these from the `BackEnd-CP` root; the expected values held on
2026-09-18. If one disagrees, this file is stale — fix it before building on it.

```bash
grep -cE '\.(GET|POST|PUT|PATCH|DELETE)\(' internal/router/router.go   # 79  \_ 82 statements
grep -cE '\.(Match|Any|Static)\(' internal/router/router.go            #  3  /  = 83 rows (Match counts twice)
find . -name '*.go' -not -path './docs/*' | wc -l                      # 144 Go files
grep -cE '^\s*Err[A-Za-z0-9]+\s*=\s*(BadRequest|Validation|Unauthorized|TokenExpired|Forbidden|NotFound|Conflict|PayloadTooLarge|PreconditionRequired|TooManyRequests|Internal|BadGateway|ServiceUnavailable)\(' pkg/errors/errors.go   # 52
python3 -c "import json;d=json.load(open('docs/swagger.json'));print(sum(len([k for k in v if k in ('get','post','put','patch','delete')]) for v in d['paths'].values()))"   # 72 documented vs 78 real -> 6 aliases
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

## /api/v1/users  (4)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `DELETE` | `/api/v1/users/me` | jwt+customer audit:users.delete_me | dto.DeleteAccountRequest (json body: password) | - (204 No Content, empty body via c.Status) | 204 400 401 403 404 409 500 503 \| 40001 40100 40300 40400 40900 |
| `GET` | `/api/v1/users/me` | jwt | - | dto.UserResponse | 200 401 403 404 500 503 \| 40100 40101 40300 40400 50300 |
| `PUT` | `/api/v1/users/me` | jwt audit:users.update_profile | dto.UpdateProfileRequest (json body: full_name 2-255 required, phone optional max 20) | dto.UserResponse | 200 400 401 403 404 413 500 503 \| 40001 40100 40101 40300 40400 41300 |
| `PUT` | `/api/v1/users/me/password` | jwt audit:users.change_password | dto.ChangePasswordRequest (json body: current_password, new_password 6-72) | dto.TokenResponse | 200 400 401 403 404 413 500 503 \| 40001 40100 40101 40300 40400 41300 50000 |

Notes:

- `DELETE /api/v1/users/me` — Irreversible: scrubs PII, sets deleted_at, expires pending holds, deletes refresh + reset tokens. Staff/admin get 403.
- `GET /api/v1/users/me` — Auth re-reads account status (30s cache); a role change since the token was minted answers 401.
- `PUT /api/v1/users/me` — Only full_name and phone are writable; empty phone clears it. Audit row written inside the transaction .
- `PUT /api/v1/users/me/password` — Revokes every other session and returns a fresh token pair; the client must replace both stored tokens.

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

## /api/v1/orders  (8)

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

Notes:

- `GET /api/v1/orders` — Own orders only, by CurrentUserID. Payment summary is always nil here; showtime filled per row .
- `GET /api/v1/orders/:id` — Reconciles then returns tickets[]; each ticket.code is the string the frontend renders as the QR.
- `POST /api/v1/orders/:id/cancel` — Releases seats and broadcasts SSE 'available'. Leaves an open provider checkout alone; late money is refunded.
- `POST /api/v1/orders/:id/confirm` — Idempotent; reconciles with provider first, then issues tickets. A refunded settle answers 409, not 200.
- `POST /api/v1/orders/:id/pay` — Empty provider uses config default ("mock"). Re-pay same provider returns the open checkout; frontend must redirect to redirect_url.
- `GET /api/v1/orders/:id/status` — Reconciles with the provider on every call (may settle the booking); poll this while waiting for payment.
- `GET /api/v1/orders/:id/tickets` — Alias: identical handler and full payload as GET /api/v1/orders/:id, not a tickets-only array.
- `POST /api/v1/orders/hold` — 201 not 200. Idempotency-key body field. One pending hold per user+show; replacing keeps old expires_at. TTL 10min default.

## /api/v1/tickets  (1)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `POST` | `/api/v1/tickets/:id/redeem` | jwt+staff/admin audit:staff.redeem_ticket | dto.RedeemRequestBody (json body: showtime_id required) + path id = ticket id OR QR code | dto.RedeemResponse (status, ticket_id, showtime_id, movie_title, age_rating, hall_name, seat_label, start_at, checkin_opens_at, checkin_closes_at) | 200 400 401 403 413 500 503 \| 40001 40100 40101 40300 41300 50000 50300 |

Notes:

- `POST /api/v1/tickets/:id/redeem` — Check-in gate: refusals are 200 with a verdict, not HTTP errors. Window default -30min/+20min around start_at.

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

## /api/v1/admin  (28)

| Method | Path | Guard | Request | Response `data` | Statuses \| codes |
|---|---|---|---|---|---|
| `GET` | `/api/v1/admin/audit-logs` | jwt+admin | dto.AuditLogListQuery (query; embeds dto.PageQuery) - | response.Paged{items: []dto.AuditLogResponse, meta} | 200 400 401 403 500 503 \| 40001 40100 40101 40300 50000 50300 |
| `GET` | `/api/v1/admin/batch/jobs` | jwt+admin | dto.PageQuery (query: page, page_size, search) - | response.Paged{items: []models.BatchJob, meta} | 200 400 401 403 500 503 \| 40001 40100 40101 40300 50000 50300 |
| `POST` | `/api/v1/admin/batch/jobs/:name/run` | jwt+admin audit:admin.run_job | - (job name is the :name path param; no body read) | gin.H{job, run_id, status:"running", triggered_by:"manual"} | 202 404 409 401 403 500 503 \| 40100 40101 40300 40400 40900 50000 50300 |
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

