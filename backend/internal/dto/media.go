package dto

// UploadResponse is a stored image; put URL into poster_url.
type UploadResponse struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}
