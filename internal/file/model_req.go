package file

type FilePublicReq struct {
	MD5List []string `json:"md5List" binding:"required"`
}

type UploadResp struct {
	Path string `json:"path"`
}
