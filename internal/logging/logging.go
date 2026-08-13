package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"
)

type contextKey struct{}

var (
	defaultLogger *slog.Logger
	once          sync.Once
)

// Setup configures the process-wide default logger based on the given level
// and format. Uses sync.Once so it only applies the first call.
func Setup(level, format string) *slog.Logger {
	once.Do(func() {
		lvl, ok := parseLevel(level)
		if !ok {
			lvl = slog.LevelWarn
		}

		levelVar := new(slog.LevelVar)
		levelVar.Set(lvl)

		opts := &slog.HandlerOptions{
			Level: levelVar,
		}

		var handler slog.Handler = slog.NewJSONHandler(os.Stdout, opts)
		if strings.EqualFold(format, "text") {
			handler = slog.NewTextHandler(os.Stdout, opts)
		}

		defaultLogger = slog.New(handler).With(
			"service", "mm-mcp-server",
			"pid", os.Getpid(),
		)
		slog.SetDefault(defaultLogger)
	})

	return defaultLogger
}

// Default returns the configured logger, calling Setup() if necessary.
func Default() *slog.Logger {
	if defaultLogger == nil {
		return Setup("warn", "json")
	}
	return defaultLogger
}

// WithContext stores logger in a context for downstream handlers.
func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext retrieves a logger from a context or falls back to Default.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(contextKey{}).(*slog.Logger); ok && logger != nil {
			return logger
		}
	}
	return Default()
}

func parseLevel(raw string) (slog.Level, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return slog.LevelInfo, false
	case "debug":
		return slog.LevelDebug, true
	case "info":
		return slog.LevelInfo, true
	case "warn", "warning":
		return slog.LevelWarn, true
	case "error":
		return slog.LevelError, true
	default:
		return slog.LevelInfo, false
	}
}
