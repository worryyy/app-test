package responses

import "net/http"

var (
	Success     = New(CodeSuccess, "success", http.StatusOK)
	ParamErr    = New(CodeParamErr, "invalid parameters", http.StatusBadRequest)
	BizErr      = New(CodeBizErr, "business error", http.StatusBadRequest)
	NotFound    = New(CodeNotFound, "not found", http.StatusNotFound)
	InternalErr = New(CodeInternalErr, "internal error", http.StatusInternalServerError)
)
