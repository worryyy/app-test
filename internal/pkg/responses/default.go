package responses

import "net/http"

var (
	Success = New(true,CodeSuccess, "success", http.StatusOK)

	ParamErr    = New(false,CodeParamErr, "invalid parameters", http.StatusBadRequest)
	BizErr      = New(false,CodeBizErr, "business error", http.StatusBadRequest)
	NotFound    = New(false,CodeNotFound, "not found", http.StatusBadRequest)
	InternalErr = New(false,CodeInternalErr, "internal error", http.StatusBadRequest)
)