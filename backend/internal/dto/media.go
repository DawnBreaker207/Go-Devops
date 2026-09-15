package dto

// URL is meant for a movie's poster_url.
type UploadResponse struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}
