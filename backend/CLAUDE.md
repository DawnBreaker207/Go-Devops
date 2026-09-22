# BackEnd-CP — Claude context

Auto-loaded when a session opens here, and lazily when a session at `CinemaProject/` reads a file in this repo.
Keep <200 lines. Deep dives: `.claude/context/contract.md` (envelope, error codes, enums) and
`.claude/skills/be-endpoint/` (every route + copyable templates).
This file cites symbol + file, not line numbers — line numbers drift, grep the symbol.

## Project overview

Cinema booking API: catalog (movies/halls/showtimes), seat holds and booking, payments through a provider
registry, tickets with QR + check-in, staff box office, admin dashboards, audit log and cron batch jobs.

Added 2026-09-22 (10 migrations now, up from 5): **multi-device sessions** (`000006`, `/users/me/sessions`),
**a `coming_soon` catalog lifecycle with preview showtimes** (`000007` — `movie.status` is no longer just
draft/showing/ended), **a concession/combo catalogue and combo orders** (`000008`, `concession_items` +
`combo_orders`), **notification preferences** (`000009`), plus `POST /orders/init` and
`POST /orders/:id/refresh` (open-then-heartbeat a hold before any seat is picked), `GET /tickets/:id/qr`
(server-rendered PNG), `POST /admin/showtimes/:id/cancel` (refunds, unlike DELETE which refuses), incremental
hall seat editing (row add/delete, seat merge/split) and `GET /admin/reports/breakdown`.
Added later the same day: **an operator write path for the concession catalogue** (`/admin/concessions`, admin
AND staff) and **real discount codes** (`000010`, `/admin/discounts` admin-only plus
`POST|DELETE /orders/:id/discount` for customers). The discount is the one place the money path changed:
`bookings.total_amount` stays the undiscounted seat subtotal because `finalizeTx` asserts the sold seats add up
to it, so the charge is `Booking.Payable()` (`total_amount - discount_amount`) and that is what lands in
`payments.amount`.
Plus `GET /pricing` (public): the only anonymous read of `hall_prices`, added because the customer price page
had been hardcoding four tiers that matched no row in the table.
Module `github.com/Cinema-Project-Juann/BackEnd-CP`. Working branch `develop`. **162** Go files outside `docs/`.
`internal/router/router.go` registers **108 `/api/v1` operations** as of 2026-09-22 (`v1.Match` on the payment IPN
counts twice; `engine.Static` counts once as `GET|HEAD /media/*filepath`).
`docs/swagger.json` covers 102 of them. The reproducible count lives in
`.claude/skills/be-endpoint/endpoints.md` — run it rather than trusting this sentence.

## Tech stack

Go 1.26.2, Gin 1.12, GORM 1.31 + pgx (Postgres), viper (config), zap (logging), golang-jwt v5 (HS256),
swaggo (generated docs), go-redis v9 (**optional** cache), amqp091 (**optional** queue), robfig/cron v3 (batch),
go-qrcode + gozxing (tickets), bcrypt. No mocking library — tests use a real throwaway Postgres.

## Architecture map (concrete paths only)

- `cmd/server/main.go` — the whole wiring, in one `run()`: config -> logger -> db -> optional redis -> optional
  rabbitmq -> repositories -> services -> handlers -> `router.New` -> `http.Server` -> graceful shutdown.
  **Manual constructor injection only** — no DI container, no `init()` registration, no globals except `pkg/logger`.
- `internal/router/router.go` — every route, in groups. `internal/router/validator.go` registers **no custom
  validation tags**, only a json/form tag-name mapper for error messages.
- `internal/middleware/` — global chain in order: `RequestID -> Recovery -> SecurityHeaders -> BodyLimit ->
  Logger -> CORS`; then `/api/v1` adds `DBGuard(db,500ms) -> NoStore`; then per-group `RateLimit`/`Auth`/
  `OptionalAuth`; then per-route `RateLimit -> Audit -> RequireRoles`.
- `internal/handlers/` — one file per resource, `XHandler` struct + `NewXHandler(...) *XHandler`, receiver `h`.
- `internal/service/` — `XService` interface + unexported `xService` impl, `NewXService(...) XService`, receiver `s`.
  Owns transactions, business rules, audit rows and model->DTO mapping.
- `internal/repository/` — `XRepository` interface + impl, receiver `r`. GORM only. (`HallRepository` and
  `ShowtimeRepository` are pre-existing exported structs, not interfaces — follow the interface pattern for new ones.)
- `internal/dto/` — `XRequest` / `XResponse` / `XQuery` structs plus `NewXResponse(model)` mappers.
- `internal/models/` — GORM models, UUID primary keys via `BeforeCreate`.
- `pkg/response/` — the single response envelope. `pkg/errors/` — 52 error sentinels + the 14 business codes.
- `pkg/jwt/`, `pkg/logger/`, `pkg/cache/`, `pkg/queue/`, `pkg/ratelimit/` — infrastructure wrappers.
- `internal/database/` — `Connect`/`Close` and `SeedAdmin` (the only admin bootstrap). `internal/notify/` — the
  mailer behind ticket emails, the password-reset link and the given-up-email alert.
