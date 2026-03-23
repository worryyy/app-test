package other

type VoteAcceptReq struct {
	OptionIDs []int64 `json:"optionIds" binding:"required"`
}

type VoteReq struct {
	OptionIDs []int64 `json:"optionIds" binding:"required"`
}

type WordsReq struct {
	Words []string `json:"words" binding:"required"`
}

type ReportReviewReq struct {
	Status int `json:"status" binding:"required"`
}

type MerchantThemeReq struct {
	ThemeID string `json:"themeId" binding:"required"`
}
