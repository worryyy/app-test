package chat

type PageResult[T any] struct {
	Data    []T   `json:"data"`
	Current int   `json:"current"`
	Total   int64 `json:"total"`
	Size    int   `json:"size"`
}

func NewPageResult[T any](data []T, total int64, page, size int) *PageResult[T] {
	if data == nil {
		data = []T{}
	}
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}

	return &PageResult[T]{
		Data:    data,
		Current: page,
		Total:   total,
		Size:    size,
	}
}
