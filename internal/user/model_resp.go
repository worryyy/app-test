package user

import "time"

type IdentityVO struct {
	UserID      int64  `json:"user_id"`
	AccountType string `json:"account_type"`
	Nickname    string `json:"nickname"`
	Avatar      string `json:"avatar"`
	Tag         string `json:"tag"`
}

type LoginResp struct {
	Token           string      `json:"token"`
	RefreshToken    string      `json:"refresh_token"`
	User            *User       `json:"user,omitempty"`
	IsNew           bool        `json:"is_new"`
	CurrentIdentity *IdentityVO `json:"currentIdentity,omitempty"`
	RootUserID      int64       `json:"rootUserId,omitempty"`
}

type AdminLoginResp struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user,omitempty"`
}

type RefreshTokenResp struct {
	Token           string      `json:"token"`
	RefreshToken    string      `json:"refresh_token"`
	CurrentIdentity *IdentityVO `json:"currentIdentity,omitempty"`
}

type SwitchIdentityResp struct {
	Token           string      `json:"token"`
	RefreshToken    string      `json:"refreshToken"`
	CurrentIdentity *IdentityVO `json:"currentIdentity,omitempty"`
	RootUserID      int64       `json:"rootUserId,omitempty"`
}

type IdentityListResp struct {
	RootUserID   int64         `json:"root_user_id"`
	Identities   []*IdentityVO `json:"identities"`
	HasAnonymous bool          `json:"hasAnonymous"`
}

type UserProfileVO struct {
	Avatar    string `json:"avatar"`
	Nickname  string `json:"nickname"`
	Gender    string `json:"gender"`
	StuCla    string `json:"stuCla"`
	Signature string `json:"signature"`
}

type UserStatsVO struct {
	FollowerCount  int64 `json:"followerCount"`
	FollowingCount int64 `json:"followingCount"`
	LikeCount      int64 `json:"likeCount"`
}

type FollowVO struct {
	Avatar      string    `json:"avatar"`
	Nickname    string    `json:"nickname"`
	FollowerID  int64     `json:"follower_id"`
	FollowingID int64     `json:"following_id"`
	FollowAt    time.Time `json:"follow_at"`
	CoFollow    bool      `json:"co_follow"`
	BothFollow  bool      `json:"both_follow"`
}

type OfficialCertificationListItem struct {
	ID                string     `json:"id"`
	AvatarURL         string     `json:"avatarUrl"`
	FullName          string     `json:"fullName"`
	ShortName         string     `json:"shortName"`
	Nature            string     `json:"nature"`
	Introduction      string     `json:"introduction"`
	ResponsiblePerson string     `json:"responsiblePerson"`
	WechatAccount     string     `json:"wechatAccount"`
	LoginAccount      string     `json:"loginAccount"`
	Status            string     `json:"status"`
	RejectReason      string     `json:"rejectReason"`
	ReviewedBy        int64      `json:"reviewedBy"`
	ReviewedAt        *time.Time `json:"reviewedAt"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func buildIdentityVO(u *User) *IdentityVO {
	if u == nil {
		return nil
	}
	return &IdentityVO{
		UserID:      u.ID,
		AccountType: u.AccountType,
		Nickname:    u.Nickname,
		Avatar:      u.Avatar,
		Tag:         u.Tag,
	}
}
