package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/cache"
)

// catalogGenKey is the generation counter every public-catalog list cache key
// embeds (movies, showtimes). Bumping it after a write invalidates every
// list key at once — the old ones simply age out via TTL, unread — without a
// Redis SCAN over an unbounded key space.
const catalogGenKey = "catalog:gen"

// catalogGeneration reads the current generation; a nil cache, a miss or a
// Redis error all read as 0 (fail-open — the cache key is still stable, just
// unshared with whatever wrote a later generation).
func catalogGeneration(ctx context.Context, c *cache.Cache) int64 {
	v, ok, err := c.Get(ctx, catalogGenKey)
	if err != nil || !ok {
		return 0
	}
	n, _ := strconv.ParseInt(v, 10, 64)
	return n
}

// bumpCatalog invalidates every cached catalog list/detail. Call it after a
// write commits, never before or inside the transaction: bumping first would
// let a concurrent reader repopulate the cache with the pre-write value
// before the write lands.
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
