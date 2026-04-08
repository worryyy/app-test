package school

type CurTermReq struct {
	TermID string `json:"termId" binding:"required"`
}

type AuthenticationReq struct {
	SchoolID string `json:"schoolId" binding:"required"`
	Password string `json:"password" binding:"required"`
	School   string `json:"school" binding:"required"`
}

type CheckLoginReq struct {
	SchoolID string `json:"schoolId" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserCourseReq struct {
	SchoolID  string `json:"schoolId" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Week      int    `json:"week" binding:"min=0,max=30"`
	Term      string `json:"term" binding:"required,checkform=^\\d{4}-\\d{4}-\\d{1}$"`
	StartDate string `json:"startDate" binding:"required,checkform=^\\d{4}-\\d{2}-\\d{2}$"`
}

type ExamReq struct {
	SchoolID string `json:"schoolId" binding:"required"`
	Password string `json:"password" binding:"required"`
	XNXQID   string `json:"xnxqid" binding:"required,checkform=^\\d{4}-\\d{4}-\\d{1}$"`
}

type ExamScoreReq struct {
	SchoolID string `json:"schoolId" binding:"required"`
	Password string `json:"password" binding:"required"`
	SS       string `json:"ss" binding:"omitempty,checkform=^\\d{4}-\\d{4}-\\d{1}$"`
}
