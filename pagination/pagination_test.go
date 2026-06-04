// Copyright 2026 Rahmad Afandi. MIT License.

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

	t.Run("ZeroPageSize", func(t *testing.T) {
		items := []string{"a"}
		assert.NotPanics(t, func() {
			p := NewPagination(items, 0, 1, 3)
			assert.Equal(t, 0, p.PageCount)
			assert.Equal(t, 0, p.StartNumber)
		})
	})

	t.Run("PageNumberBelowOne", func(t *testing.T) {
		items := []string{"a", "b"}
		p := NewPagination(items, 10, 0, 2)
		assert.Equal(t, 1, p.PageNumber)
		assert.Equal(t, 1, p.StartNumber)
	})
}
