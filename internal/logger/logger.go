package logger

import (
	"log/slog"
	"os"
)

type Logger struct {
	*slog.Logger
}

// NewLogger creates a default structured JSON logger writing to stdout.
func NewLogger() *Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return &Logger{
		Logger: slog.New(handler),
	}
}

// NewWithLevel creates a logger with the specified minimum log level.
func NewWithLevel(level slog.Level) *Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithNode returns a logger with a persistent node ID field.
func (l *Logger) WithNode(nodeID string) *Logger {
	return &Logger{
		Logger: l.Logger.With(slog.String("node_id", nodeID)),
	}
}
