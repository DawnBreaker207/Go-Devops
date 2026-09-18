---
paths:
  - "**/internal/handlers/**/*.go"
  - "**/internal/dto/**/*.go"
---

# Handler + DTO layer rules

## A handler does exactly four things

```go
func (h *XHandler) Create(c *gin.Context) {
	var req dto.XRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.xService.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, out)
}
```

- Receiver is always `h`. Constructor is `NewXHandler(svc service.XService) *XHandler` returning a pointer.
- Bind with `ShouldBindJSON` (body) or `ShouldBindQuery` (query string). **Never** `BindJSON` or `MustBindWith`.
- Pass `c.Request.Context()` down. Never pass `c`, never `context.Background()`.
- The only error statement is `response.Error(c, err); return` — no status, no message, no logging.
- Answer through `response.OK` / `Created` / `NoContentOK` / `List` only. Raw `c.JSON` exists in this repo for
  exactly two cases (202 in `batch_handler.go`, 503 in `health_handler.go`) and still builds a `response.Body`.
- Read the caller with `middleware.CurrentUserID(c)` / `CurrentUserRole(c)`, never raw `c.Get`.
- No `gorm.io/gorm` import, no query, no `TrimSpace`, no business `if` chains. Those belong in the service.

## Paged lists

```go
var query dto.XListQuery
if err := c.ShouldBindQuery(&query); err != nil { response.Error(c, err); return }
query.Normalize()   // MUST be the next line: unclamped Page/PageSize produce a negative OFFSET
items, total, err := h.xService.List(c.Request.Context(), query)
...
response.List(c, items, query.Page, query.PageSize, total)
```

Never build a `response.Paged` or `Meta` by hand — `response.List` computes `total_pages`.

## DTOs

- Names end in `Request` (inbound), `Response` (outbound) or `Query` (query string). Response structs carry no
  `binding` tag and never expose a password.
- Validate with stock validator tags only: `required omitempty min=N max=N email url uuid oneof=a b datetime=2006-01-02 dive`.
  **There are no custom validation tags in this repo** — `internal/router/validator.go` only maps tag names, so a
  tag that is not built into go-playground/validator will silently not work.
- `details` keys in a 400 come from the `json` tag (then `form`), so name tags the way the frontend should see them.
- Embed `dto.PageQuery` in every list query and take the offset from its `Offset()` method.
- Add a `NewXResponse(*models.X) XResponse` (+ `NewXResponses`) mapper here and call it from the service.
- Money is `int64` whole VND. IDs are `string` (UUID). Times are `time.Time` except the deliberate
  `YYYY-MM-DD` string dates (`release_date`, report `date`/`from`/`to`).

## Swagger annotation block

```go
// Create godoc
//
//	@Summary		Create movie
//	@Tags			movies
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		dto.MovieRequest	true	"Movie"
//	@Success		201		{object}	response.Body{data=dto.MovieResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/movies [post]
```

- Mandatory: `@Summary`, `@Tags`, `@Success`, `@Router`. Path is relative to the `/api/v1` base.
- `@Security BearerAuth` on every authenticated endpoint; omit it for `/auth/*`, the public catalog reads and the
  provider webhooks. `BearerAuth` is the only security name.
- Wrap the payload in the envelope: `response.Body{data=...}`, or `response.Body{data=response.Paged{items=[]dto.X}}`,
  or bare `response.Body` when there is no data.
- One `@Failure` line per status that can really happen, always `{object} response.Body`. No handler declares 500.
- Run `make swag` afterwards. Do not edit `docs/` by hand.
