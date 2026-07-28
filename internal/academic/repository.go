package academic

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }
func (r *Repository) gormDB(ctx context.Context) (*gorm.DB, error) {
	if r.db == nil {
		return nil, errors.New("academic database not initialized")
	}
	return r.db.WithContext(ctx), nil
}

func (r *Repository) CreateCourse(ctx context.Context, course *Course) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	if err := db.Create(course).Error; err != nil {
		return fmt.Errorf("create course: %w", err)
	}
	return nil
}

func (r *Repository) FindCourseByIdentity(ctx context.Context, school, normalizedName, normalizedTeacher string) (*Course, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var course Course
	err = db.Where("school = ? AND normalized_name = ? AND normalized_teacher = ?", school, normalizedName, normalizedTeacher).Take(&course).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &course, nil
}

func (r *Repository) FindCourse(ctx context.Context, id int64) (*Course, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var course Course
	err = db.Where("id = ?", id).Take(&course).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &course, err
}

func (r *Repository) SearchCourses(ctx context.Context, school, keyword string, page, size int) ([]Course, int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, 0, err
	}
	base := db.Model(&Course{}).Where("school = ? AND status = ?", school, StatusNormal)
	if keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where("name LIKE ? OR teacher LIKE ?", like, like)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []Course
	if err := base.Order("updated_at DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	if list == nil {
		list = []Course{}
	}
	return list, total, nil
}

func (r *Repository) ListReviews(ctx context.Context, courseID int64, page, size int) ([]Review, int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, 0, err
	}
	base := db.Model(&Review{}).Where("course_id = ? AND status = ?", courseID, StatusNormal)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []Review
	if err := base.Order("updated_at DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	if list == nil {
		list = []Review{}
	}
	return list, total, nil
}

func (r *Repository) RatingStats(ctx context.Context, courseID int64) (RatingStats, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return RatingStats{}, err
	}
	var stats RatingStats
	err = db.Model(&Review{}).Select("COUNT(*) count, COALESCE(AVG(overall_rating),0) overall, COALESCE(AVG(difficulty_rating),0) difficulty, COALESCE(AVG(workload_rating),0) workload, COALESCE(AVG(gain_rating),0) gain").
		Where("course_id = ? AND status = ?", courseID, StatusNormal).Scan(&stats).Error
	return stats, err
}

func (r *Repository) SaveReview(ctx context.Context, review *Review) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var existing Review
		err := tx.Where("course_id = ? AND root_user_id = ?", review.CourseID, review.RootUserID).Take(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(review).Error
		}
		if err != nil {
			return err
		}
		review.ID, review.CreatedAt = existing.ID, existing.CreatedAt
		return tx.Model(&Review{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"semester": review.Semester, "content": review.Content, "overall_rating": review.OverallRating,
			"difficulty_rating": review.DifficultyRating, "workload_rating": review.WorkloadRating,
			"gain_rating": review.GainRating, "status": StatusNormal, "updated_at": review.UpdatedAt,
		}).Error
	})
}

func (r *Repository) DeleteReview(ctx context.Context, courseID, rootUserID int64, now time.Time) (bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return false, err
	}
	result := db.Model(&Review{}).Where("course_id = ? AND root_user_id = ? AND status <> ?", courseID, rootUserID, StatusDeleted).Updates(map[string]any{"status": StatusDeleted, "updated_at": now})
	return result.RowsAffected > 0, result.Error
}

func (r *Repository) CreateMaterial(ctx context.Context, material *Material) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	return db.Create(material).Error
}

func (r *Repository) ListMaterials(ctx context.Context, query string, args []any, page, size int) ([]Material, int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, 0, err
	}
	base := db.Model(&Material{}).Where(query, args...)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []Material
	if err := base.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	if list == nil {
		list = []Material{}
	}
	return list, total, nil
}

func (r *Repository) FindMaterial(ctx context.Context, id int64) (*Material, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var item Material
	err = db.Where("id = ?", id).Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, err
}

func (r *Repository) UpdateMaterialStatus(ctx context.Context, id int64, owner *int64, status string, now time.Time) (bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return false, err
	}
	query := db.Model(&Material{}).Where("id = ?", id)
	if owner != nil {
		query = query.Where("root_user_id = ?", *owner)
	}
	result := query.Updates(map[string]any{"status": status, "updated_at": now})
	return result.RowsAffected > 0, result.Error
}

func (r *Repository) UpdateCourseStatus(ctx context.Context, id int64, status string, now time.Time) (bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return false, err
	}
	result := db.Model(&Course{}).Where("id = ?", id).Updates(map[string]any{"status": status, "updated_at": now})
	return result.RowsAffected > 0, result.Error
}

func (r *Repository) UpdateReviewStatus(ctx context.Context, id int64, status string, now time.Time) (bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return false, err
	}
	result := db.Model(&Review{}).Where("id = ?", id).Updates(map[string]any{"status": status, "updated_at": now})
	return result.RowsAffected > 0, result.Error
}

func (r *Repository) FindReview(ctx context.Context, id int64) (*Review, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var item Review
	err = db.Where("id = ?", id).Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, err
}

func (r *Repository) MergeCourses(ctx context.Context, sourceID, targetID int64, now time.Time) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var source, target Course
		if err := tx.Where("id = ? AND status = ?", sourceID, StatusNormal).Take(&source).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ? AND status = ?", targetID, StatusNormal).Take(&target).Error; err != nil {
			return err
		}
		var reviews []Review
		if err := tx.Where("course_id = ?", sourceID).Find(&reviews).Error; err != nil {
			return err
		}
		for _, review := range reviews {
			var existing Review
			err := tx.Where("course_id = ? AND root_user_id = ?", targetID, review.RootUserID).Take(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Model(&Review{}).Where("id = ?", review.ID).Updates(map[string]any{"course_id": targetID, "updated_at": now}).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			if review.UpdatedAt.After(existing.UpdatedAt) {
				if err := tx.Model(&Review{}).Where("id = ?", existing.ID).Updates(map[string]any{"semester": review.Semester, "content": review.Content, "overall_rating": review.OverallRating, "difficulty_rating": review.DifficultyRating, "workload_rating": review.WorkloadRating, "gain_rating": review.GainRating, "status": review.Status, "updated_at": now}).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&Review{}).Where("id = ?", review.ID).Updates(map[string]any{"status": StatusMerged, "updated_at": now}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&Material{}).Where("course_id = ?", sourceID).Updates(map[string]any{"course_id": targetID, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&Course{}).Where("id = ?", sourceID).Updates(map[string]any{"status": StatusMerged, "merge_target_id": targetID, "updated_at": now}).Error
	})
}
