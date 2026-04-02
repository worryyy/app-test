package responses

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
)

func FromError(err error) Response {
	if err == nil {
		return Success
	}

	var be *bizerr.Error
	if errors.As(err, &be) {
		switch be.Code {
		case bizerr.CodeParamErr:
			return ParamErr.withMessage(be.Message)
		case bizerr.CodeBizErr:
			return BizErr.withMessage(be.Message)
		case bizerr.CodeNotFound:
			return NotFound.withMessage(be.Message)
		case bizerr.CodeInternalErr:
			return InternalErr.withMessage(be.Message)
		default:
			return InternalErr.withMessage(be.Message)
		}
	}

	return InternalErr
}

func Fail(ctx *gin.Context, err error) {
	FromError(err).Resp(ctx)
}

func FailMessage(ctx *gin.Context, err error, message string) {
	resp := FromError(err)
	resp.RespMessage(ctx, message)
}