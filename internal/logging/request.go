package logging

import (
	"net"
	"net/http"
	"strings"
	"time"
)

// RequestLogger decorates an http.Handler to provide structured request logs.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		logger := Default().With(
			"method", r.Method,
			"path", r.URL.Path,
		)

		if rid := headerValue(r, "X-Request-Id", "X-Correlation-Id"); rid != "" {
			logger = logger.With("request_id", rid)
		}

		if remote := remoteAddr(r); remote != "" {
			logger = logger.With("remote_addr", remote)
		}

		ctx := WithContext(r.Context(), logger)
		recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r.WithContext(ctx))

		duration := time.Since(start)

		switch {
		case recorder.status >= 500:
			logger.Error("request completed",
				"status", recorder.status,
				"duration_ms", duration.Milliseconds(),
				"bytes_written", recorder.bytes,
			)
		case recorder.status >= 400:
			logger.Warn("request completed",
				"status", recorder.status,
				"duration_ms", duration.Milliseconds(),
				"bytes_written", recorder.bytes,
			)
		default:
			logger.Debug("request completed",
				"status", recorder.status,
				"duration_ms", duration.Milliseconds(),
				"bytes_written", recorder.bytes,
			)
		}
	})
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func headerValue(r *http.Request, keys ...string) string {
	for _, key := range keys {
		if val := strings.TrimSpace(r.Header.Get(key)); val != "" {
			return val
		}
	}
	return ""
}

func remoteAddr(r *http.Request) string {
	if xfwd := headerValue(r, "X-Forwarded-For"); xfwd != "" {
		parts := strings.Split(xfwd, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}
