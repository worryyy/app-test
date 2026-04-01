package user

type LoginReq struct {
	Code string `json:"code" binding:"required"`
}

type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
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

type OfficialLoginReq struct {
	LoginAccount  string `json:"loginAccount" binding:"required"`
	LoginPassword string `json:"loginPassword" binding:"required"`
}

type OfficialCertReq struct {
	AvatarURL         string `json:"avatarUrl" binding:"omitempty,max=255"`
	FullName          string `json:"fullName" binding:"required,max=100"`
	ShortName         string `json:"shortName" binding:"required,max=50"`
	Nature            string `json:"nature" binding:"omitempty,max=50"`
	Introduction      string `json:"introduction" binding:"omitempty,max=500"`
	ResponsiblePerson string `json:"responsiblePerson" binding:"required,max=50"`
	WechatAccount     string `json:"wechatAccount" binding:"omitempty,max=50"`
	LoginAccount      string `json:"loginAccount" binding:"required,max=50"`
	LoginPassword     string `json:"loginPassword" binding:"required,max=50"`
}

type IdentitySwitchReq struct {
	AccountType string `json:"account_type" binding:"required"`
}

type UpdateAnonymousNicknameReq struct {
	Nickname string `json:"nickname" binding:"required,max=50"`
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

type UserEditReq struct {
	Nickname  string `json:"nickname" binding:"omitempty,max=50"`
	Avatar    string `json:"avatar" binding:"omitempty,max=255"`
	Gender    string `json:"gender"`
	Signature string `json:"signature" binding:"omitempty,max=50"`
}

type AddAdminReq struct {
	UserID   int64  `json:"user_id" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Power    *int   `json:"power"`
}

type UserIDReq struct {
	UserID int64 `json:"user_id" binding:"required"`
}

type AdminEditUserReq struct {
	Nickname   string `json:"nickname"`
	Avatar     string `json:"avatar"`
	Power      *int   `json:"power"`
	StuNum     string `json:"stu_num"`
	StuName    string `json:"stuName"`
	StuCla     string `json:"stuCla"`
	StuIsCheck *bool  `json:"stu_is_check"`
}

type CertReviewReq struct {
	CertificationID string `json:"certificationId" binding:"required"`
	Action          string `json:"action" binding:"required"`
	RejectReason    string `json:"rejectReason"`
	Tag             string `json:"tag" binding:"required"`
}
