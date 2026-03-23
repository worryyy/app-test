package comment

type CreateCommentReq struct {
	Comment     string `json:"comment" binding:"required"`
	ParentCmtID string `json:"parentCmtId"`
	RootCmtID   string `json:"rootCmtId"`
}
