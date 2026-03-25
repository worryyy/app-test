package user

type IdentityVO struct {
	UserID      int64  `json:"userId"`
	AccountType string `json:"accountType"`
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
