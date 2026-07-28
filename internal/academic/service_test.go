package academic

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type profileStub struct{ school string }

func (p profileStub) RootSchool(context.Context, int64) (string, error) { return p.school, nil }

func TestCourseDedupAndReviewUpsert(t *testing.T) {
	svc, db := academicTestService(t)
	course, err := svc.CreateCourse(context.Background(), 11, 10, CreateCourseReq{Name: "  Data   Structures ", Teacher: "Alice"})
	if err != nil {
		t.Fatalf("create course: %v", err)
	}
	if _, err := svc.CreateCourse(context.Background(), 11, 10, CreateCourseReq{Name: "Ｄａｔａ Structures", Teacher: "alice"}); !errors.Is(err, ErrCourseDuplicated) {
		t.Fatalf("duplicate error = %v", err)
	}
	req := ReviewReq{Semester: "2026-1", Content: "first", OverallRating: 5, DifficultyRating: 4, WorkloadRating: 3, GainRating: 5}
	first, err := svc.SaveReview(context.Background(), 11, 10, course.ID, req)
	if err != nil {
		t.Fatalf("save review: %v", err)
	}
	req.Content = "updated"
	req.OverallRating = 4
	second, err := svc.SaveReview(context.Background(), 11, 10, course.ID, req)
	if err != nil {
		t.Fatalf("update review: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("review id changed: %s -> %s", first.ID, second.ID)
	}
	var count int64
	if err := db.Model(&Review{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("review count=%d err=%v", count, err)
	}
	detail, err := svc.CourseDetail(context.Background(), course.ID)
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail.Stats.Count != 1 || detail.Stats.Overall != 4 {
		t.Fatalf("stats=%+v", detail.Stats)
	}
	if err := svc.DeleteReview(context.Background(), 10, course.ID); err != nil {
		t.Fatalf("delete review: %v", err)
	}
	req.Content = "revived"
	third, err := svc.SaveReview(context.Background(), 11, 10, course.ID, req)
	if err != nil {
		t.Fatalf("revive review: %v", err)
	}
	if third.ID != first.ID {
		t.Fatalf("revived review did not reuse row")
	}
}

func TestCourseMergeKeepsLatestConflictingReview(t *testing.T) {
	svc, db := academicTestService(t)
	source, _ := svc.CreateCourse(context.Background(), 11, 10, CreateCourseReq{Name: "Source"})
	target, _ := svc.CreateCourse(context.Background(), 11, 10, CreateCourseReq{Name: "Target"})
	now := svc.now()
	targetReview := Review{ID: 101, CourseID: mustID(t, target.ID), RootUserID: 10, Semester: "x", Content: "old", OverallRating: 1, DifficultyRating: 1, WorkloadRating: 1, GainRating: 1, Status: StatusNormal, CreatedAt: now, UpdatedAt: now}
	sourceReview := Review{ID: 102, CourseID: mustID(t, source.ID), RootUserID: 10, Semester: "x", Content: "new", OverallRating: 5, DifficultyRating: 5, WorkloadRating: 5, GainRating: 5, Status: StatusNormal, CreatedAt: now, UpdatedAt: now.Add(time.Minute)}
	if err := db.Create(&targetReview).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&sourceReview).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.MergeCourses(context.Background(), source.ID, target.ID); err != nil {
		t.Fatalf("merge: %v", err)
	}
	var got Review
	if err := db.Where("course_id = ? AND root_user_id = ?", targetReview.CourseID, 10).Take(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Content != "new" || got.OverallRating != 5 {
		t.Fatalf("merged review=%+v", got)
	}
}

func academicTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Course{}, &Review{}, &Material{}); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil)
	svc.SetProfileResolver(profileStub{school: "SZTU"})
	svc.now = func() time.Time { return time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC) }
	return svc, db
}
func mustID(t *testing.T, raw string) int64 {
	t.Helper()
	value, err := parseID(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
