package reservation

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

var (
	errSlotUnavailable = errors.New("slot unavailable")
	errBookingConflict = errors.New("booking conflict")
	errDailyLimit      = errors.New("daily limit reached")
	errCancelDeadline  = errors.New("cancel deadline passed")
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }
func (r *Repository) gormDB(ctx context.Context) (*gorm.DB, error) {
	if r.db == nil {
		return nil, errors.New("reservation database not initialized")
	}
	return r.db.WithContext(ctx), nil
}

func (r *Repository) ListVenues(ctx context.Context) ([]Venue, error) {
	db, e := r.gormDB(ctx)
	if e != nil {
		return nil, e
	}
	var list []Venue
	e = db.Where("status = ?", StatusActive).Order("name").Find(&list).Error
	if list == nil {
		list = []Venue{}
	}
	return list, e
}
func (r *Repository) FindVenue(ctx context.Context, id int64) (*Venue, error) {
	db, e := r.gormDB(ctx)
	if e != nil {
		return nil, e
	}
	var item Venue
	e = db.Where("id = ?", id).Take(&item).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, e
}
func (r *Repository) FindResource(ctx context.Context, id int64) (*Resource, error) {
	db, e := r.gormDB(ctx)
	if e != nil {
		return nil, e
	}
	var item Resource
	e = db.Where("id = ?", id).Take(&item).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, e
}
func (r *Repository) ListResources(ctx context.Context, venueID int64) ([]Resource, error) {
	db, e := r.gormDB(ctx)
	if e != nil {
		return nil, e
	}
	var list []Resource
	e = db.Where("venue_id = ? AND status = ?", venueID, StatusActive).Order("name").Find(&list).Error
	if list == nil {
		list = []Resource{}
	}
	return list, e
}
func (r *Repository) ListRules(ctx context.Context, resourceID int64, weekday int) ([]WeeklyRule, error) {
	db, e := r.gormDB(ctx)
	if e != nil {
		return nil, e
	}
	var list []WeeklyRule
	e = db.Where("resource_id = ? AND weekday = ? AND status = ?", resourceID, weekday, StatusActive).Order("start_minute").Find(&list).Error
	return list, e
}
func (r *Repository) EnsureSlots(ctx context.Context, slots []Slot) error {
	db, e := r.gormDB(ctx)
	if e != nil {
		return e
	}
	if len(slots) == 0 {
		return nil
	}
	return db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "resource_id"}, {Name: "start_at"}, {Name: "end_at"}}, DoNothing: true}).Create(&slots).Error
}
func (r *Repository) ListSlots(ctx context.Context, resourceID int64, start, end time.Time) ([]Slot, error) {
	db, e := r.gormDB(ctx)
	if e != nil {
		return nil, e
	}
	var list []Slot
	e = db.Where("resource_id = ? AND start_at >= ? AND start_at < ?", resourceID, start, end).Order("start_at").Find(&list).Error
	if list == nil {
		list = []Slot{}
	}
	return list, e
}
func (r *Repository) HasClosure(ctx context.Context, venueID, resourceID int64, start, end time.Time) (bool, error) {
	db, e := r.gormDB(ctx)
	if e != nil {
		return false, e
	}
	var count int64
	e = db.Model(&Closure{}).Where("(resource_id = ? OR (resource_id IS NULL AND venue_id = ?)) AND start_at < ? AND end_at > ?", resourceID, venueID, end, start).Count(&count).Error
	return count > 0, e
}

func (r *Repository) CreateVenue(ctx context.Context, item *Venue) error {
	db, e := r.gormDB(ctx)
	if e != nil {
		return e
	}
	return db.Create(item).Error
}
func (r *Repository) CreateResource(ctx context.Context, item *Resource) error {
	db, e := r.gormDB(ctx)
	if e != nil {
		return e
	}
	return db.Create(item).Error
}
func (r *Repository) CreateRule(ctx context.Context, item *WeeklyRule) error {
	db, e := r.gormDB(ctx)
	if e != nil {
		return e
	}
	return db.Create(item).Error
}

