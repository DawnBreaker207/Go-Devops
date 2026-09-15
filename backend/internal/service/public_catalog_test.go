package service_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/ratelimit"
)

// T52 / E-CAT6 / E-CAT8: the catalog is public without drafts; buying needs an account; a bad token is 401.
func TestHTTP_PublicCatalog(t *testing.T) {
	h := newHTTPEnv(t)
	draft := &models.Movie{Title: "Draft Cut", Genre: "Drama", Duration: 90, Director: "Tester",
		ReleaseDate: time.Now(), Status: models.MovieStatusDraft}
	h.must(h.db.Create(draft).Error)
	var show models.Showtime
	h.must(h.db.First(&show, "id = ?", h.showID).Error)
	day := show.StartAt.UTC().Format(dto.DateLayout)

	for _, path := range []string{
		"/api/v1/movies",
		"/api/v1/movies/" + h.movieID,
		"/api/v1/movies/" + h.movieID + "/showtimes?date=" + day,
		"/api/v1/showtimes?date=" + day,
	} {
		if status, _, body := h.call(http.MethodGet, path, "", nil); status != http.StatusOK {
			t.Errorf("anonymous %s: HTTP %d %v", path, status, body["message"])
		}
	}
	if status, _, _ := h.call(http.MethodGet, "/api/v1/movies/"+draft.ID, "", nil); status != http.StatusNotFound {
		t.Errorf("anonymous draft: HTTP %d, want 404", status)
	}
	admin, _ := h.login(models.RoleAdmin)
	if status, _, _ := h.call(http.MethodGet, "/api/v1/movies/"+draft.ID, admin, nil); status != http.StatusOK {
		t.Errorf("admin draft on the public route: HTTP %d, want 200", status)
	}

	for name, c := range map[string]struct {
		method, path string
		body         any
	}{
		"seat map":       {http.MethodGet, "/api/v1/shows/" + h.showID + "/seats", nil},
		"hold":           {http.MethodPost, "/api/v1/orders/hold", map[string]any{"show_id": h.showID, "seat_ids": h.ids("A1")}},
		"realtime token": {http.MethodGet, "/api/v1/events/token?show_id=" + h.showID, nil},
		"create movie":   {http.MethodPost, "/api/v1/movies", map[string]any{"title": "Nope"}},
	} {
		if status, _, _ := h.call(c.method, c.path, "", c.body); status != http.StatusUnauthorized {
			t.Errorf("anonymous %s: HTTP %d, want 401", name, status)
		}
	}
	if status, _, _ := h.call(http.MethodGet, "/api/v1/movies", "not-a-jwt", nil); status != http.StatusUnauthorized {
		t.Errorf("wrong token on a public route: HTTP %d, want 401", status)
	}
}

// E-CAT7: the public catalog is throttled per client IP.
func TestHTTP_PublicCatalogRateLimited(t *testing.T) {
	h := newHTTPEnv(t)
	h.publicLimiter = ratelimit.New(3, 0.001)
	srv := httptest.NewServer(h.buildEngine(h.db))
	defer srv.Close()

	for i := 1; i <= 4; i++ {
		status, _, _ := h.callAt(srv.URL, http.MethodGet, "/api/v1/movies", "", nil)
		want := http.StatusOK
		if i == 4 {
			want = http.StatusTooManyRequests
		}
		if status != want {
			t.Fatalf("request %d: HTTP %d, want %d", i, status, want)
		}
	}
}
