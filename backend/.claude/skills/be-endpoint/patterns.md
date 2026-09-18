# Copyable patterns — BackEnd-CP

Every snippet below is either copied verbatim from this repo or is the skeleton those files share. When in doubt,
open the cited file and imitate it rather than this page — the code is the truth.

Canonical files to read before writing: `internal/handlers/auth_handler.go`, `internal/service/movie_service.go`,
`internal/repository/movie_repository.go`, `internal/dto/movie.go`, `internal/router/router.go`.

## 1. Handler — verbatim from `internal/handlers/auth_handler.go`

```go
// Register godoc
//
//	@Summary		Register account
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		dto.RegisterRequest	true	"Registration details"
//	@Success		201		{object}	response.Body{data=dto.UserResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, user)
}
```

Add `@Security BearerAuth` for an authenticated endpoint. Path in `@Router` is relative to the `/api/v1` base.

## 2. Handler — paged list

```go
// List godoc
//
//	@Summary		List movies
//	@Tags			movies
//	@Security		BearerAuth
//	@Produce		json
//	@Param			page		query		int		false	"Page"	default(1)
//	@Param			page_size	query		int		false	"Page size"	default(10)
//	@Param			search		query		string	false	"Search by title"
//	@Success		200			{object}	response.Body{data=response.Paged{items=[]dto.MovieResponse}}
//	@Failure		400			{object}	response.Body
//	@Router			/movies [get]
func (h *MovieHandler) List(c *gin.Context) {
	var query dto.MovieListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	query.Normalize()

	// canSeeDrafts(c) is the draft-visibility security boundary — customers never see drafts.
	items, total, err := h.movieService.List(c.Request.Context(), query, canSeeDrafts(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.List(c, items, query.Page, query.PageSize, total)
}
```

`query.Normalize()` must be the line right after a successful bind.

## 3. Handler constructor

```go
type MovieHandler struct {
	movieService service.MovieService
}

func NewMovieHandler(movieService service.MovieService) *MovieHandler {
	return &MovieHandler{movieService: movieService}
}
```

## 4. DTO

```go
type MovieRequest struct {
	Title       string `json:"title" binding:"required,max=255"`
	Duration    int    `json:"duration" binding:"required,min=1,max=600"`
	ReleaseDate string `json:"release_date" binding:"required,datetime=2006-01-02"`
	PosterURL   string `json:"poster_url" binding:"omitempty,url"`
	Status      string `json:"status" binding:"omitempty,oneof=draft showing ended"`
}

type MovieListQuery struct {
	PageQuery // unqualified: this file IS package dto. page, page_size, search + Normalize() + Offset()
	Status string `form:"status" binding:"omitempty,oneof=draft showing ended"`
	Sort   string `form:"sort" binding:"omitempty,oneof=release_date title created_at"`
	Order  string `form:"order" binding:"omitempty,oneof=asc desc"`
}

func NewMovieResponse(m *models.Movie) MovieResponse { /* field-by-field */ }

func NewMovieResponses(list []models.Movie) []MovieResponse {
	out := make([]MovieResponse, 0, len(list))
	for i := range list {
		out = append(out, NewMovieResponse(&list[i]))
	}
	return out
}
```

Response structs carry no `binding` tag. Money is `int64`, IDs are `string`, times are `time.Time`.

## 5. Service — shape shared by `internal/service/*_service.go`

```go
type MovieService interface {
	// The includeDrafts flag is real and load-bearing: customers must never see drafts.
	List(ctx context.Context, query dto.MovieListQuery, includeDrafts bool) ([]dto.MovieResponse, int64, error)
	Create(ctx context.Context, req dto.MovieRequest) (*dto.MovieResponse, error)
}

type movieService struct {
	db        *gorm.DB
	movieRepo repository.MovieRepository
	cache     *cache.Cache // a nil cache is a legal no-op — Redis is optional
	cacheTTL  time.Duration
}

func NewMovieService(db *gorm.DB, repo repository.MovieRepository, c *cache.Cache, ttl time.Duration) MovieService {
	// Do not drop ttl: a zero cacheTTL means entries never expire.
	return &movieService{db: db, movieRepo: repo, cache: c, cacheTTL: ttl}
}

func (s *movieService) Create(ctx context.Context, req dto.MovieRequest) (*dto.MovieResponse, error) {
	m := models.Movie{Title: strings.TrimSpace(req.Title)} // normalisation belongs HERE, not in the handler
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.Create(ctx, tx, &m); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = m.ID
			// Before/After are map[string]any — a model does not compile here and
			// would leak json:"-" fields. Pick the fields that matter.
			rec.After = map[string]any{"title": m.Title, "status": m.Status}
			return audit.In(ctx, tx, rec)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	out := dto.NewMovieResponse(&m)
	return &out, nil
}
```

