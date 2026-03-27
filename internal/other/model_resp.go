package other

import "time"

type NoticeVO struct {
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updated_at"`
}
