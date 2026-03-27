package vote

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
