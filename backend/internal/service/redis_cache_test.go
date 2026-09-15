package service_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/cache"
)

func redisAddr() string {
	if a := os.Getenv("TEST_REDIS_ADDR"); a != "" {
		return a
	}
	return "localhost:6379"
}

// T54: public movie reads are served from Redis until a write invalidates them.
func TestMovies_CachedUntilWriteInvalidates(t *testing.T) {
	e := newEnv(t)
	c := cache.New(redisAddr(), "", 0)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Ping(ctx); err != nil {
		t.Skipf("redis unavailable at %s: %v", redisAddr(), err)
	}
	t.Cleanup(func() { _ = c.Close() })

	svc := service.NewMovieService(e.db, repository.NewMovieRepository(e.db), c, time.Minute)

	req := dto.MovieRequest{
		Title: "Cached Title", Genre: "Drama", Duration: 90, Director: "Tester",
		ReleaseDate: time.Now().Format(dto.DateLayout), Status: models.MovieStatusShowing,
	}
	made, err := svc.Create(e.ctx, req)
	e.must(err)
	if _, ok, err := c.Get(e.ctx, "movie:"+made.ID); err != nil || ok {
		t.Fatalf("fresh create must not leave a stale cache entry (ok=%v err=%v)", ok, err)
	}

	first, err := svc.GetByID(e.ctx, made.ID, false)
	e.must(err)
	if first.Title != "Cached Title" {
		t.Fatalf("first read = %q", first.Title)
	}

	e.must(e.db.Model(&models.Movie{}).Where("id = ?", made.ID).Update("title", "Changed Behind Cache").Error)
	second, err := svc.GetByID(e.ctx, made.ID, false)
	e.must(err)
	if second.Title != first.Title {
		t.Fatalf("second read should come from cache, got %q want %q", second.Title, first.Title)
	}

	req.Title = "Updated Title"
	if _, err := svc.Update(e.ctx, made.ID, req); err != nil {
		t.Fatalf("update: %v", err)
	}
	after, err := svc.GetByID(e.ctx, made.ID, false)
	e.must(err)
	if after.Title != "Updated Title" {
		t.Fatalf("read after update = %q, cache was not invalidated", after.Title)
	}
}

// T54: List() is cached too (not just GetByID), and a write bumps the shared
// generation so every cached list, not just this one query, is invalidated.
func TestMovies_ListCachedUntilWriteInvalidates(t *testing.T) {
	e := newEnv(t)
	c := cache.New(redisAddr(), "", 0)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Ping(ctx); err != nil {
		t.Skipf("redis unavailable at %s: %v", redisAddr(), err)
	}
	t.Cleanup(func() { _ = c.Close() })

	svc := service.NewMovieService(e.db, repository.NewMovieRepository(e.db), c, time.Minute)
	query := dto.MovieListQuery{PageQuery: dto.PageQuery{Page: 1, PageSize: 50}}

	first, total, err := svc.List(e.ctx, query, false)
	e.must(err)

	e.must(e.db.Model(&models.Movie{}).Where("id = ?", e.movieID).Update("title", "Changed Behind Cache").Error)
	second, total2, err := svc.List(e.ctx, query, false)
	e.must(err)
	if total2 != total || len(second) != len(first) {
		t.Fatalf("second list should still be the cached one: %d/%d vs %d/%d", len(second), total2, len(first), total)
	}
	for _, m := range second {
		if m.Title == "Changed Behind Cache" {
			t.Fatal("list read the DB despite a warm cache")
		}
	}

	if _, err := svc.Create(e.ctx, dto.MovieRequest{
		Title: "New In List", Genre: "Drama", Duration: 90, Director: "Tester",
		ReleaseDate: time.Now().Format(dto.DateLayout), Status: models.MovieStatusShowing,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	third, total3, err := svc.List(e.ctx, query, false)
	e.must(err)
	if total3 != total+1 || len(third) == len(first) {
		t.Fatalf("list after create = %d/%d, want %d/%d", len(third), total3, len(first)+1, total+1)
	}
}

// T54: showtime listings (public, by date) are cached and a write — a new
// showtime, or a hall price change that moves from_price — bumps them.
func TestShowtimes_CachedUntilWriteInvalidates(t *testing.T) {
	e := newEnv(t)
	c := cache.New(redisAddr(), "", 0)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Ping(ctx); err != nil {
		t.Skipf("redis unavailable at %s: %v", redisAddr(), err)
	}
	t.Cleanup(func() { _ = c.Close() })

	hallRepo := repository.NewHallRepository(e.db)
	halls := service.NewHallService(e.db, hallRepo, c)
	showtimes := service.NewShowtimeService(e.db, repository.NewShowtimeRepository(e.db), hallRepo,
		repository.NewMovieRepository(e.db), 20, time.UTC, c, time.Minute)

	var show models.Showtime
	e.must(e.db.First(&show, "id = ?", e.showID).Error)
	day := show.StartAt.UTC().Format(dto.DateLayout)

	first, err := showtimes.ListByDate(e.ctx, day)
	e.must(err)
	if len(first) != 1 || first[0].FromPrice != priceStandard {
		t.Fatalf("first list = %+v", first)
	}

	// Directly raising the DB price must not show up while the cache is warm.
	e.must(e.db.Exec(`UPDATE hall_prices SET price = price * 10 WHERE hall_id = ? AND seat_type = 'standard'`, e.hallID).Error)
	second, err := showtimes.ListByDate(e.ctx, day)
	e.must(err)
	if second[0].FromPrice != first[0].FromPrice {
		t.Fatalf("second list should be the cached one: from_price %d, want %d", second[0].FromPrice, first[0].FromPrice)
	}

	// SetPrices through the service bumps the shared generation.
	if _, err := halls.SetPrices(e.ctx, e.hallID, dto.PriceRequest{Prices: fullPrices()}); err != nil {
		t.Fatalf("set prices: %v", err)
	}
	third, err := showtimes.ListByDate(e.ctx, day)
	e.must(err)
	if third[0].FromPrice != priceStandard {
		t.Fatalf("list after price change = %d, want the reset price %d", third[0].FromPrice, priceStandard)
	}
}

// T53: with Redis unreachable (not just absent — a dead address the client
// actually tries and fails to reach), catalog reads still succeed from
// Postgres. Unlike the tests above, this one never skips: it needs no real
// Redis, only an address nothing answers on.
func TestMovies_ReadsSucceedWhenRedisUnreachable(t *testing.T) {
	e := newEnv(t)
	c := cache.New("127.0.0.1:1", "", 0)
	t.Cleanup(func() { _ = c.Close() })

	svc := service.NewMovieService(e.db, repository.NewMovieRepository(e.db), c, time.Minute)
	if _, err := svc.GetByID(e.ctx, e.movieID, false); err != nil {
		t.Fatalf("GetByID with redis down: %v", err)
	}
	if _, _, err := svc.List(e.ctx, dto.MovieListQuery{PageQuery: dto.PageQuery{Page: 1, PageSize: 10}}, false); err != nil {
		t.Fatalf("List with redis down: %v", err)
	}
	if _, err := svc.Create(e.ctx, dto.MovieRequest{
		Title: "Survives Redis Down", Genre: "Drama", Duration: 90, Director: "Tester",
		ReleaseDate: time.Now().Format(dto.DateLayout), Status: models.MovieStatusShowing,
	}); err != nil {
		t.Fatalf("Create with redis down: %v", err)
	}
}