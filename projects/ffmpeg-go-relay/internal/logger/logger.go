package logger

import (
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with structured logging capabilities.
type Logger struct {
	handler slog.Handler
	logger  *slog.Logger
}

// New creates a new logger with JSON output to stdout.
func New() *Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return &Logger{
		handler: handler,
		logger:  slog.New(handler),
	}
}

// Info logs an info level message with key-value pairs.
func (l *Logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Error logs an error level message with key-value pairs.
func (l *Logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// Warn logs a warn level message with key-value pairs.
func (l *Logger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Debug logs a debug level message with key-value pairs.
func (l *Logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Fatal logs a fatal level message with key-value pairs and exits with code 1.
func (l *Logger) Fatal(msg string, args ...any) {
	l.logger.Error(msg, args...)
	os.Exit(1)
}

// With adds key-value pairs to the logger context.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		handler: l.handler.WithAttrs(parseArgs(args...)),
		logger:  l.logger.With(args...),
	}
}

// WithGroup creates a new logger with a grouped namespace.
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{
		handler: l.handler.WithGroup(name),
		logger:  l.logger.WithGroup(name),
	}
}

// parseArgs converts alternating key-value arguments to slog.Attr slice
func parseArgs(args ...any) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(args)/2)
	for i := 0; i < len(args)-1; i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		attrs = append(attrs, slog.Any(key, args[i+1]))
	}
	return attrs
}

// Deprecated methods for backward compatibility
// These are deprecated and will be removed in v2.0

// Infof is deprecated, use Info instead.
func (l *Logger) Infof(format string, args ...any) {
	l.logger.Info(format, slog.Any("args", args))
}

// Errorf is deprecated, use Error instead.
func (l *Logger) Errorf(format string, args ...any) {
	l.logger.Error(format, slog.Any("args", args))
}

// Fatalf is deprecated, use Fatal instead.
func (l *Logger) Fatalf(format string, args ...any) {
	l.logger.Error(format, slog.Any("args", args))
	os.Exit(1)
}
