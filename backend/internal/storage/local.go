package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Local stores images on disk; the router serves Dir under MediaPath.
type Local struct {
	dir     string
	baseURL string
}

func NewLocal(dir, publicBaseURL string) *Local {
	return &Local{dir: dir, baseURL: strings.TrimRight(publicBaseURL, "/")}
}

func (l *Local) Name() string { return "local" }

// Dir is the directory served under MediaPath.
func (l *Local) Dir() string { return l.dir }

func (l *Local) Upload(_ context.Context, img Image) (string, error) {
	name := uuid.NewString() + extensionFor(img.ContentType)
	folder := filepath.Join(l.dir, img.Folder)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if err := os.WriteFile(filepath.Join(folder, name), img.Data, 0o644); err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return l.baseURL + MediaPath + "/" + img.Folder + "/" + name, nil
}