func (r *Repository) CreateBooking(ctx context.Context, booking *Booking, now time.Time) (*Slot, *Venue, error) {
	db, e := r.gormDB(ctx)
	if e != nil {
		return nil, nil, e
	}
	var locked Slot
	var venue Venue
	e = db.Transaction(func(tx *gorm.DB) error {
		var candidate Slot
		if e := tx.Where("id = ?", booking.SlotID).Take(&candidate).Error; e != nil {
			return errSlotUnavailable
		}
		date := time.Date(candidate.StartAt.Year(), candidate.StartAt.Month(), candidate.StartAt.Day(), 0, 0, 0, 0, candidate.StartAt.Location())
		dayLock := UserDayLock{ID: booking.ID, RootUserID: booking.RootUserID, ReservationDate: date}
		if e := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "root_user_id"}, {Name: "reservation_date"}}, DoNothing: true}).Create(&dayLock).Error; e != nil {
			return e
		}
		var lockedDay UserDayLock
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("root_user_id = ? AND reservation_date = ?", booking.RootUserID, date).Take(&lockedDay).Error; e != nil {
			return e
		}
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", booking.SlotID).Take(&locked).Error; e != nil {
			return errSlotUnavailable
		}
		if locked.Status != SlotOpen || locked.ReservedCount >= locked.Capacity || !locked.StartAt.After(now) {
			return errSlotUnavailable
		}
		var resource Resource
		if e := tx.Where("id = ? AND status = ?", locked.ResourceID, StatusActive).Take(&resource).Error; e != nil {
			return errSlotUnavailable
		}
		if e := tx.Where("id = ? AND status = ?", resource.VenueID, StatusActive).Take(&venue).Error; e != nil {
			return errSlotUnavailable
		}
		latest := dayStart(now.In(locked.StartAt.Location())).AddDate(0, 0, venue.AdvanceDays+1)
		if !locked.StartAt.Before(latest) {
			return errSlotUnavailable
		}
		valid := []string{BookingReserved, BookingCheckedIn}
		var daily int64
		if e := tx.Model(&Booking{}).Joins("JOIN reservation_slots s ON s.id = reservations.slot_id").Where("reservations.root_user_id = ? AND reservations.status IN ? AND s.start_at >= ? AND s.start_at < ?", booking.RootUserID, valid, date, date.AddDate(0, 0, 1)).Count(&daily).Error; e != nil {
			return e
		}
		if daily >= int64(venue.DailyLimit) {
			return errDailyLimit
		}
		var overlap int64
		if e := tx.Model(&Booking{}).Joins("JOIN reservation_slots s ON s.id = reservations.slot_id").Where("reservations.root_user_id = ? AND reservations.status IN ? AND s.start_at < ? AND s.end_at > ?", booking.RootUserID, valid, locked.EndAt, locked.StartAt).Count(&overlap).Error; e != nil {
			return e
		}
		if overlap > 0 {
			return errBookingConflict
		}
		result := tx.Model(&Slot{}).Where("id = ? AND reserved_count < capacity", locked.ID).Update("reserved_count", gorm.Expr("reserved_count + 1"))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errSlotUnavailable
		}
		return tx.Create(booking).Error
	})
	return &locked, &venue, e
}

func (r *Repository) ListBookings(ctx context.Context, rootUserID int64, page, size int) ([]Booking, int64, error) {
	db, e := r.gormDB(ctx)
	if e != nil {
		return nil, 0, e
	}
	base := db.Model(&Booking{}).Where("root_user_id = ?", rootUserID)
	var total int64
	if e = base.Count(&total).Error; e != nil {
		return nil, 0, e
	}
	var list []Booking
	e = base.Preload("Slot").Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error
	if list == nil {
		list = []Booking{}
	}
	return list, total, e
}
func (r *Repository) CancelBooking(ctx context.Context, id, rootUserID int64, now time.Time) (bool, error) {
	db, e := r.gormDB(ctx)
	if e != nil {
		return false, e
	}
	updated := false
	e = db.Transaction(func(tx *gorm.DB) error {
		var item Booking
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND root_user_id = ?", id, rootUserID).Take(&item).Error; e != nil {
			return nil
		}
		if item.Status != BookingReserved {
			return nil
		}
		var slot Slot
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", item.SlotID).Take(&slot).Error; e != nil {
			return e
		}
		var resource Resource
		if e := tx.Where("id = ?", slot.ResourceID).Take(&resource).Error; e != nil {
			return e
		}
		var venue Venue
		if e := tx.Where("id = ?", resource.VenueID).Take(&venue).Error; e != nil {
			return e
		}
		if !now.Before(slot.StartAt.Add(-time.Duration(venue.CancelBeforeMinutes) * time.Minute)) {
			return errCancelDeadline
		}
		bookingResult := tx.Model(&Booking{}).Where("id = ? AND status = ?", id, BookingReserved).Updates(map[string]any{"status": BookingCancelled, "cancelled_at": now, "updated_at": now})
		if bookingResult.Error != nil {
			return bookingResult.Error
		}
		if bookingResult.RowsAffected != 1 {
			return nil
		}
		slotResult := tx.Model(&Slot{}).Where("id = ? AND reserved_count > 0", slot.ID).Update("reserved_count", gorm.Expr("reserved_count - 1"))
		if slotResult.Error != nil {
			return slotResult.Error
		}
		if slotResult.RowsAffected != 1 {
			return errSlotUnavailable
		}
		updated = true
		return nil
	})
	return updated, e
}

