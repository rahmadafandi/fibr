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

package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPagination(t *testing.T) {
	t.Run("NewPagination", func(t *testing.T) {
		items := []string{"a", "b", "c"}
		pageSize := 10
		pageNumber := 1
		totalCount := 3

		pagination := NewPagination(items, pageSize, pageNumber, totalCount)

		assert.Equal(t, items, pagination.Data)
		assert.Equal(t, pageSize, pagination.PageSize)
		assert.Equal(t, len(items), pagination.Count)
		assert.Equal(t, totalCount, pagination.TotalCount)
		assert.Equal(t, 1, pagination.PageCount)
		assert.Equal(t, pageNumber, pagination.PageNumber)
		assert.Equal(t, 1, pagination.StartNumber)
	})
}