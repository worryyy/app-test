package user

import "time"

type Identity struct {
	UserID      int64  `json:"userId"`
	AccountType string `json:"accountType"`
	Nickname    string `json:"nickname"`
	Avatar      string `json:"avatar"`
	Tag         string `json:"tag"`
}

type LoginResp struct {
	Token           string    `json:"token"`
	RefreshToken    string    `json:"refresh_token"`
	User            *User     `json:"user"`
	IsNew           bool      `json:"is_new"`
	CurrentIdentity *Identity `json:"currentIdentity"`
	RootUserID      int64     `json:"rootUserId"`
}

type AdminLoginResp struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}

type RefreshTokenResp struct {
	Token           string    `json:"token"`
	RefreshToken    string    `json:"refresh_token"`
	CurrentIdentity *Identity `json:"currentIdentity"`
}

type SwitchIdentityResp struct {
	Token           string    `json:"token"`
	RefreshToken    string    `json:"refreshToken"`
	CurrentIdentity *Identity `json:"currentIdentity"`
	RootUserID      int64     `json:"rootUserId"`
}

type IdentityListResp struct {
	RootUserID   int64       `json:"rootUserId"`
	Identities   []*Identity `json:"identities"`
	HasAnonymous bool        `json:"hasAnonymous"`
}

type UserProfile struct {
	Avatar    string `json:"avatar"`
	Nickname  string `json:"nickname"`
	Gender    string `json:"gender"`
	StuCla    string `json:"stuCla"`
	Signature string `json:"signature"`
}

type UserStats struct {
	FollowerCount  int64 `json:"followerCount"`
	FollowingCount int64 `json:"followingCount"`
	LikeCount      int64 `json:"likeCount"`
}

type FollowItem struct {
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
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

func buildIdentity(u *User) *Identity {
	if u == nil {
		return nil
	}
	return &Identity{
		UserID:      u.ID,
		AccountType: u.AccountType,
		Nickname:    u.Nickname,
		Avatar:      u.Avatar,
		Tag:         u.Tag,
	}
}
