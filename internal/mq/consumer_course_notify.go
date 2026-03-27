package mq

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/encrypt"
)

type userCourseRow struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    int64     `gorm:"column:user_id"`
	Status    int       `gorm:"column:status"`
	Term      string    `gorm:"column:term"`
	Week      int       `gorm:"column:week"`
	Course    string    `gorm:"column:course;type:text"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (userCourseRow) TableName() string {
	return "campus_user_course"
}

func (c *Consumers) handleGetCourse(ctx context.Context, data json.RawMessage) error {
	var msg CourseMsg
	if err := decodeData(data, &msg); err != nil {
		return err
	}

	c.setCourseStatus(ctx, msg.UserID, msg.Term, msg.Week, 1, "")
	cookies, err := c.jwLogin(ctx, msg.StuNum, msg.StuPwd)
	if err != nil {
		c.setCourseStatus(ctx, msg.UserID, msg.Term, msg.Week, 3, "")
		return err
	}

	courseData, err := c.jwGetCourse(ctx, cookies, msg.Term, msg.Week)
	if err != nil {
		c.setCourseStatus(ctx, msg.UserID, msg.Term, msg.Week, 3, "")
		return err
	}

	raw, err := json.Marshal(courseData)
	if err != nil {
		return fmt.Errorf("marshal course payload: %w", err)
	}
	c.setCourseStatus(ctx, msg.UserID, msg.Term, msg.Week, 2, string(raw))
	return nil
}

func (c *Consumers) handleNotifyUser(ctx context.Context, data json.RawMessage) error {
	var msg NotifyMsg
	if err := decodeData(data, &msg); err != nil {
		return err
	}
	if msg.TargetUserID == "" {
		return nil
	}
	createdTime := msg.CreatedTime
	if createdTime.IsZero() {
		createdTime = time.Now()
	}

	id := primitive.NewObjectID()
	notification := bson.M{
		"_id":          id,
		"receiver_id":  msg.TargetUserID,
		"sender_id":    msg.SenderUserID,
		"type":         msg.Type,
		"content":      msg.Content,
		"topic_id":     msg.TopicID,
		"comment_id":   msg.CommentID,
		"created_time": createdTime,
		"is_read":      false,
	}
	if _, err := c.mongoDB.Collection("campus_notifications").InsertOne(ctx, notification); err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}

	if c.notifyPusher != nil {
		if err := c.notifyPusher(ctx, msg.TargetUserID, notification); err != nil {
			c.logger.Warn("push realtime notification failed", zap.Error(err), zap.String("targetUserID", msg.TargetUserID))
		}
	}
	return nil
}

func (c *Consumers) setCourseStatus(ctx context.Context, userID int64, term string, week int, status int, course string) {
	if c.db == nil {
		return
	}

	model := userCourseRow{
		UserID:    userID,
		Term:      term,
		Week:      week,
		Status:    status,
		Course:    course,
		UpdatedAt: time.Now(),
	}

	tx := c.db.WithContext(ctx)
	query := tx.Where("user_id = ? AND term = ? AND week = ?", userID, term, week)
	assign := userCourseRow{Status: status, Course: course, UpdatedAt: time.Now()}
	if err := query.Assign(assign).FirstOrCreate(&model).Error; err != nil {
		c.logger.Warn("update course status failed", zap.Error(err), zap.Int64("userID", userID), zap.String("term", term), zap.Int("week", week))
	}
}

func (c *Consumers) jwLogin(ctx context.Context, stuNum, stuPwd string) ([]*http.Cookie, error) {
	if c.cfg == nil || strings.TrimSpace(c.cfg.JW.BaseURL) == "" {
		return nil, fmt.Errorf("jw base url not configured")
	}

	key := []byte("PassB01I")[:8]
	encrypted, err := encrypt.DESECBEncrypt([]byte(stuPwd), key)
	if err != nil {
		return nil, fmt.Errorf("jw encrypt password: %w", err)
	}
	encodedPwd := base64.StdEncoding.EncodeToString(encrypted)

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create jw cookie jar: %w", err)
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 15 * time.Second,
	}

	form := url.Values{
		"username": {stuNum},
		"password": {encodedPwd},
	}
	loginURL := strings.TrimRight(c.cfg.JW.BaseURL, "/") + "/auth"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create jw login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jw login request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn("close jw login response body failed", zap.Error(closeErr))
		}
	}()
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("jw login status code: %d", resp.StatusCode)
	}

	baseURL, err := url.Parse(c.cfg.JW.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse jw base url: %w", err)
	}
	return jar.Cookies(baseURL), nil
}

func (c *Consumers) jwGetCourse(ctx context.Context, cookies []*http.Cookie, term string, week int) (interface{}, error) {
	if c.cfg == nil || strings.TrimSpace(c.cfg.JW.BaseURL) == "" {
		return nil, fmt.Errorf("jw base url not configured")
	}
	if c.db == nil {
		return map[string]interface{}{
			"term":  term,
			"week":  week,
			"items": []interface{}{},
		}, nil
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create jw cookie jar for course: %w", err)
	}
	baseURL, err := url.Parse(c.cfg.JW.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse jw base url: %w", err)
	}
	jar.SetCookies(baseURL, cookies)

	client := &http.Client{
		Jar:     jar,
		Timeout: 20 * time.Second,
	}
	values := url.Values{}
	values.Set("term", term)
	values.Set("week", fmt.Sprintf("%d", week))
	courseURL := strings.TrimRight(c.cfg.JW.BaseURL, "/") + "/course?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, courseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create jw course request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jw course request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn("close jw course response body failed", zap.Error(closeErr))
		}
	}()
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("jw course status code: %d", resp.StatusCode)
	}

	return map[string]interface{}{
		"term":  term,
		"week":  week,
		"items": []interface{}{},
	}, nil
}
