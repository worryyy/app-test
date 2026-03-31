package notice

import "time"

type NoticeItem struct {
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updatedAt"`
}
