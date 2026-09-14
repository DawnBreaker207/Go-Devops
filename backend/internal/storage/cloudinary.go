package storage

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CloudinaryOptions configures the Cloudinary store.
type CloudinaryOptions struct {
	CloudName string
	APIKey    string
	APISecret string
	// Folder prefixes every upload, e.g. "cinema".
	Folder string
	// BaseURL defaults to https://api.cloudinary.com (overridden in tests).
	BaseURL    string
	HTTPClient *http.Client
	Now        func() time.Time
}

// Cloudinary uploads through Cloudinary's signed upload REST API, no SDK.
type Cloudinary struct {
	opts CloudinaryOptions
}

func NewCloudinary(opts CloudinaryOptions) *Cloudinary {
	if opts.BaseURL == "" {
		opts.BaseURL = "https://api.cloudinary.com"
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Cloudinary{opts: opts}
}

func (c *Cloudinary) Name() string { return "cloudinary" }

func (c *Cloudinary) Upload(ctx context.Context, img Image) (string, error) {
	params := map[string]string{
		"folder":    strings.Trim(path.Join(c.opts.Folder, img.Folder), "/"),
		"timestamp": strconv.FormatInt(c.opts.Now().Unix(), 10),
	}

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for key, value := range params {
		_ = form.WriteField(key, value)
	}
	_ = form.WriteField("api_key", c.opts.APIKey)
	_ = form.WriteField("signature", c.Sign(params))
	filename := img.Filename
	if filename == "" {
		filename = "upload" + extensionFor(img.ContentType)
	}
	part, err := form.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if _, err := part.Write(img.Data); err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if err := form.Close(); err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	endpoint := fmt.Sprintf("%s/v1_1/%s/image/upload", strings.TrimRight(c.opts.BaseURL, "/"), url.PathEscape(c.opts.CloudName))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	req.Header.Set("Content-Type", form.FormDataContentType())

	resp, err := c.opts.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	var out struct {
		SecureURL string `json:"secure_url"`
		Error     struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	_ = json.Unmarshal(raw, &out)
	if resp.StatusCode != http.StatusOK || out.SecureURL == "" {
		return "", fmt.Errorf("%w: cloudinary HTTP %d: %s", ErrUnavailable, resp.StatusCode, out.Error.Message)
	}
	return out.SecureURL, nil
}

// Sign is Cloudinary's API signature: parameters sorted by name as key=value
// joined with "&", followed by the API secret, SHA-1, hex encoded.
func (c *Cloudinary) Sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+"="+params[key])
	}
	sum := sha1.Sum([]byte(strings.Join(pairs, "&") + c.opts.APISecret))
	return hex.EncodeToString(sum[:])
}
