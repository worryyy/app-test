package other

type VoteAcceptReq struct {
	OptionIDs []int64 `json:"optionIds" binding:"required"`
}

type VoteReq struct {
	OptionIDs []int64 `json:"optionIds" binding:"required"`
}

type VoteCreateReq struct {
	Info    VoteInfo     `json:"info" binding:"required"`
	Options []VoteOption `json:"options"`
}

type ReportReviewReq struct {
	HandlerContent string `json:"handlerContent" binding:"required"`
}

type MerchantThemeReq struct {
	ThemeID string `json:"themeId" binding:"required"`
}

type TaskNameReq struct {
	Name string `json:"name" binding:"required"`
}
