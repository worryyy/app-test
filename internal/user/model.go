package user

type User struct {
	ID          int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OpenID      string `gorm:"column:openId" json:"openId"`
	Power       int    `gorm:"column:power" json:"power"`
	AccountType string `gorm:"column:accountType" json:"accountType"`
	RootUserID  int64  `gorm:"column:rootUserId" json:"rootUserId"`
}

func (User) TableName() string {
	return "campus_user"
}

type Admin struct {
	ID       int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID   int64  `gorm:"column:userId;uniqueIndex" json:"userId"`
	Username string `gorm:"column:username;uniqueIndex" json:"username"`
	Password string `gorm:"column:password" json:"-"`
	Power    int    `gorm:"column:power;default:2" json:"power"`
}

func (Admin) TableName() string {
	return "admin"
}
