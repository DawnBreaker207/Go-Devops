package service_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/storage"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

func TestUploadPoster(t *testing.T) {
	dir := t.TempDir()
	svc := service.NewMediaService(storage.NewLocal(dir, "http://api.test"), 1024)
	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{1}, 100)...)

	res, err := svc.UploadPoster(context.Background(), "poster.png", bytes.NewReader(png))
	if err != nil {
		t.Fatal(err)
	}
	if res.ContentType != "image/png" || res.Size != int64(len(png)) || !strings.HasPrefix(res.URL, "http://api.test/media/posters/") {
		t.Fatalf("upload = %+v", res)
	}
	rel := strings.TrimPrefix(res.URL, "http://api.test/media/")
	if stored, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel))); err != nil || !bytes.Equal(stored, png) {
		t.Fatalf("stored file mismatch: %v", err)
	}

	if _, err := svc.UploadPoster(context.Background(), "fake.png", strings.NewReader("hello, not an image")); !isAppErr(err, apperrors.ErrUploadInvalid) {
		t.Fatalf("fake image: err = %v", err)
	}
	big := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{1}, 2000)...)
	if _, err := svc.UploadPoster(context.Background(), "big.png", bytes.NewReader(big)); !isAppErr(err, apperrors.ErrUploadTooLarge) {
		t.Fatalf("too large: err = %v", err)
	}
}
