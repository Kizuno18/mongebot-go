// Package logger provides structured logging with a ring buffer for UI consumption.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Setup initializes the global slog logger with the given level and outputs.
func Setup(level string, logFile string) (*slog.Logger, error) {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	writers := []io.Writer{os.Stdout}

	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, err
		}
		writers = append(writers, f)
	}

	multi := io.MultiWriter(writers...)

	opts := &slog.HandlerOptions{
		Level: lvl,
	}

	handler := slog.NewJSONHandler(multi, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger, nil
}

// WithComponent returns a logger with a "component" attribute.
func WithComponent(logger *slog.Logger, component string) *slog.Logger {
	return logger.With("component", component)
}

// WithWorker returns a logger with "worker" and "workerId" attributes.
func WithWorker(logger *slog.Logger, workerID string) *slog.Logger {
	return logger.With("worker", workerID)
}

// FromContext extracts a logger from context, or returns the default.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// WithContext stores a logger in the context.
func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

type loggerKey struct{}
