package report

type ReportReviewReq struct {
	HandlerContent string `json:"handlerContent" binding:"required"`
}