func (r *Repository) Checkin(ctx context.Context, code string, now time.Time) (bool, error) {
	db, e := r.gormDB(ctx)
	if e != nil {
		return false, e
	}
	result := db.Model(&Booking{}).Where("checkin_code = ? AND status = ?", code, BookingReserved).Updates(map[string]any{"status": BookingCheckedIn, "checked_at": now, "updated_at": now})
	return result.RowsAffected > 0, result.Error
}
func (r *Repository) MarkNoShows(ctx context.Context, now time.Time) ([]Booking, error) {
	db, e := r.gormDB(ctx)
	if e != nil {
		return nil, e
	}
	items := []Booking{}
	e = db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Model(&Booking{}).Select("reservations.*").Joins("JOIN reservation_slots s ON s.id = reservations.slot_id").Where("reservations.status = ? AND s.end_at <= ?", BookingReserved, now).Find(&items).Error; e != nil {
			return e
		}
		if len(items) == 0 {
			return nil
		}
		ids := make([]int64, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		result := tx.Model(&Booking{}).Where("id IN ? AND status = ?", ids, BookingReserved).Updates(map[string]any{"status": BookingNoShow, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != int64(len(items)) {
			return fmt.Errorf("mark no-shows: expected %d updates, got %d", len(items), result.RowsAffected)
		}
		return nil
	})
	return items, e
}

func (r *Repository) CreateClosure(ctx context.Context, item *Closure, now time.Time) ([]int64, error) {
	db, e := r.gormDB(ctx)
	if e != nil {
		return nil, e
	}
	roots := []int64{}
	e = db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(item).Error; e != nil {
			return e
		}
		slotQuery := tx.Model(&Slot{}).Where("start_at < ? AND end_at > ?", item.EndAt, item.StartAt)
		if item.ResourceID != nil {
			slotQuery = slotQuery.Where("resource_id = ?", *item.ResourceID)
		} else if item.VenueID != nil {
			slotQuery = slotQuery.Where("resource_id IN (?)", tx.Model(&Resource{}).Select("id").Where("venue_id = ?", *item.VenueID))
		}
		var slots []Slot
		if e := slotQuery.Clauses(clause.Locking{Strength: "UPDATE"}).Find(&slots).Error; e != nil {
			return e
		}
		if len(slots) == 0 {
			return nil
		}
		slotIDs := make([]int64, 0, len(slots))
		for _, slot := range slots {
			slotIDs = append(slotIDs, slot.ID)
		}
		if e := tx.Model(&Booking{}).Where("slot_id IN ? AND status = ?", slotIDs, BookingReserved).Distinct().Pluck("root_user_id", &roots).Error; e != nil {
			return e
		}
		if e := tx.Model(&Booking{}).Where("slot_id IN ? AND status = ?", slotIDs, BookingReserved).Updates(map[string]any{"status": BookingClosureCancelled, "cancelled_at": now, "updated_at": now}).Error; e != nil {
			return e
		}
		return tx.Model(&Slot{}).Where("id IN ?", slotIDs).Updates(map[string]any{"status": SlotClosed, "reserved_count": 0, "updated_at": now}).Error
	})
	return roots, e
}
