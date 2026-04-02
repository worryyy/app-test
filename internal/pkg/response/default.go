package response

import "net/http"

var (
	Success     = New(CodeSuccess, "", http.StatusOK)
	ParamErr    = New(CodeParamErr, "", http.StatusBadRequest)
	BizErr      = New(CodeBizErr, "", http.StatusBadRequest)
	NotFound    = New(CodeNotFound, "", http.StatusNotFound)
	InternalErr = New(CodeInternalErr, "", http.StatusInternalServerError)
)
