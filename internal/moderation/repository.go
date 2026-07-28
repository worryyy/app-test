package moderation

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
		return nil, errors.New("moderation database not initialized")
	}
	return r.db.WithContext(ctx), nil
}

func (r *Repository) CreateReport(ctx context.Context, report *Report, snapshot *ReportSnapshot) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(report).Error; err != nil {
			return fmt.Errorf("create report: %w", err)
		}
		if err := tx.Create(snapshot).Error; err != nil {
			return fmt.Errorf("create report snapshot: %w", err)
		}
		return nil
	})
}

func (r *Repository) ListReports(ctx context.Context, query any, args []any, page, size int) ([]Report, int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, 0, err
	}
	base := db.Model(&Report{})
	if query != nil {
		base = base.Where(query, args...)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count reports: %w", err)
	}
	var list []Report
	if err := base.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list reports: %w", err)
	}
	if list == nil {
		list = []Report{}
	}
	return list, total, nil
}

func (r *Repository) FindReport(ctx context.Context, id int64) (*Report, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var report Report
	err = db.Where("id = ?", id).Take(&report).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find report: %w", err)
	}
	return &report, nil
}

func (r *Repository) WithdrawReport(ctx context.Context, id, rootUserID int64, now time.Time) (bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return false, err
	}
	result := db.Model(&Report{}).Where("id = ? AND reporter_root_user_id = ? AND status = ?", id, rootUserID, ReportPending).
		Updates(map[string]any{"status": ReportWithdrawn, "withdrawn_at": now, "updated_at": now})
	return result.RowsAffected > 0, result.Error
}

func (r *Repository) ClaimReport(ctx context.Context, id, adminID int64, now time.Time) (bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return false, err
	}
	result := db.Model(&Report{}).Where("id = ? AND status = ?", id, ReportPending).
		Updates(map[string]any{"status": ReportReviewing, "assignee_admin_id": adminID, "updated_at": now})
	if result.Error != nil {
		return false, fmt.Errorf("claim report: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *Repository) CompleteReview(ctx context.Context, reportID, adminID int64, status string, punishment *Punishment, audit *AuditLog, now time.Time) (bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return false, err
	}
	updated := false
	err = db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&Report{}).Where("id = ? AND status = ? AND assignee_admin_id = ?", reportID, ReportReviewing, adminID).
			Updates(map[string]any{"status": status, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		updated = true
		if punishment != nil {
			if err := tx.Create(punishment).Error; err != nil {
				return fmt.Errorf("create punishment: %w", err)
			}
		}
		if err := tx.Create(audit).Error; err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}
		return nil
	})
	return updated, err
}

func (r *Repository) ListPunishments(ctx context.Context, rootUserID int64, page, size int) ([]Punishment, int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, 0, err
	}
	base := db.Model(&Punishment{}).Where("root_user_id = ?", rootUserID)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []Punishment
	if err := base.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	if list == nil {
		list = []Punishment{}
	}
	return list, total, nil
}

func (r *Repository) ActivePunishments(ctx context.Context, rootUserID int64, now time.Time) ([]Punishment, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var list []Punishment
	err = db.Where("root_user_id = ? AND status = ? AND starts_at <= ? AND (ends_at IS NULL OR ends_at > ?)", rootUserID, PunishmentActive, now, now).Find(&list).Error
	if list == nil {
		list = []Punishment{}
	}
	return list, err
}

func (r *Repository) FindPunishment(ctx context.Context, id int64) (*Punishment, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var item Punishment
	err = db.Where("id = ?", id).Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, err
}

func (r *Repository) CreateAppeal(ctx context.Context, appeal *Appeal) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	if err := db.Create(appeal).Error; err != nil {
		return fmt.Errorf("create appeal: %w", err)
	}
	return nil
}

func (r *Repository) ListAppeals(ctx context.Context, rootUserID *int64, page, size int) ([]Appeal, int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, 0, err
	}
	base := db.Model(&Appeal{})
	if rootUserID != nil {
		base = base.Where("root_user_id = ?", *rootUserID)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []Appeal
	if err := base.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	if list == nil {
		list = []Appeal{}
	}
	return list, total, nil
}

func (r *Repository) DecideAppeal(ctx context.Context, appealID, adminID int64, approved bool, resolution string, now time.Time, audit *AuditLog) (int64, bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return 0, false, err
	}
	var rootUserID int64
	updated := false
	err = db.Transaction(func(tx *gorm.DB) error {
		var appeal Appeal
		if err := tx.Where("id = ? AND status = ?", appealID, AppealPending).Take(&appeal).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		status := AppealRejected
		if approved {
			status = AppealApproved
		}
		if err := tx.Model(&Appeal{}).Where("id = ? AND status = ?", appealID, AppealPending).
			Updates(map[string]any{"status": status, "resolution": resolution, "reviewed_by": adminID, "updated_at": now}).Error; err != nil {
			return err
		}
		if approved {
			if err := tx.Model(&Punishment{}).Where("id = ? AND status = ?", appeal.PunishmentID, PunishmentActive).
				Updates(map[string]any{"status": PunishmentRevoked, "revoked_by": adminID, "revoked_at": now, "revoke_reason": resolution}).Error; err != nil {
				return err
			}
		}
		audit.AppealID = &appeal.ID
		audit.PunishmentID = &appeal.PunishmentID
		if err := tx.Create(audit).Error; err != nil {
			return err
		}
		rootUserID, updated = appeal.RootUserID, true
		return nil
	})
	return rootUserID, updated, err
}

func (r *Repository) RevokePunishment(ctx context.Context, id, adminID int64, reason string, now time.Time, audit *AuditLog) (int64, bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return 0, false, err
	}
	var rootUserID int64
	updated := false
	err = db.Transaction(func(tx *gorm.DB) error {
		var item Punishment
		if err := tx.Where("id = ? AND status = ?", id, PunishmentActive).Take(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if err := tx.Model(&Punishment{}).Where("id = ? AND status = ?", id, PunishmentActive).
			Updates(map[string]any{"status": PunishmentRevoked, "revoked_by": adminID, "revoked_at": now, "revoke_reason": reason}).Error; err != nil {
			return err
		}
		audit.PunishmentID = &item.ID
		if err := tx.Create(audit).Error; err != nil {
			return err
		}
		rootUserID, updated = item.RootUserID, true
		return nil
	})
	return rootUserID, updated, err
}
