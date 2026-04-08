package rediskey

import "fmt"

const (
	TokenPrefix        = "campus:token:"
	RefreshTokenPrefix = "campus:refresh_token:"
)

func Token(sha1 string) string {
	return TokenPrefix + sha1
}

func RefreshToken(sha1 string) string {
	return RefreshTokenPrefix + sha1
}

const (
	TokenStatusOK   = "OK"
	TokenStatusUsed = "USED"
)

const (
	UserPrefix     = "campus:user:"
	UserSignPrefix = "campus:userSign:"
)

func User(id int64) string {
	return fmt.Sprintf("%s%d", UserPrefix, id)
}

func UserSign(yearMonth string) string {
	return UserSignPrefix + yearMonth
}

const (
	AddMsgCache      = "campus:AMC:"
	DeleteMsgCache   = "campus:DMC:"
	UpdateMsgCache   = "campus:UMC:"
	TopicCreateCache = "campus:TCC:"
	TopicInfoCache   = "campus:TIC:"
	DeleteTopicCache = "campus:DTC:"
	GetAllCourse     = "campus:GAC:"
	NotifyCache      = "campus:NOTIFY:"
)

const (
	UserCoursePrefix = "campus:userCourse:"
)

func UserCourse(userID int64, term string, week int) string {
	return fmt.Sprintf("%s%d:%s:%d", UserCoursePrefix, userID, term, week)
}

const (
	AdminLoginLockPrefix = "admin:login:lock:"
	AdminLoginFailPrefix = "admin:login:fail:count:"
)

func AdminLoginLock(username string) string {
	return AdminLoginLockPrefix + username
}

func AdminLoginFail(username string) string {
	return AdminLoginFailPrefix + username
}

const MQUUIDKey = "uuid"
