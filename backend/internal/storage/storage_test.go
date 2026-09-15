package storage

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var pngBytes = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)

func TestLocal_Upload(t *testing.T) {
	dir := t.TempDir()
	store := NewLocal(dir, "http://api.test/")
	u, err := store.Upload(context.Background(), Image{Folder: "posters", ContentType: "image/png", Data: pngBytes})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(u, "http://api.test/media/posters/") || !strings.HasSuffix(u, ".png") {
		t.Fatalf("url = %s", u)
	}
	rel := strings.TrimPrefix(u, "http://api.test/media/")
	got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil || string(got) != string(pngBytes) {
		t.Fatalf("stored file: %v", err)
	}
}

func TestCloudinary_SignedUpload(t *testing.T) {
	ts := time.Unix(1757840000, 0)
	var sawSignature bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1_1/demo/image/upload" {
			http.Error(w, "bad path "+r.URL.Path, http.StatusNotFound)
			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		want := sha1.Sum([]byte("folder=cinema/posters&timestamp=1757840000" + "top-secret"))
		sawSignature = r.FormValue("signature") == hex.EncodeToString(want[:]) &&
			r.FormValue("api_key") == "key-1" && r.FormValue("folder") == "cinema/posters"
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		data, _ := io.ReadAll(file)
		if !sawSignature || string(data) != string(pngBytes) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"Invalid Signature"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"secure_url":"https://res.cloudinary.com/demo/image/upload/v1/cinema/posters/abc.png"}`))
	}))
	defer srv.Close()

	store := NewCloudinary(CloudinaryOptions{
		CloudName: "demo", APIKey: "key-1", APISecret: "top-secret", Folder: "cinema",
		BaseURL: srv.URL, Now: func() time.Time { return ts },
	})
	u, err := store.Upload(context.Background(), Image{Folder: "posters", Filename: "p.png", ContentType: "image/png", Data: pngBytes})
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://res.cloudinary.com/demo/image/upload/v1/cinema/posters/abc.png" || !sawSignature {
		t.Fatalf("url = %s signature ok = %v", u, sawSignature)
	}

	bad := NewCloudinary(CloudinaryOptions{CloudName: "demo", APIKey: "key-1", APISecret: "wrong", Folder: "cinema",
		BaseURL: srv.URL, Now: func() time.Time { return ts }})
	if _, err := bad.Upload(context.Background(), Image{Folder: "posters", ContentType: "image/png", Data: pngBytes}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("rejected upload: err = %v", err)
	}
}
