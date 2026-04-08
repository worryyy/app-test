package responses

import "net/http"

const (
	CodeSuccess = http.StatusOK
	CodeFail    = http.StatusBadRequest

	HTTPStatusOK           = http.StatusOK
	HTTPStatusBadRequest   = http.StatusBadRequest
	HTTPStatusUnauthorized = http.StatusUnauthorized
	HTTPStatusForbidden    = http.StatusForbidden
	HTTPStatusNotFound     = http.StatusNotFound
	HTTPStatusInternalErr  = http.StatusInternalServerError
)
