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
	"bytes"
	"encoding/json"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, InfoLevel)

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
