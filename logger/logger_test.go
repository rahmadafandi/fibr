package logger

import (
	"bytes"
	"encoding/json"
	
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, zerolog.InfoLevel)

	t.Run("Info", func(t *testing.T) {
		buf.Reset()
		logger.Info("test message", "key", "value")

		var log map[string]interface{}
		json.Unmarshal(buf.Bytes(), &log)

		assert.Equal(t, "info", log["level"])
		assert.Equal(t, "test message", log["message"])
		assert.Contains(t, log, "key")
	})

	t.Run("Warning", func(t *testing.T) {
		buf.Reset()
		logger.Warning("test message", "key", "value")

		var log map[string]interface{}
		json.Unmarshal(buf.Bytes(), &log)

		assert.Equal(t, "warn", log["level"])
		assert.Equal(t, "test message", log["message"])
		assert.Contains(t, log, "key")
	})

	t.Run("Error", func(t *testing.T) {
		buf.Reset()
		logger.Error(nil, "test message", "key", "value")

		var log map[string]interface{}
		json.Unmarshal(buf.Bytes(), &log)

		assert.Equal(t, "error", log["level"])
		assert.Equal(t, "test message", log["message"])
		assert.Contains(t, log, "key")
	})

	t.Run("Debug", func(t *testing.T) {
		buf.Reset()
		logger.Debug("test message", "key", "value")

		assert.Equal(t, "", buf.String())
	})
}
