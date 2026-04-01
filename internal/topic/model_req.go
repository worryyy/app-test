package topic

type CreateTopicReq struct {
	ThemeID     string      `json:"themeId" binding:"required"`
	Title       string      `json:"title" binding:"required,max=20"`
	Content     string      `json:"content" binding:"required,max=6000"`
	Imgs        []string    `json:"imgs" binding:"omitempty,max=9"`
	Ext         interface{} `json:"ext"`
	AccountType string      `json:"account_type"`
}

type UpdateTopicReq struct {
	Title    string   `json:"title" binding:"omitempty,max=20"`
	Content  string   `json:"content" binding:"omitempty,max=65535"`
	Imgs     []string `json:"imgs" binding:"omitempty,max=9"`
	Ext      *string  `json:"ext"`
	HasCheck *bool    `json:"hasCheck"`
}
