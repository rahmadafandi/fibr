// Copyright 2026 Rahmad Afandi. MIT License.

// Package logger is a structured logger built on zerolog.
package logger

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

type (
	// Level is re-exported from zerolog so callers need not import that package
	// directly.
	Level = zerolog.Level
)

// Log-level constants re-exported from zerolog.
const (
	InfoLevel  = zerolog.InfoLevel
	DebugLevel = zerolog.DebugLevel
	WarnLevel  = zerolog.WarnLevel
	ErrorLevel = zerolog.ErrorLevel
	FatalLevel = zerolog.FatalLevel
	PanicLevel = zerolog.PanicLevel
	TraceLevel = zerolog.TraceLevel
)

// Logger represents a logger.
type Logger struct {
	logger zerolog.Logger
}

// New creates a new logger.
func New(out io.Writer, level Level) *Logger {
	return &Logger{
		logger: zerolog.New(out).With().Timestamp().Logger().Level(level),
	}
}

// Default creates a new logger with default settings.
func Default() *Logger {
	return New(os.Stdout, zerolog.InfoLevel)
}

// Debug logs a message at the debug level.
func (l *Logger) Debug(msg string, data ...any) {
	l.logger.Debug().Fields(data).Msg(msg)
}

// Info logs a message at the info level.
func (l *Logger) Info(msg string, data ...any) {
	l.logger.Info().Fields(data).Msg(msg)
}

// Warning logs a message at the warning level.
func (l *Logger) Warning(msg string, data ...any) {
	l.logger.Warn().Fields(data).Msg(msg)
}

// Error logs a message at the error level.
func (l *Logger) Error(err error, msg string, data ...any) {
	l.logger.Error().Err(err).Fields(data).Msg(msg)
}

// Fatal logs a message at the fatal level and exits.
func (l *Logger) Fatal(err error, msg string, data ...any) {
	l.logger.Fatal().Err(err).Fields(data).Msg(msg)
}
