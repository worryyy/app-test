package topic

type CreateTopicReq struct {
	ThemeID     string      `json:"themeId" binding:"required"`
	Title       string      `json:"title" binding:"required"`
	Content     string      `json:"content" binding:"required"`
	Imgs        []string    `json:"imgs"`
	Ext         interface{} `json:"ext"`
	AccountType string      `json:"account_type"`
}

type UpdateTopicReq struct {
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Imgs     []string `json:"imgs"`
	Ext      *string  `json:"ext"`
	HasCheck *bool    `json:"hasCheck"`
}
