package user


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
