package user

type LoginReq struct {
	Code string `json:"code" binding:"required"`
}

type RefreshTokenReq struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type AuthenticationReq struct {
	StuNum string `json:"stuNum" binding:"required"`
	StuPwd string `json:"stuPwd" binding:"required"`
}

type OfficialLoginReq struct {
	Username          string `json:"username" binding:"required"`
	Password          string `json:"password" binding:"required"`
	SecondaryPassword string `json:"secondaryPassword"`
}

type OfficialCertReq struct {
	Name   string `json:"name" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

type IdentityAnonymousReq struct {
	Nickname string `json:"nickname" binding:"required"`
}

type IdentitySwitchReq struct {
	TargetUserID int64 `json:"targetUserId" binding:"required"`
}

type FollowReq struct {
	TargetUserID int64 `json:"targetUserId" binding:"required"`
}

type UserCourseReq struct {
	Weeks []int  `json:"weeks" binding:"required"`
	Term  string `json:"term" binding:"required"`
}

type AddAdminReq struct {
	UserID   int64  `json:"userId" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserIDReq struct {
	UserID int64 `json:"userId" binding:"required"`
}

type UserIDsReq struct {
	UserIDs []int64 `json:"userIds" binding:"required"`
}

type CertReviewReq struct {
	CertID   string `json:"certId" binding:"required"`
	Approved bool   `json:"approved"`
}

type CourseFetchReq struct {
	Key string `json:"key" binding:"required"`
}
