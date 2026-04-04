package user

type LoginReq struct {
	Code string `json:"code" binding:"required"`
}

type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type IdentitySwitchReq struct {
	AccountType       string `json:"accountType"`
	LegacyAccountType string `json:"account_type"`
}

func (r IdentitySwitchReq) ResolvedAccountType() string {
	return firstNonBlank(r.AccountType, r.LegacyAccountType)
}

type UpdateAnonymousNicknameReq struct {
	Nickname string `json:"nickname" binding:"required,max=50"`
}

type UserEditReq struct {
	Nickname  string `json:"nickname" binding:"omitempty,max=50"`
	Avatar    string `json:"avatar" binding:"omitempty,max=255"`
	Gender    string `json:"gender"`
	Signature string `json:"signature" binding:"omitempty,max=50"`
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
