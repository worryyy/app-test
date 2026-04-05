package user

type userIDURI struct {
	ID int64 `uri:"id" binding:"required,gt=0"`
}

type randomNicknameQuery struct {
	Type string `form:"type" binding:"required"`
}

type userProfileQuery struct {
	TargetUserID       string `form:"target_user_id"`
	LegacyTargetUserID string `form:"targetUserId"`
}

func (q userProfileQuery) ResolvedTargetUserID() string {
	return firstNonBlank(q.TargetUserID, q.LegacyTargetUserID)
}

type adminListUsersQuery struct {
	Page     int    `form:"page"`
	Size     int    `form:"size"`
	NickName string `form:"nickName"`
}

type preAuthQuery struct {
	UserID   int64  `form:"user_id" binding:"required,gt=0"`
	NickName string `form:"nick_name" binding:"required"`
	Pwd      string `form:"pwd" binding:"required"`
}

type courseKeyQuery struct {
	Key string `form:"key" binding:"required"`
}
