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
	UserPrefix      = "campus:user:"
	UserExpPrefix   = "campus:userExp:"
	UserSignPrefix  = "campus:userSign:"
	ExpDetailKey    = "campus:expDetail:DETAIL_KEY"
	GlobalBlacklist = "campus:global_blacklist"
)

func User(id int64) string {
	return fmt.Sprintf("%s%d", UserPrefix, id)
}

func UserExp(id int64) string {
	return fmt.Sprintf("%s%d", UserExpPrefix, id)
}

func UserSign(yearMonth string) string {
	return UserSignPrefix + yearMonth
}

const ActiveDayPrefix = "campus:active:day:"

func ActiveDay(date string) string {
	return ActiveDayPrefix + date
}

const (
	AddMsgCache       = "campus:AMC:"
	DeleteMsgCache    = "campus:DMC:"
	UpdateMsgCache    = "campus:UMC:"
	TopicCreateCache  = "campus:TCC:"
	TopicInfoCache    = "campus:TIC:"
	DeleteTopicCache  = "campus:DTC:"
	UpdateTopicSearch = "campus:UTS:"
	AddTopicSearch    = "campus:ATS:"
	GetAllCourse      = "campus:GAC:"
	NotifyCache       = "campus:NOTIFY:"
)

const (
	SuggestRankPrefix = "rank:"
	SuggestCurKey     = "suggest:cur"
	SuggestPrevKey    = "suggest:prev"
	SuggestCountKey   = "suggest:cnt"
)

func SuggestRank(setName string) string {
	return SuggestRankPrefix + setName
}

const (
	SuggestTopicListKey = "campus:suggest_topic_list"
	UserCoursePrefix    = "campus:userCourse:"
	EventKey            = "campus:event:EVENT_KEY"
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
