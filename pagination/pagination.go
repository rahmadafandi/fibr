package pagination

type Pagination[T any] struct {
	Data        []T `json:"data"`
	PageSize    int `json:"page_size"`
	Count       int `json:"count"`
	TotalCount  int `json:"total_count"`
	PageCount   int `json:"page_count"`
	PageNumber  int `json:"page_number"`
	StartNumber int `json:"start_number"`
}

func NewPagination[T any](items []T, pageSize int, pageNumber int, totalCount int) *Pagination[T] {
	pageCount := totalCount / pageSize
	if totalCount%pageSize != 0 {
		pageCount++
	}
	startNumber := (pageNumber-1)*pageSize + 1
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