- `internal/batch/` + `internal/jobs/` — cron manager and its 4 jobs. A new job is a `*batch.Job` constructor in
  `internal/jobs/<name>.go` plus one `batchManager.Register(...)` line in `cmd/server/main.go` — there is no
  registry file. `Job.Schedule` is a **6-field** cron spec (seconds first), optionally `CRON_TZ=<zone>` prefixed;
  an empty schedule means manual-trigger only. `internal/sse/` — realtime seat hub.
- `internal/payment/` — provider registry + `mock` provider with a simulated gateway. `internal/storage/` —
  local disk or Cloudinary. `internal/audit/` — audit record stashing.
- `migrations/schema/` — the **only** source of schema truth. `migrations/seed/` — idempotent psql seeds.

## The first admin account

There is no admin self-registration: `POST /auth/register` always forces `role=customer`, and
`POST /admin/users` needs an admin token. The only bootstrap is `database.SeedAdmin`, which runs **only when
`app.env == "development"`** and **only while the `users` table is empty**, using `app.admin_email` /
`app.admin_password` (`config.yaml` defaults: `admin@cinema.local` / `admin123`). So: seed on a fresh dev DB, or
promote a user with SQL. `make migrate-db-reset` wipes the DB and lets the seed run again.

## Critical conventions

**Layering.** A handler does exactly four things: declare `var req dto.XRequest`, bind it, call **one** service
method with `c.Request.Context()`, and answer through a `pkg/response` helper. Two pre-existing files opt out and
already import gorm — `health_handler.go` (pings the DB, hand-builds its 503) and `batch_handler.go` (writes its
own audit row, logs, answers a hand-built 202). Do not add a third.
Services return DTOs, with two grandfathered exceptions: `HallService.SeatsByHall` and `PricesByHall` return
`[]models.Seat` / `[]models.HallPrice`, and `HallHandler.Prices` maps them in the handler. Do not add more.
Repositories either translate `gorm.ErrRecordNotFound` into an `apperrors` sentinel at the query site
(`movie_repository.go`, `user_repository.go`) or return `(nil, nil)` and let the service raise the sentinel
(`hall`, `showtime`, `password_reset`, `booking` — see the `firstOrNil` helper). Match the file you are editing.
Wrap other errors as `fmt.Errorf("verb phrase: %w", err)`.

**Catalog cache.** Any write touching movies, showtimes or hall prices must call `bumpCatalog` **after the
transaction commits** — it bumps the generation embedded in every public list/detail cache key. Nine existing
write paths do it. Omitting it serves stale data to anonymous readers for the whole Redis TTL; calling it inside
the transaction is also wrong.

**Refunds** are never a direct provider call: the service flips the payment with `MarkRefundPending` inside the
transaction, then a post-commit `afterFinalize` hook calls `settleRefund`, which takes a lease via `ClaimRefund`
so the IPN path cannot double-refund. Read `internal/service/booking_payment.go` before touching any of it.

**Transactions.** The service opens `s.db.Transaction(func(tx *gorm.DB) error {...})` and passes `tx` into
repository write methods (which therefore take `db *gorm.DB` as a parameter). GORM is opened with
`SkipDefaultTransaction`, so a single write is **not** implicitly transactional — a change plus its audit row
needs an explicit transaction.

**Errors.** One funnel: `response.Error(c, err)`. It maps `*http.MaxBytesError` -> 413,
`validator.ValidationErrors` -> 400/40001 with a `details` map, else `apperrors.From(err)`; it logs everything
>= 500 exactly once. Declare a new business error as a sentinel in the `var` block of `pkg/errors/errors.go`
built from a status constructor (`NotFound("movie not found")`). Match a sentinel with `errors.Is` — that is what
production code does (`auth_service.go`, `batch/manager.go`) and `*AppError` has `Unwrap`. Two traps: `Code` is
**not** an identity (every `NotFound` is 40400), and `.Wrap`/`.WithDetails` return a **clone**, so `errors.Is`
fails on a wrapped sentinel — return sentinels unwrapped. Tests compare `Code`+`Message` via `isAppErr`.

**Validation.** Only on the DTO, with stock validator tags (`required omitempty min max email url uuid oneof
datetime dive`). Bind with `ShouldBindJSON` / `ShouldBindQuery` (never `BindJSON`/`MustBindWith` — they write
their own 400). Every list query embeds `dto.PageQuery` and calls `query.Normalize()` on the next line.
Trimming and normalisation happen in the **service**, never the handler.

**Swagger.** Every endpoint carries `@Summary @Tags @Success @Router`; authenticated ones add
`@Security BearerAuth`. Express the envelope with generics: `response.Body{data=dto.XResponse}` or
`response.Body{data=response.Paged{items=[]dto.XResponse}}`. Regenerate with `make swag`; `docs/` is generated.

