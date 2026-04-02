package bizerr

func Param(message string) *Error {
	return New(CodeParamErr, message)
}

func ParamWrap(message string, cause error) *Error {
	return Wrap(CodeParamErr, message, cause)
}

func Biz(message string) *Error {
	return New(CodeBizErr, message)
}

func BizWrap(message string, cause error) *Error {
	return Wrap(CodeBizErr, message, cause)
}

func NotFound(message string) *Error {
	return New(CodeNotFound, message)
}

func NotFoundWrap(message string, cause error) *Error {
	return Wrap(CodeNotFound, message, cause)
}

func Internal(message string) *Error {
	return New(CodeInternalErr, message)
}

func InternalWrap(message string, cause error) *Error {
	return Wrap(CodeInternalErr, message, cause)
}