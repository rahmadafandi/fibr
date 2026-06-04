// Copyright 2026 Rahmad Afandi. MIT License.

package pagination

// Pagination holds a single page of results together with the metadata needed
// to render page navigation controls.
type Pagination[T any] struct {
	Data        []T `json:"data"`
	PageSize    int `json:"page_size"`
	Count       int `json:"count"`
	TotalCount  int `json:"total_count"`
	PageCount   int `json:"page_count"`
	PageNumber  int `json:"page_number"`
	StartNumber int `json:"start_number"`
}

// NewPagination constructs a Pagination value from a slice of items and the
// associated paging parameters. pageNumber is clamped to 1 if less than 1.
func NewPagination[T any](items []T, pageSize int, pageNumber int, totalCount int) *Pagination[T] {
	if pageNumber < 1 {
		pageNumber = 1
	}

	pageCount := 0
	startNumber := 0
	if pageSize > 0 {
		pageCount = totalCount / pageSize
		if totalCount%pageSize != 0 {
			pageCount++
		}
		startNumber = (pageNumber-1)*pageSize + 1
	}

	return &Pagination[T]{
		Data:        items,
		PageSize:    pageSize,
		Count:       len(items),
		TotalCount:  totalCount,
		PageCount:   pageCount,
		PageNumber:  pageNumber,
		StartNumber: startNumber,
	}
}
