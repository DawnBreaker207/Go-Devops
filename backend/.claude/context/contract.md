# BackEnd-CP HTTP contract

The wire-level truth: one envelope, 15 status/code rows, 24 enums, and the formats.
Verified against source on 2026-09-18. `FrontEnd-CP/src/types` mirrors this by hand, so anything that changes here
must be reported to the frontend. `docs/swagger.json` is 19 operations stale — this file and the Go code win.

Per-route detail (guard, DTO, statuses) is in `../skills/be-endpoint/endpoints.md`.

This file deliberately keeps `file.go:NN` citations inside its code blocks, unlike the CLAUDE.md files: they were
accurate on the generation date and make the snippets checkable. They WILL drift — grep the symbol, not the line.

## Base URL and prefix

- `/api/v1` is **hardcoded** — `engine.Group("/api/v1")` in `internal/router/router.go`, not configurable.
  Only the host and port vary; the default is `http://localhost:8080` (`SERVER_PORT`).
- Outside the prefix:
  - `GET /health` and `GET /healthz` — probes, no `DBGuard`, no `NoStore`.
  - `GET|HEAD /media/*filepath` — static posters, mounted **only** when the storage driver is not cloudinary.
  - `GET /swagger/*any` — registered **only** when `app.env != "production"`.
  - `ANY /mock-gateway/*path` — the mock provider's simulated checkout, registered only when that provider is enabled.
- Inside the prefix but easy to mistake for outside it:
  - `GET|POST /api/v1/payments/:provider/ipn` and `GET /api/v1/payments/:provider/return` — the webhooks.
  - `GET /api/v1/health` / `healthz` also exist. They sit behind `DBGuard`, so a tripped guard makes them answer
    **503 / 50300 "service busy"** instead of a health body. Use the root `/health` or `/healthz` for probes.

## Authentication

- `Authorization: Bearer <access_token>` — HS256 JWT, **header only, no cookies anywhere** (a repo-wide grep for
  `SetCookie`/`http.Cookie` returns nothing). The scheme match is case-insensitive and the header must be exactly
  two whitespace-separated fields.
- JWT claims are `uid`, `email`, `role`, `typ` (`access` | `refresh`) plus the standard registered claims.
- The login response is `access_token`, `refresh_token`, `token_type: "Bearer"`, `expires_in`.
- The refresh token travels as a **JSON body field**, rotates on use, and revokes its whole family on replay.
- `middleware.Auth` re-checks the account on every request against a 30s-TTL status cache: a locked account
  gets `ErrAccountLocked`, and a role that no longer matches the token gets `ErrInvalidToken`.
- Roles are `admin` | `staff` | `customer`, compared by exact case-sensitive map lookup. **No hierarchy.**
- `OptionalAuth` lets a request with no `Authorization` header through as a guest, but a present-but-invalid
  header still gets 401.

## CORS

- Code default in `internal/config/config.go`: `v.SetDefault("cors.allowed_origins", []string{"http://localhost:3000"})`.
- `config.yaml` widens it to both `http://localhost:3000` and `http://localhost:5173`; `.env.example` lists both too.
- **TODO: confirm** — a real `.env` exists and `loadDotEnv` + `AutomaticEnv` make it beat `config.yaml`, so
  the effective allowlist is whatever `CORS_ALLOWED_ORIGINS` says there. Ask the user; do not read `.env`.
- The Vite dev server runs on **:3000** with no proxy, so the browser calls the API cross-origin.

## The envelope

```go
// pkg/response/response.go:18-23 — the ONLY envelope, used for success and error alike
type Body struct {
	Code    int               `json:"code" example:"0"`
	Message string            `json:"message" example:"success"`
	Data    any               `json:"data,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

const CodeSuccess = 0 // pkg/response/response.go:40

