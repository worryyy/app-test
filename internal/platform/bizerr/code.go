package bizerr

import "net/http"

const (
	CodeSuccess      = http.StatusOK
	CodeParamErr     = http.StatusBadRequest
	CodeBizErr       = http.StatusBadRequest
	CodeUnauthorized = http.StatusUnauthorized
	CodeForbidden    = http.StatusForbidden
	CodeNotFound     = http.StatusNotFound
	CodeInternalErr  = http.StatusInternalServerError
)
