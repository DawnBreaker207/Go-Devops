// Package storage keeps uploaded images (movie posters). Like payments, the
// backend is chosen by config: "local" writes files served under /media (dev),
// "cloudinary" uploads to Cloudinary. The database only stores the URL.
package storage

import (
	"context"
	"errors"
)

// MediaPath is where the local store's files are served.
const MediaPath = "/media"

// ErrUnavailable wraps every failure to store an image.
var ErrUnavailable = errors.New("image storage unavailable")

// Image is one validated upload.
type Image struct {
	Folder      string // e.g. "posters"
	Filename    string // original name, informative only
	ContentType string // sniffed, never trusted from the client
	Data        []byte
}

// Store saves an image and returns its public URL.
type Store interface {
	Name() string
	Upload(ctx context.Context, img Image) (url string, err error)
}

func extensionFor(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}
