package service_test

import (
	"net/http"
	"testing"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// Article CMS: public reads only see published posts, admin manages drafts,
// and slugs auto-suffix on collision.
func TestHTTP_ArticleCMS(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	staff, _ := h.login(models.RoleStaff)
	customer, _ := h.login(models.RoleCustomer)

	// Draft is invisible to the public but visible to operators.
	status, _, body := h.call(http.MethodPost, "/api/v1/admin/articles", admin, map[string]any{
		"title": "Khuyen mai thang 10", "content": "hello", "status": "draft",
	})
	if status != http.StatusCreated {
		t.Fatalf("create draft: HTTP %d %v", status, body["message"])
	}
	draft := dataMap(body)
	if draft["slug"] != "khuyen-mai-thang-10" {
		t.Fatalf("slug = %v, want khuyen-mai-thang-10", draft["slug"])
	}
	draftID, _ := draft["id"].(string)

	status, _, _ = h.call(http.MethodGet, "/api/v1/articles", "", nil)
	if status != http.StatusOK {
		t.Fatalf("public list: HTTP %d", status)
	}
	status, _, body = h.call(http.MethodGet, "/api/v1/articles/khuyen-mai-thang-10", "", nil)
	if status != http.StatusNotFound {
		t.Fatalf("public draft: HTTP %d, want 404", status)
	}

	// Same title publishes with a suffixed slug, no 409.
	status, _, body = h.call(http.MethodPost, "/api/v1/admin/articles", staff, map[string]any{
		"title": "Khuyen mai thang 10", "content": "hello", "status": "published", "type": "promotion",
	})
	if status != http.StatusCreated {
		t.Fatalf("create published: HTTP %d %v", status, body["message"])
	}
	pub := dataMap(body)
	if pub["slug"] != "khuyen-mai-thang-10-2" {
		t.Fatalf("slug = %v, want khuyen-mai-thang-10-2", pub["slug"])
	}

	// Public list shows only the published one; detail works by slug.
	status, _, body = h.call(http.MethodGet, "/api/v1/articles", "", nil)
	if status != http.StatusOK {
		t.Fatalf("public list: HTTP %d %v", status, body["message"])
	}
	meta, _ := dataMap(body)["meta"].(map[string]any)
	total, _ := meta["total"].(float64)
	if int64(total) != 1 {
		t.Fatalf("public total = %v, want 1", meta["total"])
	}
	status, _, body = h.call(http.MethodGet, "/api/v1/articles/khuyen-mai-thang-10-2", "", nil)
	if status != http.StatusOK {
		t.Fatalf("public detail: HTTP %d %v", status, body["message"])
	}

	// Customers can not write; publishing the draft makes two public.
	status, _, _ = h.call(http.MethodPost, "/api/v1/admin/articles", customer, map[string]any{
		"title": "spam", "content": "x",
	})
	if status != http.StatusForbidden {
		t.Fatalf("customer create: HTTP %d, want 403", status)
	}
	status, _, _ = h.call(http.MethodPut, "/api/v1/admin/articles/"+draftID, admin, map[string]any{
		"status": "published",
	})
	if status != http.StatusOK {
		t.Fatalf("publish draft: HTTP %d", status)
	}
	status, _, _ = h.call(http.MethodDelete, "/api/v1/admin/articles/"+draftID, admin, nil)
	if status != http.StatusNoContent {
		t.Fatalf("delete: HTTP %d, want 204", status)
	}
	status, _, body = h.call(http.MethodGet, "/api/v1/admin/articles/"+draftID, admin, nil)
	if status != http.StatusNotFound {
		t.Fatalf("read deleted: HTTP %d, want 404", status)
	}
}
