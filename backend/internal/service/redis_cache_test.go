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