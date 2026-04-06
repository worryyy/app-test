package responses

var (
	Success      = New(true, HTTPStatusOK, "success")
	ParamErr     = New(false, HTTPStatusBadRequest, "invalid parameters")
	BizErr       = New(false, HTTPStatusBadRequest, "business error")
	Unauthorized = New(false, HTTPStatusUnauthorized, "unauthorized")
	Forbidden    = New(false, HTTPStatusForbidden, "forbidden")
	NotFound     = New(false, HTTPStatusNotFound, "not found")
	InternalErr  = New(false, HTTPStatusInternalErr, "internal error")
)
