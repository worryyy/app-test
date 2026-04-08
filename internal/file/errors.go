package file

import "github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"

const errMsgInvalidParam = "参数错误"

var (
	ErrFileNotFound          = bizerr.NotFound("文件不存在")
	ErrFileLimited           = bizerr.Biz("文件大小超出限制")
	ErrUnsupportedFileFormat = bizerr.Param("图片格式只能是[image/png image/jpeg image/x-icon application/octet-stream]")
	ErrFileUpdateFailed      = bizerr.Biz("更新失败")
)
