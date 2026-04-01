package comment

type CreateCommentReq struct {
	Comment     string `json:"comment" binding:"required,max=524"`
	ParentCmtID string `json:"parentCmtId" binding:"required"`
}
