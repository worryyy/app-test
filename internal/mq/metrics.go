package mq

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	postPublishTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "campus_post_publish_total",
		Help: "Total count of topic publish processing results",
	}, []string{"result"})

	commentPublishTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "campus_comment_publish_total",
		Help: "Total count of comment publish processing results",
	}, []string{"result"})
)

func incPostPublish(result string) {
	postPublishTotal.WithLabelValues(result).Inc()
}

func incCommentPublish(result string) {
	commentPublishTotal.WithLabelValues(result).Inc()
}