Return the interface from the constructor. Return DTOs, never models.

## 6. Repository — shape shared by `internal/repository/*_repository.go`

```go
type MovieRepository interface {
	FindByID(ctx context.Context, id string) (*models.Movie, error)
	Create(ctx context.Context, db *gorm.DB, m *models.Movie) error
}

type movieRepository struct{ db *gorm.DB }

func NewMovieRepository(db *gorm.DB) MovieRepository { return &movieRepository{db: db} }

func (r *movieRepository) FindByID(ctx context.Context, id string) (*models.Movie, error) {
	var m models.Movie
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrMovieNotFound   // sentinel, at the query site
		}
		return nil, fmt.Errorf("find movie: %w", err)
	}
	return &m, nil
}

func (r *movieRepository) Create(ctx context.Context, db *gorm.DB, m *models.Movie) error {
	if err := db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("create movie: %w", err)
	}
	return nil
}
```

Write methods take the `*gorm.DB` to use. Read methods use `r.db`.

## 7. Route registration — verbatim from `internal/router/router.go`

```go
v1 := engine.Group("/api/v1")
v1.Use(middleware.DBGuard(db, 500*time.Millisecond), middleware.NoStore())

protected := v1.Group("")
protected.Use(middleware.Auth(jwtManager, accounts))
{
	orders := protected.Group("/orders")
	{
		orders.GET("", h.Booking.List)
		orders.POST("/hold", middleware.RateLimit(limits.Hold), middleware.Audit(db, "orders.hold", "booking"),
			middleware.RequireRoles(models.RoleCustomer), h.Booking.Hold)
		orders.POST("/:id/pay", middleware.Audit(db, "orders.pay", "booking"),
			middleware.RequireRoles(models.RoleCustomer), h.Booking.Pay)
	}
}

// Variant B — role enforced once for a whole subgroup instead of per route:
staff := protected.Group("/staff")
staff.Use(middleware.RequireRoles(models.RoleStaff, models.RoleAdmin))
{
	staff.GET("/dashboard", h.Staff.Dashboard)
	staff.POST("/orders", middleware.Audit(db, "orders.counter_sell", "booking"), h.Staff.CounterSell)
}
```

Available groups: `auth` (IP rate-limited, public), `public` (`RateLimit` + `OptionalAuth`), `protected`
(`Auth` required), a `RequireRoles` subgroup (`staff`, `admin`), and the bare `v1` group for credential-less
callbacks (SSE stream, payment IPN/return).

## 8. New error sentinel — `pkg/errors/errors.go`

```go
var (
	ErrMovieNotFound = NotFound("movie not found")
	ErrSeatTaken     = Conflict("one or more seats are no longer available")
)
```

Constructors: `BadRequest Validation Unauthorized TokenExpired Forbidden NotFound Conflict PayloadTooLarge
PreconditionRequired TooManyRequests Internal BadGateway ServiceUnavailable`. Add detail with
`.WithDetails(map[string]string{"retry_after_seconds": "1"})` — it returns a clone, the sentinel is untouched.

## 9. Service test

```go
// SPEC F2-03: a draft movie is invisible to a guest.
func TestMovieService_ListHidesDraftFromGuest(t *testing.T) {
	e := newEnv(t)          // skips when Postgres is unavailable; wipes + seeds the DB
	...
	if !isAppErr(err, apperrors.ErrMovieNotFound) {
		t.Fatalf("want ErrMovieNotFound, got %v", err)
	}
}
```

`package service_test`. Helpers: `e.must`, `e.count`, `e.wantStatus`, `e.wantSeat`, `e.checkInvariants`,
`httpStatus(err)`, `isAppErr(err, sentinel)`. Router-level tests use `newHTTPEnv(t)` and are named `TestHTTP_*`.
