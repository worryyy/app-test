package school

type CourseColorReq struct {
	Colors []string `json:"colors" binding:"required"`
}

type CurTermReq struct {
	TermID string `json:"termId" binding:"required"`
}
