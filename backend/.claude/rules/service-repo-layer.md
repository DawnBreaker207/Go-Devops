---
paths:
  - "**/internal/service/**/*.go"
  - "**/internal/repository/**/*.go"
  - "**/internal/models/**/*.go"
---

# Service + repository layer rules

## Shapes

- Service: exported `XService` interface + unexported `xService` struct; `NewXService(...) XService` returns the
  **interface**. Receiver `s`. Use an options struct only when there are many optional collaborators
  (`service.BookingOptions` is the single precedent).
- Repository: exported `XRepository` interface + unexported impl; `NewXRepository(db *gorm.DB) XRepository`.
  Receiver `r`. (`HallRepository`/`ShowtimeRepository` are older exported structs — do not copy that.)
- New methods take `ctx context.Context` first and repositories chain `.WithContext(ctx)`. The two older exported
  structs are different: `HallRepository` (14 methods) and `ShowtimeRepository` (8+) take `tx *gorm.DB` first with
  no ctx and never chain `WithContext`. Match the surrounding file when editing those two.
- Service methods return DTOs, never models. Map with `dto.NewXResponse`.

## Transactions and audit

```go
return s.db.Transaction(func(tx *gorm.DB) error {
	if err := s.repo.Update(ctx, tx, &current); err != nil {
		return err
	}
	if rec, ok := audit.FromContext(ctx); ok {
		rec.ResourceID = current.ID
		// Before/After are map[string]any — never assign a model: it will not
		// compile, and it would leak json:"-" fields such as User.Password.
		rec.After = map[string]any{"title": current.Title, "status": current.Status}
		return audit.In(ctx, tx, rec)
	}
	return nil
})
```

- Repository **write** methods take `db *gorm.DB` as a parameter so the service can hand in its transaction.
  Repositories never open a transaction.
- GORM runs with `SkipDefaultTransaction`, so a single write is not implicitly atomic. A change plus its audit
  row needs the explicit transaction above.
- `audit.Record.Before`/`After` are `map[string]any` (never a model — it will not compile and would leak
  `json:"-"` fields). `BookingID` is mandatory on any row belonging to a booking's lifecycle. When there is no
  `middleware.Audit` in play (IPN, cron, sweep), `audit.FromContext` misses and the code falls back to
  `audit.Record{ActorRole: "system"}` — see the helper in `internal/service/booking_service.go`.
- Write the audit row for a successful change **inside** that transaction, using the same action string the route
  passed to `middleware.Audit`. The exception is the auth flows, where a failed audit must not roll the change
  back — those write it afterwards and only log a warning.

## Catalog cache invalidation

A write that touches movies, showtimes or hall prices must call `bumpCatalog` (`internal/service/catalog_cache.go`)
**after the transaction commits** — never inside it and never before. It increments `catalog:gen`, the generation
baked into every public list/detail cache key. Nine write paths already do this
(`movie_service.go`, `showtime_service.go`, `hall_service.go`); a new one that forgets serves stale data to
anonymous readers until the Redis TTL expires.

## Errors

- Translate not-found at the query site: `if errors.Is(err, gorm.ErrRecordNotFound) { return nil, apperrors.ErrXNotFound }`.
- Wrap everything else: `fmt.Errorf("create movie: %w", err)`.
- New business errors are package-level sentinels in the `var` block of `pkg/errors/errors.go`, built from a
  status constructor: `BadRequest Validation Unauthorized TokenExpired Forbidden NotFound Conflict
  PayloadTooLarge PreconditionRequired TooManyRequests Internal BadGateway ServiceUnavailable`.
  Never construct `&AppError{...}` inline.
- Attach context with `.Wrap(err)` (log only) and `.WithDetails(map[string]string{...})` (visible to the client).
  Both return a **clone**, so never mutate a shared sentinel.
- Compare with `errors.Is(err, apperrors.ErrX)` — that is what production code does, and `*AppError` implements
  `Unwrap`. It matches only a sentinel returned **unchanged**: `.Wrap`/`.WithDetails` return a clone and break it.
  `Code` is not an identity (every `NotFound` is 40400); tests use `isAppErr`, which compares `Code`+`Message`.
- Detect Postgres races with the predicates, not string matching: `IsUniqueViolation`, `IsExclusionViolation`,
  `IsDeadlock`, `IsRetryable`.
- Do not log an expected business error — `response.Error` already logs everything >= 500 exactly once.

## Normalisation, logging, privacy

- `strings.TrimSpace` on stored text and `normalizeEmail` (lower+trim) belong here, not in the handler.
  Repositories additionally trim the search term and lower-case both sides of a `LIKE`.
- Log via `logger.L()` + `logger.String/Int/Err`. Never import zap here.
- Never log an email, password, hash or token. Prefer `logger.String("user_id", ...)`.
- `models.User.Password` is `json:"-"`; return `dto.UserResponse`, never the model.

## Models and enums

- UUID primary key with `gorm:"type:uuid;primaryKey"` and `uuid.NewString()` in `BeforeCreate`.
- A new enum value must be added in three places: the Go const, the `oneof=` binding tag of any DTO that accepts
  it, and the SQL constraint in a **new** migration. Then tell the frontend — it mirrors these as string unions.
- Soft delete uses `deleted_at`; `active` is a separate lock flag. They are not interchangeable.

## Tests

- `package service_test` (black box) in `internal/service/*_test.go`, against a real throwaway Postgres.
- Start with `e := newEnv(t)` (or `h := newHTTPEnv(t)` for a full router over httptest). `newEnv` skips when
  Postgres is unreachable, wipes every table and seeds 8 users, 1 movie, "Hall 1" (2x5 seats, gap A5) and a
  showtime 3 hours out.
- Use the helpers: `e.must`, `e.count`, `e.wantStatus`, `e.wantSeat`, `e.checkInvariants`, `httpStatus(err)`,
  `isAppErr(err, apperrors.ErrX)`.
- Name router tests `TestHTTP_<Behaviour>` and service tests `Test<Subject>_<Behaviour>`, with a one-line comment
  naming the spec requirement above each.
- No mocking framework. Fakes are small local structs in the test package.
- Run with `-count=1`. A new handler type must also be added to `internal/service/http_test.go` `buildEngine`.
