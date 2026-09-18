---
name: be-endpoint
description: be-endpoint - add or change a BackEnd-CP HTTP endpoint through the enforced dto/repository/service/handler/router order, with swagger and tests. Use for any backend API work in this repo.
argument-hint: <what the endpoint should do, e.g. "GET /admin/showtimes phan trang + filter">
---

# /be-endpoint $ARGUMENTS

Build **$ARGUMENTS** the way this repo already does it. Templates to copy: `patterns.md` in this directory.
Existing routes, guards and DTOs: `endpoints.md`. Envelope, error codes and enums: `../../context/contract.md`.

## 0. Before touching anything

- Check the route does not already exist (and is not almost-existing under a different path):
  `grep -nE '\.(GET|POST|PUT|PATCH|DELETE|Match)\(' internal/router/router.go`
- Decide the access level, because it decides the group: public / optional-auth / authenticated /
  staff+admin / admin / credential-less callback.
- Decide whether the response is a single object, a **paged** `{items, meta}` list, or a bare array. Prefer paged
  for anything that can grow — several existing list endpoints return a bare array and the frontend hates it.

## 1. `internal/dto/<domain>.go`

Request/response structs with `json` + `binding` tags (list queries use `form` and embed `dto.PageQuery`), plus a
`NewXResponse(*models.X) XResponse` mapper. Only stock validator tags exist — see `../../rules/handler-layer.md`.

## 2. `internal/models/` + `migrations/schema/` — only if a new table or column is needed

`make migrate-create name=<module>_<change>`. Never edit an applied migration. There is no `AutoMigrate`.

## 3. `internal/repository/<domain>_repository.go`

Add the method to the exported interface **and** the unexported impl. Write methods take `db *gorm.DB` as a
parameter so the service can pass its transaction. Translate `gorm.ErrRecordNotFound` to an `apperrors` sentinel
here; wrap everything else with `fmt.Errorf("verb phrase: %w", err)`.

## 4. `internal/service/<domain>_service.go`

Add the method to the interface **and** the impl. Business rules live here. Open
`s.db.Transaction(func(tx *gorm.DB) error {...})` for anything that writes, and write the success audit row inside
it with `audit.FromContext(ctx)` + `audit.In(ctx, tx, rec)`. Return a DTO, never a model.

## 5. `internal/handlers/<domain>_handler.go`

A method on the existing handler struct: bind, call one service method with `c.Request.Context()`, answer through
`pkg/response`. Add the full swagger godoc block. Copy the shape from `patterns.md` exactly.

## 6. `internal/router/router.go`

Register in the right group. Per-route middleware order is **`RateLimit` -> `Audit` -> `RequireRoles` -> handler**;
`Audit` must come before `RequireRoles` so a 403 is still recorded. Action name is `<namespace>.<verb>` and must
match the string the service uses for the success row.

## 7. Only when you add a brand-new handler TYPE

Three places, or the build/tests break:
`internal/router/router.go` `Handlers` struct, `cmd/server/main.go` `router.Handlers{...}` literal, and
`internal/service/http_test.go` `buildEngine`. Adding a method to an existing handler needs none of this.

## 8. `make swag`

Regenerates `docs/`. `swag v1.16.4` is already installed at `$(go env GOPATH)/bin/swag` and the Makefile calls it
by absolute path, so this installs nothing and does not need `swag` on `PATH`. Never hand-edit `docs/`.

## 9. Tests

- Service behaviour: `internal/service/<domain>_test.go`, `package service_test`, `e := newEnv(t)`.
- End-to-end HTTP: `internal/service/http_test.go`, `h := newHTTPEnv(t)`.
- Both need a real Postgres and self-skip without one.

```bash
make lint
go test ./internal/handlers/... ./internal/middleware/... ./pkg/... -count=1   # no infra needed
make test                                                                      # needs Postgres + RabbitMQ
```

## 10. Report

In Vietnamese: the final `METHOD /path`, its guard, the request and response shape, the statuses it can answer,
which files you touched, whether `make swag` ran, and which tests actually ran versus were skipped. If the
endpoint changes or adds a DTO the frontend consumes, say so explicitly — `FrontEnd-CP/src/types` is hand-written.
