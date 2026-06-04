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