// OK      -> 200 {"code":0,"message":"success","data":<any>}      (response.go:43-45)
// Created -> 201 {"code":0,"message":"created","data":<any>}      (response.go:48-50)
// NoContentOK -> 200 {"code":0,"message":"<custom>"}  (no data)   (response.go:53-55)
```

- `data` has `omitempty`, so it is **absent** on payload-less responses. Do not type it as always present.
- Errors use the same struct: `code` is the 5-digit business code, `data` is always absent, `details` is an
  optional **flat** `map[string]string`.
- Exceptions that do NOT use the envelope: `DELETE /admin/halls/:id` and `DELETE /users/me` answer **204 with
  an empty body**; the payment IPN answers in the provider's own shape (with `code` as a *string*); the SSE
  stream writes `text/event-stream` frames; `/media/*`, `/swagger/*` and `/mock-gateway/*` serve raw bytes.

## Pagination

```go
// pkg/response/response.go:26-37 — returned as the `data` of the standard envelope
type Paged struct {
	Items any  `json:"items"`
	Meta  Meta `json:"meta"`
}

type Meta struct {
	Page       int   `json:"page" example:"1"`
	PageSize   int   `json:"page_size" example:"10"`
	Total      int64 `json:"total" example:"42"`
	TotalPages int   `json:"total_pages" example:"5"`
}

// internal/dto/common.go:3-25 — the request side
const (
	DefaultPage     = 1
	DefaultPageSize = 10
	MaxPageSize     = 100
)

type PageQuery struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Search   string `form:"search" binding:"omitempty,max=255"`
}

// Full wire shape:
// {"code":0,"message":"success","data":{"items":[...],"meta":{"page":1,"page_size":10,"total":42,"total_pages":5}}}
```

Not every list is paged. Bare-array list endpoints: `GET /showtimes`, `GET /movies/:id/showtimes`,
`GET /halls/:id/seats`, `GET /admin/halls/:id/seats`, `GET /admin/halls/:id/prices`,
`GET /admin/hall-templates`, `GET /payments/providers`, `GET /staff/showtimes/:id/tickets`.

## Validation failures

```go
// POST /api/v1/auth/register  body: {"email":"nope","password":"123"}
// -> HTTP 400 (never 422)
{
  "code": 40001,
  "message": "validation failed",
  "details": {
    "email": "must be a valid email address",
    "password": "must be at least 6",
    "full_name": "field is required"
  }
}
// Keys are the json/form tag names (internal/router/validator.go:18-30).
// Messages per validator tag (pkg/response/response.go:128-144):
//   required -> "field is required"
//   email    -> "must be a valid email address"
//   min      -> "must be at least <param>"
//   max      -> "must be at most <param>"
//   oneof    -> "must be one of: <param>"
//   datetime -> "must match format <param>"
//   default  -> "failed on rule <tag>"
//
// 429 body (rate limit / login lockout), internal/middleware/guard.go:42-43:
// {"code":42900,"message":"too many requests, try again later","details":{"retry_after_seconds":"1"}}
//
// 428 body (terms gate), internal/service/auth_service.go:116-121:
// {"code":42800,"message":"you must accept the current terms before continuing",
//  "details":{"required_terms_version":"1","accepted_terms_version":"0"}}
```

## Status and business codes

| Business code | HTTP | Meaning |
|---|---|---|
| `0 (response.CodeSuccess)` | 200 / 201 / 202 | Success. 202 on `POST /admin/batch/jobs/{name}/run`. The two bare-204 DELETEs send no body at all, so they carry no code |
| `40000 (CodeBadRequest)` | 400 | Generic bad request, and the Postgres 22P02 (bad uuid/value) / 23503 (fk) mappings. NOT returned for a wrong HTTP verb — see the note below the table |
| `40001 (CodeValidation)` | 400 | Validation failure — field-level `details`. Also upload-invalid and upload-too-large. NOTE: 400, never 422 |
| `40100 (CodeUnauthorized)` | 401 | Missing/malformed Authorization header, bad credentials, invalid or replayed refresh token, role changed since the token was minted, invalid payment signature, invalid SSE token |
| `40101 (CodeTokenExpired)` | 401 | 'token has expired' — the distinct signal telling the FE to call /auth/refresh |
| `40300 (CodeForbidden)` | 403 | Wrong role for the route, account locked, showtime closed, accessing another user's order |
| `40400 (CodeNotFound)` | 404 | Resource not found, unknown route, unknown payment provider, showtime not open |
| `40900 (CodeConflict)` | 409 | State conflict: seats taken, hold expired, booking not payable, idempotency key reused, duplicate email, hall/showtime edit locks |
| `41300 (CodePayloadTooLarge)` | 413 | 'request body is too large' — JSON body over server.max_body_bytes |
| `42800 (CodeTermsRequired)` | 428 | 'you must accept the current terms before continuing' — login blocked until POST /auth/terms-accept |
| `42900 (CodeTooManyRequests)` | 429 | Token-bucket rate limit, login lockout, or too many open SSE streams. Carries details.retry_after_seconds + a Retry-After header |
| `50000 (CodeInternal)` | 500 | 'internal server error' — the catch-all for any non-AppError, INCLUDING a malformed JSON body or a non-numeric ?page |
| `50200 (CodeBadGateway)` | 502 | Upstream failed: payment provider unavailable, image storage unavailable |
| `50300 (CodeServiceUnavailable)` | 503 | 'service busy, try again later' — the DBGuard tripped, so EVERY /api/v1 route returns this. Retryable |
| `503 (raw HTTP status as the body code)` | 503 | INCONSISTENCY: the health endpoints put the literal 503 in `code`, not 50300, and /health also returns `data` |

There is **no 422** anywhere in the backend. A malformed JSON body or a non-numeric `?page` is neither a
`MaxBytesError` nor a `validator.ValidationErrors`, so it falls through to **500 / 50000** — not 400.
A wrong HTTP verb answers **404 / 40400**, because `engine.NoMethod` is dead code (gin's
`HandleMethodNotAllowed` defaults to false and nothing sets it).

## Formats

**Time.** All time.Time fields marshal with Go's default encoding/json, i.e. RFC3339 with nanoseconds (e.g. "2026-09-15T19:00:00+07:00"). Columns are TIMESTAMPTZ and the GORM DSN sets TimeZone=Asia/Ho_Chi_Minh (config.yaml:31), so offsets come back as +07:00, not Z. Exceptions: movie `release_date` is a plain "YYYY-MM-DD" STRING in both directions, and the `date` query param on /showtimes and /movies/:id/showtimes is "YYYY-MM-DD".

**Money.** int64 whole Vietnamese dong — NOT minor units, NOT decimal, and there is no currency field anywhere in any DTO. "CreateRequest amounts are integer VND". Fields: total_amount, price, amount, paid_amount, from_price, and the `prices` map values. The FE does all formatting; the backend only formats for emails (formatVND renders 120000 as "120.000 ₫",).

**IDs.** UUID v4 string. Every model has `gorm:"type:uuid;primaryKey"` with uuid.NewString in BeforeCreate; JSON type is string. Some request fields enforce it with `binding:"uuid"`. A malformed UUID in a path reaches Postgres and comes back as 400/40000 "invalid identifier or value" via the 22P02 mapping. Exception: ticket `code` is a varchar(32) non-UUID string.

## Enums

A new value must be added to the Go const, every `oneof=` tag that accepts it, and a **new** SQL migration —
then reported to the frontend, which mirrors these as TypeScript string unions.

| Enum | Values |
|---|---|
| user.role | `admin` \| `staff` \| `customer` |
| movie.status | `draft` \| `showing` \| `ended` |
| movie.age_rating | `P` \| `K` \| `T13` \| `T16` \| `T18` |
| showtime.status | `open` \| `closed` |
| showtime_seat.status (seat map + SSE `seats` events) | `available` \| `held` \| `sold` |
| seat.seat_type | `standard` \| `vip` \| `couple` \| `recliner` |
| hall.screen_position | `front` \| `back` |
| hall.template (request only) | `small` \| `medium` \| `large` |
| booking.status | `pending` \| `confirmed` \| `expired` \| `refunded` |
| booking.status_reason | `replaced` \| `hold_expired` \| `seats_lost` \| `showtime_closed` \| `amount_mismatch` \| `paid_after_expiry` \| `canceled` |
| booking.sold_via | `online` \| `counter` |
| ticket.status | `issued` \| `redeemed` |
| RedeemResponse.status (gate verdict) | `ok` \| `used` \| `wrong_show` \| `not_found` \| `too_early` \| `closed` |
| payment.status | `pending` \| `paid` \| `failed` \| `refund_pending` \| `refunded` |
| payment.status_reason | `create_failed` \| `declined` \| `abandoned` \| `duplicate_payment` |
| payment provider name | `mock` |
| batch_job.status | `running` \| `success` \| `failed` \| `skipped` \| `stopped` |
| batch_job trigger | `cron` \| `manual` \| `confirm` |
| audit_log.outcome | `success` \| `failure` |
| movie list ?sort | `release_date` \| `title` \| `created_at` |
| list ?order | `asc` \| `desc` |
| jwt claim `typ` | `access` \| `refresh` |
| SSE event names | `connected` \| `seats` |
| TokenResponse.token_type | `Bearer` |

## Realtime (SSE)

Two steps: mint a short-lived stream token, then open the stream with it in the query string. Event names are
`connected` and `seats`; the `seats` payload is `{showtime_id, hall_id, seats:[{id, status}]}`. A 429 here is a
bare `too many realtime streams open` with **no** `details.retry_after_seconds` — unlike the other 429s.
`middleware.Logger` redacts the `token`, `sig` and `signature` query keys because of this.

<!-- last verified: 2026-09-18 against 6cb71bf (develop) -->
