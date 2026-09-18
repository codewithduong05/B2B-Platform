package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
)

var (
	defaultLogger *slog.Logger
	once          sync.Once
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
)

func Init(level, format string, serviceName string) *slog.Logger {
	once.Do(func() {
		var handler slog.Handler
		opts := &slog.HandlerOptions{
			Level: parseLevel(level),
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.Attr{Key: "timestamp", Value: a.Value}
				}
				if a.Key == slog.LevelKey {
					return slog.Attr{Key: "level", Value: a.Value}
				}
				return a
			},
			AddSource: true,
		}

		var writer io.Writer = os.Stdout
		if format == "json" {
			handler = slog.NewJSONHandler(writer, opts)
		} else {
			handler = slog.NewTextHandler(writer, opts)
		}

		defaultLogger = slog.New(handler.WithAttrs([]slog.Attr{
			slog.String("service", serviceName),
			slog.String("go_version", runtime.Version()),
		}))
		slog.SetDefault(defaultLogger)
	})
	return defaultLogger
}

func Default() *slog.Logger {
	if defaultLogger == nil {
		return slog.Default()
	}
	return defaultLogger
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestID(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

func With(ctx context.Context, args ...any) *slog.Logger {
	l := Default()
	if requestID := RequestID(ctx); requestID != "" {
		l = l.With(slog.String("request_id", requestID))
	}
	if len(args) > 0 {
		l = l.With(args...)
	}
	return l
}

func Debug(ctx context.Context, msg string, args ...any) {
	With(ctx).DebugContext(ctx, msg, args...)
}

func Info(ctx context.Context, msg string, args ...any) {
	With(ctx).InfoContext(ctx, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	With(ctx).WarnContext(ctx, msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	With(ctx).ErrorContext(ctx, msg, args...)
}

func ErrorWithErr(ctx context.Context, msg string, err error, args ...any) {
	if err == nil {
		With(ctx).ErrorContext(ctx, msg, args...)
		return
	}
	allArgs := append([]any{slog.String("error", err.Error())}, args...)
	With(ctx).ErrorContext(ctx, msg, allArgs...)
}
