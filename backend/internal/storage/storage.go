// Package storage saves uploaded images locally or on Cloudinary; the database keeps only the URL.
package storage

import (
	"context"
	"errors"
)

const MediaPath = "/media"

// ErrUnavailable wraps every failure to store an image.
var ErrUnavailable = errors.New("image storage unavailable")

type Image struct {
	Folder      string
	Filename    string // original name, informative only
	ContentType string // sniffed, never trusted from the client
	Data        []byte
}

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
