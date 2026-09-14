package service

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/storage"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

type MediaService interface {
	UploadPoster(ctx context.Context, filename string, body io.Reader) (*dto.UploadResponse, error)
}

type mediaService struct {
	store    storage.Store
	maxBytes int64
}

func NewMediaService(store storage.Store, maxBytes int64) MediaService {
	return &mediaService{store: store, maxBytes: maxBytes}
}

var allowedImageTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

// UploadPoster sniffs the content type itself: the client's is not trusted.
func (s *mediaService) UploadPoster(ctx context.Context, filename string, body io.Reader) (*dto.UploadResponse, error) {
	data, err := io.ReadAll(io.LimitReader(body, s.maxBytes+1))
	if err != nil {
		return nil, apperrors.BadRequest("could not read upload").Wrap(err)
	}
	if int64(len(data)) > s.maxBytes {
		return nil, apperrors.ErrUploadTooLarge.WithDetails(map[string]string{"max_bytes": strconv.FormatInt(s.maxBytes, 10)})
	}
	contentType := http.DetectContentType(data)
	if len(data) == 0 || !allowedImageTypes[contentType] {
		return nil, apperrors.ErrUploadInvalid
	}
	u, err := s.store.Upload(ctx, storage.Image{Folder: "posters", Filename: filename, ContentType: contentType, Data: data})
	if err != nil {
		return nil, apperrors.ErrImageStoreUnavailable.Wrap(err)
	}
	return &dto.UploadResponse{URL: u, ContentType: contentType, Size: int64(len(data))}, nil
}
