package models

type PaginationResult[T any] struct {
	Data        []T   `json:"data"`
	Total       int64 `json:"total"`
	CurrentPage int32 `json:"current_page"`
	PageSize    int32 `json:"page_size"`
}
