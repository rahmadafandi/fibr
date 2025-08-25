// Copyright 2025 Rahmad Afandi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

// Logger represents a logger.
type Logger struct {
	logger zerolog.Logger
}

// New creates a new logger.
func New(out io.Writer, level zerolog.Level) *Logger {
	return &Logger{
		logger: zerolog.New(out).With().Timestamp().Logger().Level(level),
	}
}

// Default creates a new logger with default settings.
func Default() *Logger {
	return New(os.Stdout, zerolog.InfoLevel)
}

// Debug logs a message at the debug level.
func (l *Logger) Debug(msg string, data ...interface{}) {
	l.logger.Debug().Fields(data).Msg(msg)
}

// Info logs a message at the info level.
func (l *Logger) Info(msg string, data ...interface{}) {
	l.logger.Info().Fields(data).Msg(msg)
}

// Warning logs a message at the warning level.
func (l *Logger) Warning(msg string, data ...interface{}) {
	l.logger.Warn().Fields(data).Msg(msg)
}

// Error logs a message at the error level.
func (l *Logger) Error(err error, msg string, data ...interface{}) {
	l.logger.Error().Err(err).Fields(data).Msg(msg)
}

// Fatal logs a message at the fatal level and exits.
func (l *Logger) Fatal(err error, msg string, data ...interface{}) {
	l.logger.Fatal().Err(err).Fields(data).Msg(msg)
}