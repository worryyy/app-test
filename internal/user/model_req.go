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
	AvatarURL         string `json:"avatarUrl"`
	FullName          string `json:"fullName" binding:"required"`
	ShortName         string `json:"shortName" binding:"required"`
	Nature            string `json:"nature"`
	Introduction      string `json:"introduction"`
	ResponsiblePerson string `json:"responsiblePerson" binding:"required"`
	WechatAccount     string `json:"wechatAccount"`
	LoginAccount      string `json:"loginAccount" binding:"required"`
	LoginPassword     string `json:"loginPassword" binding:"required"`
}

type IdentitySwitchReq struct {
	AccountType string `json:"accountType" binding:"required"`
}

type UpdateAnonymousNicknameReq struct {
	Nickname string `json:"nickname" binding:"required"`
}

type UserCourseReq struct {
	SchoolID  string `json:"schoolId" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Week      int    `json:"week"`
	Term      string `json:"term" binding:"required"`
	StartDate string `json:"startDate" binding:"required"`
}

type ExamReq struct {
	SchoolID string `json:"schoolId" binding:"required"`
	Password string `json:"password" binding:"required"`
	XNXQID   string `json:"xnxqid" binding:"required"`
}

type ExamScoreReq struct {
	SchoolID string `json:"schoolId" binding:"required"`
	Password string `json:"password" binding:"required"`
	SS       string `json:"ss"`
}

type UserEditReq struct {
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Gender    string `json:"gender"`
	Signature string `json:"signature"`
}

type AddAdminReq struct {
	UserID   int64  `json:"userId" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Power    *int   `json:"power"`
}

type UserIDReq struct {
	UserID int64 `json:"userId" binding:"required"`
}

type AdminEditUserReq struct {
	Nickname   string `json:"nickname"`
	Avatar     string `json:"avatar"`
	Power      *int   `json:"power"`
	StuNum     string `json:"stuNum"`
	StuName    string `json:"stuName"`
	StuCla     string `json:"stuCla"`
	StuIsCheck *bool  `json:"stuIsCheck"`
}

type CertReviewReq struct {
	CertificationID string `json:"certificationId" binding:"required"`
	Action          string `json:"action" binding:"required"`
	RejectReason    string `json:"rejectReason"`
	Tag             string `json:"tag" binding:"required"`
}
