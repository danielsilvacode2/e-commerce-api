package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

var gzipWriterPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

type gzipResponseWriter struct {
	gin.ResponseWriter
	writer      *gzip.Writer
	wroteHeader bool
	compressed  bool
}

func (g *gzipResponseWriter) WriteHeader(code int) {
	if g.wroteHeader {
		return
	}
	g.wroteHeader = true
	if code == http.StatusNoContent || code == http.StatusNotModified || code < 200 {
		g.ResponseWriter.WriteHeader(code)
		return
	}
	g.compressed = true
	g.Header().Del("Content-Length")
	g.Header().Set("Content-Encoding", "gzip")
	g.Header().Add("Vary", "Accept-Encoding")
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	if !g.wroteHeader {
		if g.Header().Get("Content-Type") == "" {
			g.Header().Set("Content-Type", http.DetectContentType(data))
		}
		g.WriteHeader(http.StatusOK)
	}
	if !g.compressed {
		return g.ResponseWriter.Write(data)
	}
	return g.writer.Write(data)
}

func (g *gzipResponseWriter) WriteString(s string) (int, error) {
	return g.Write([]byte(s))
}

func (g *gzipResponseWriter) Flush() {
	if g.compressed && g.writer != nil {
		_ = g.writer.Flush()
	}
	g.ResponseWriter.Flush()
}

// GzipMiddleware comprime respostas HTTP com gzip quando o cliente suporta Accept-Encoding: gzip.
func GzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodHead ||
			c.Request.Method == http.MethodOptions ||
			!strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") ||
			strings.HasPrefix(c.Request.URL.Path, "/swagger/") {
			c.Next()
			return
		}

		gz := gzipWriterPool.Get().(*gzip.Writer)
		gz.Reset(c.Writer)

		origWriter := c.Writer
		gzWriter := &gzipResponseWriter{
			ResponseWriter: origWriter,
			writer:         gz,
		}
		c.Writer = gzWriter

		defer func() {
			if gzWriter.compressed {
				_ = gz.Close()
			}
			gz.Reset(io.Discard)
			gzipWriterPool.Put(gz)
			c.Writer = origWriter
		}()

		c.Next()
	}
}
