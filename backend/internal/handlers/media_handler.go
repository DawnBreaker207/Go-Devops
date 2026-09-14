package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

// MediaHandler receives image uploads (movie posters).
type MediaHandler struct {
	media    service.MediaService
	localDir string
	maxBytes int64
}

// NewMediaHandler builds the handler; localDir is served under /media when
// the local store is used, empty otherwise.
func NewMediaHandler(media service.MediaService, localDir string, maxBytes int64) *MediaHandler {
	return &MediaHandler{media: media, localDir: localDir, maxBytes: maxBytes}
}

// LocalDir is the directory to serve under /media ("" for remote stores).
func (h *MediaHandler) LocalDir() string { return h.localDir }

// UploadPoster godoc
//
//	@Summary		Upload a movie poster (admin/staff)
//	@Description	multipart/form-data field "file": JPEG, PNG or WebP. Returns the URL to put in poster_url.
//	@Tags			media
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file	formData	file	true	"Poster image"
//	@Success		201		{object}	response.Body{data=dto.UploadResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Failure		502		{object}	response.Body
//	@Router			/admin/uploads/poster [post]
func (h *MediaHandler) UploadPoster(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxBytes+1<<20)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			response.Error(c, apperrors.ErrUploadTooLarge)
			return
		}
		response.Error(c, apperrors.Validation(`multipart field "file" is required`))
		return
	}
	defer file.Close()

	res, err := h.media.UploadPoster(c.Request.Context(), header.Filename, file)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, res)
}
