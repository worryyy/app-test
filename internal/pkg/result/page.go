package result

func EnsureSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

type PageResult[T any] struct {
	Records []T   `json:"records"`
	Total   int64 `json:"total"`
	Current int   `json:"current"`
	Size    int   `json:"size"`
	Pages   int   `json:"pages"`
}

func NewPage[T any](records []T, total int64, current, size int) *PageResult[T] {
	if size <= 0 {
		size = 1
	}

	pages := int(total) / size
	if int(total)%size > 0 {
		pages++
	}

	return &PageResult[T]{
		Records: EnsureSlice(records),
		Total:   total,
		Current: current,
		Size:    size,
		Pages:   pages,
	}
}

type CusPage[T any] struct {
	Data    []T   `json:"data"`
	Current int   `json:"current"`
	Total   int64 `json:"total"`
	Size    int   `json:"size"`
}

func NewCusPage[T any](data []T, total int64, current, size int) *CusPage[T] {
	return &CusPage[T]{
		Data:    EnsureSlice(data),
		Current: current,
		Total:   total,
		Size:    size,
	}
}
