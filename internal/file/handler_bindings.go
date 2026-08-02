package file

type fileMD5URI struct {
	MD5 string `uri:"md5" binding:"required"`
}

type fileDownloadQuery struct {
	ShowOrigin int `form:"show_origin"`
}
