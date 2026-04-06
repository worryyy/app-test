package mq

const (
	Exchange    = "campus.exchange"
	DieExchange = "campus.die_exchange"
)

const (
	QueueCommentUpdate = "campus.comment_update"
	QueueTopicUpdate   = "campus.topic_update"
	QueueTopicDelete   = "campus.topic_delete"
	QueueCommentDelete = "campus.comment_delete"
	QueueCommentAdd    = "campus.comment_add"
	QueueTopicCheck    = "campus.topic_check"
	QueueGetCourse     = "campus.get_course"
	QueueNotifyUser    = "campus.notify_user"
	QueueDie           = "campus.die"
)

const (
	KeyUpdateCommentUser = "update_cmt_user"
	KeyUpdateTopicUser   = "update_topic_user"
	KeyDeleteTopic       = "delete_topic"
	KeyDeleteComment     = "delete_comment"
	KeyAddComment        = "add_comment"
	KeyTopicCheck        = "topic_check"
	KeyGetCourse         = "get_course"
	KeyNotifyUser        = "notify_user"
	KeyDie               = "die"
)

const (
	MsgPrev = 0
	MsgIng  = 1
	MsgPost = 2
)