**Logging and audit.** `logger.L().Error(...)` with `logger.String/Int/Err` — never import zap outside
`pkg/logger` and `internal/middleware/logger.go`. Handlers do not log, with one exception: `BatchHandler.Run`
warns when its audit row fails to write. Write the success audit row inside the
business transaction via `audit.FromContext(ctx)` + `audit.In(ctx, tx, rec)`; `middleware.Audit` writes the
failure row for any status >= 400, so register it **before** a route-level `RequireRoles` to keep 403s audited.
Known exception: on the `/staff` and `/admin` groups the role check is a group-level `.Use(...)`, which runs first,
so a 403 there is **not** audited. Audit action names are `<namespace>.<verb>` and the success row must use the
same string as the route.

**Context.** New methods take `ctx context.Context` first (the older `HallRepository`/`ShowtimeRepository` take
`tx *gorm.DB` instead — match the file). The handler is the only place a context is created, always
`c.Request.Context()` — never `c` itself, never `context.Background()`.

**Gin context keys.** Only `request_id`, `user_id`, `user_email`, `user_role`, `audit_error_message` exist. Read
them through `middleware.GetRequestID` / `CurrentUserID` / `CurrentUserRole`, never raw `c.Get`.

## Never

- Never import `gorm.io/gorm` into a new `internal/handlers/` file, and never build a query there.
- Never hand-build an error body (`c.JSON(400, gin.H{...})`) or call `c.AbortWithStatusJSON` from a handler —
  that bypasses the business code, the `details` map, the `Retry-After` header and the audit error message.
  `response.Abort` is for middleware; `response.Error` is for handlers.
- Never return a `models.*` type from a new service method — add a `dto.NewXResponse` mapper first.
- Never log or audit an email, password, hash, refresh token, or a query string containing `token`/`sig`/`signature`.
- Never hand-edit `docs/docs.go`, `docs/swagger.json` or `docs/swagger.yaml`.
- Never edit an already-applied file in `migrations/schema/` — add a new pair via `make migrate-create`.
  There is **no `AutoMigrate`** anywhere; the SQL is the only schema truth.
- Never add a mocking library or an in-memory DB to the tests, and never run them without `-count=1`.
- Never read `.env`. Config comes from `internal/config`; tests read `TEST_DB_*` / `TEST_REDIS_ADDR` / `QUEUE_TEST_URL`.
- Never assume a role hierarchy: `internal/middleware/auth.go` does an exact, case-sensitive map lookup, so
  `admin` does **not** imply `staff`.

## Build & dev commands (verified 2026-09-18)

| Purpose | Command | Needs |
|---|---|---|
| Static analysis | `make lint` (= `go vet ./...`) | nothing — **passes today** |
| Format | `make fmt` | nothing |
| Tests without infra | `go test ./internal/handlers/... ./internal/middleware/... ./pkg/... -count=1` | nothing — **passes today** |
| Full suite | `make test` | Postgres + RabbitMQ; `internal/service` self-skips when Postgres is down |
| Full suite + infra | `make test-system` | docker (starts postgres + rabbitmq itself) |
| Run the server | `make run` | Postgres on :5432; Redis/RabbitMQ optional |
| Build binary | `make build` -> `bin/backend-cp` | nothing |
| Regenerate swagger | `make swag` | nothing — `swag v1.16.4` is already at `$(go env GOPATH)/bin/swag`, which the Makefile calls by absolute path |
| Apply migrations | `make migrate-up` | Postgres + the `migrate` CLI, which is **not installed on this machine** |
| New migration | `make migrate-create name=<module>_<change>` | the `migrate` CLI |
| Seed / reset | `make migrate-seed` / `make migrate-db-reset` | seed: the `cp-postgres` compose container. reset: that **plus** the `migrate` CLI, and it **destroys all data** |
| Whole stack | `make docker-up` / `make docker-down` | docker + `JWT_*` secrets set |

Config precedence: real env vars > `.env` (gap-fill) > `config.yaml` > code defaults; keys map `.` -> `_`, no
prefix. The only hard boot checks are that the two JWT secrets are non-empty and **differ** — and `config.yaml`
already ships dev placeholders for both, so a local run boots with no env vars set. In `production` the extra
checks reject short, placeholder or `dev-` prefixed secrets.
Postgres is required; **Redis and RabbitMQ degrade gracefully** — keep new code tolerant of both being absent.

## When you finish a task

1. `make lint`, then `go test` on the packages you touched (say plainly if the full suite needs infra you lack).
2. `make swag` if you touched any annotation. Never hand-edit the output.
3. If you changed a DTO or a route, say so — `FrontEnd-CP/src/types` is hand-written and will drift.
   A new **handler type** must be added to the `router.Handlers` struct, the `cmd/server/main.go` literal, and
   `internal/service/http_test.go` `buildEngine` — the last one uses a keyed literal, so tests keep compiling but
   the new handler is nil there until you wire it.
4. Report what you did NOT do and every `TODO: confirm` you left.
5. Leave `git status` clean except the files you meant to change.

<!-- last verified: 2026-09-18 against 6cb71bf (develop) -->
