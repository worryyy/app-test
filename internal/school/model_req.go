package school

type CourseColorReq struct {
	Colors map[string]string `json:"colors" binding:"required"`
}

type CurTermReq struct {
	TermID string `json:"termId" binding:"required"`
}
