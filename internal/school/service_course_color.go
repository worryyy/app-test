package school

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) SetCourseColor(ctx context.Context, userID int64, colors []string) error {
	filtered := normalizeCourseColors(colors)
	if len(filtered) == 0 {
		return nil
	}

	term, err := s.currentTermEntity(ctx)
	if err != nil {
		return err
	}
	course, err := s.currentTermCourse(ctx, userID, term)
	if err != nil {
		return err
	}
	if course == nil {
		return result.NewBizError(result.CodeFail, "失败")
	}

	var week weekCourseVO
	if err := json.Unmarshal([]byte(course.Course), &week); err != nil {
		return fmt.Errorf("unmarshal current week course: %w", err)
	}
	names := sortedCourseNames(week.CourseList)
	if len(names) == 0 {
		return nil
	}

	rows := make([]CourseColor, 0, len(filtered))
	for i, name := range names {
		if i >= len(filtered) {
			break
		}
		rows = append(rows, CourseColor{
			UserID:     userID,
			CourseName: name,
			Color:      filtered[i],
		})
	}
	if len(rows) == 0 {
		return nil
	}

	if err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "course_name"}},
			DoUpdates: clause.AssignmentColumns([]string{"color"}),
		}).
		Create(&rows).Error; err != nil {
		return fmt.Errorf("upsert course colors: %w", err)
	}
	return nil
}

func (s *Service) GetCourseColor(ctx context.Context, userID int64) (map[string]string, error) {
	var rows []CourseColor
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query course colors: %w", err)
	}
	colors := make(map[string]string, len(rows))
	for _, row := range rows {
		colors[row.CourseName] = row.Color
	}
	return colors, nil
}

func (s *Service) currentTermEntity(ctx context.Context) (*Term, error) {
	current, err := s.currentTermDoc(ctx)
	if err != nil {
		return nil, fmt.Errorf("find current term doc: %w", err)
	}

	var term Term
	if err := s.mongoDB.Collection("campus_term").FindOne(ctx, bson.M{"term": current.Term}).Decode(&term); err != nil {
		return nil, fmt.Errorf("find current term entity: %w", err)
	}
	return &term, nil
}

func (s *Service) currentTermCourse(ctx context.Context, userID int64, term *Term) (*UserCourse, error) {
	curWeek := currentWeek(term.StartDate)
	course, err := s.userCourseByWeek(ctx, userID, term.Term, curWeek)
	if err != nil {
		return nil, err
	}
	if course != nil {
		return course, nil
	}
	if curWeek <= 3 {
		return nil, nil
	}
	return s.userCourseByWeek(ctx, userID, term.Term, curWeek-3)
}

func (s *Service) userCourseByWeek(ctx context.Context, userID int64, term string, week int) (*UserCourse, error) {
	var row UserCourse
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND term = ? AND week = ?", userID, term, week).
		First(&row).Error
	if err == nil {
		return &row, nil
	}
	if strings.Contains(err.Error(), "doesn't exist") {
		return nil, result.NewBizError(result.CodeFail, "失败")
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return nil, fmt.Errorf("query user course by week: %w", err)
}

func normalizeCourseColors(colors []string) []string {
	out := make([]string, 0, len(colors))
	for _, color := range colors {
		color = strings.TrimSpace(color)
		if color == "" {
			continue
		}
		out = append(out, color)
	}
	return out
}

func sortedCourseNames(list []innerCourse) []string {
	if len(list) == 0 {
		return []string{}
	}
	type stat struct {
		name  string
		count int
		first int
	}
	stats := make(map[string]*stat)
	for i, item := range list {
		name := formatCourseName(item.Name)
		if name == "" {
			continue
		}
		if existing, ok := stats[name]; ok {
			existing.count++
			continue
		}
		stats[name] = &stat{name: name, count: 1, first: i}
	}
	ordered := make([]stat, 0, len(stats))
	for _, item := range stats {
		ordered = append(ordered, *item)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].count == ordered[j].count {
			return ordered[i].first < ordered[j].first
		}
		return ordered[i].count > ordered[j].count
	})
	out := make([]string, 0, len(ordered))
	for _, item := range ordered {
		out = append(out, item.name)
	}
	return out
}

func formatCourseName(name string) string {
	name = strings.TrimSpace(name)
	runes := []rune(name)
	if len(runes) <= 15 {
		return name
	}
	return string(runes[:15])
}

func currentWeek(startDate string) int {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return 1
	}
	monday := start
	for monday.Weekday() != time.Monday {
		monday = monday.AddDate(0, 0, -1)
	}
	days := int(time.Since(monday).Hours() / 24)
	if days < 0 {
		days = -days
	}
	return days / 7
}
