package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/cache"
)

// catalogGenKey: generation embedded in catalog keys; bump invalidates all lists
// at once without SCAN (old keys age out via TTL).
const catalogGenKey = "catalog:gen"

// catalogGeneration reads current generation; miss or Redis error reads as 0 (fail-open).
func catalogGeneration(ctx context.Context, c *cache.Cache) int64 {
	v, ok, err := c.Get(ctx, catalogGenKey)
	if err != nil || !ok {
		return 0
	}
	n, _ := strconv.ParseInt(v, 10, 64)
	return n
}

// bumpCatalog invalidates catalog lists. Call after commit, never inside tx:
// bumping first lets a reader repopulate cache with pre-write value.
func bumpCatalog(ctx context.Context, c *cache.Cache) {
	_, _ = c.Incr(ctx, catalogGenKey)
}

func movieListKey(gen int64, includeDrafts bool, status, genre, sort, order, search string, page, pageSize int) string {
	return fmt.Sprintf("catalog:%d:movies:list:%v:%s:%s:%s:%s:%s:%d:%d",
		gen, includeDrafts, status, genre, sort, order, search, page, pageSize)
}

func showtimeListKey(gen int64, movieID, date string) string {
	return fmt.Sprintf("catalog:%d:showtimes:list:%s:%s", gen, movieID, date)
}
