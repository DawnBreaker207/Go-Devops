package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders sets defensive headers on every response (L7).
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

// NoStore marks API answers as not cacheable by browsers or shared proxies;
// handlers with their own caching policy override it.
func NoStore() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}

// BodyLimit caps request bodies at n bytes; reading past it fails and the
// request answers 413. Multipart uploads enforce their own, larger limit.
func BodyLimit(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if n > 0 && c.Request.Body != nil && !strings.HasPrefix(c.ContentType(), "multipart/") {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		}
		c.Next()
	}
}
