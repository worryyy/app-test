package user

type PageResult[T any] struct {
	Records          []T   `json:"records"`
	Total            int64 `json:"total"`
	Current          int   `json:"current"`
	Size             int   `json:"size"`
	Pages            int   `json:"pages"`
	Orders           []any `json:"orders"`
	OptimizeCountSql bool  `json:"optimizeCountSql"`
	SearchCount      bool  `json:"searchCount"`
	CountID          *int  `json:"countId"`
	MaxLimit         *int  `json:"maxLimit"`
}

func NewPageResult[T any](records []T, total int64, page, size int) *PageResult[T] {
	if records == nil {
		records = []T{}
	}

	pages := 0
	if size > 0 {
		pages = int((total + int64(size) - 1) / int64(size))
	}

	return &PageResult[T]{
		Records:          records,
		Total:            total,
		Current:          page,
		Size:             size,
		Pages:            pages,
		Orders:           []any{},
		OptimizeCountSql: true,
		SearchCount:      true,
	}
}
