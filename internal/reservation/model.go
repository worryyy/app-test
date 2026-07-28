package reservation

import "time"

const (
	StatusActive            = "active"
	StatusDisabled          = "disabled"
	SlotOpen                = "open"
	SlotClosed              = "closed"
	BookingReserved         = "reserved"
	BookingCancelled        = "cancelled"
	BookingClosureCancelled = "closureCancelled"
	BookingCheckedIn        = "checkedIn"
	BookingNoShow           = "noShow"
)

type Venue struct {
	ID                  int64     `gorm:"column:id;primaryKey"`
	Name                string    `gorm:"column:name"`
	Description         string    `gorm:"column:description"`
	Status              string    `gorm:"column:status"`
	AdvanceDays         int       `gorm:"column:advance_days"`
	SlotMinutes         int       `gorm:"column:slot_minutes"`
	DailyLimit          int       `gorm:"column:daily_limit"`
	CancelBeforeMinutes int       `gorm:"column:cancel_before_minutes"`
	CreatedAt           time.Time `gorm:"column:created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at"`
}

func (Venue) TableName() string { return "reservation_venues" }

type Resource struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	VenueID   int64     `gorm:"column:venue_id"`
	Name      string    `gorm:"column:name"`
	Capacity  int       `gorm:"column:capacity"`
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (Resource) TableName() string { return "reservation_resources" }

type WeeklyRule struct {
	ID          int64     `gorm:"column:id;primaryKey"`
	ResourceID  int64     `gorm:"column:resource_id"`
	Weekday     int       `gorm:"column:weekday"`
	StartMinute int       `gorm:"column:start_minute"`
	EndMinute   int       `gorm:"column:end_minute"`
	Status      string    `gorm:"column:status"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (WeeklyRule) TableName() string { return "reservation_weekly_rules" }

type Closure struct {
	ID         int64     `gorm:"column:id;primaryKey"`
	VenueID    *int64    `gorm:"column:venue_id"`
	ResourceID *int64    `gorm:"column:resource_id"`
	StartAt    time.Time `gorm:"column:start_at"`
	EndAt      time.Time `gorm:"column:end_at"`
	Reason     string    `gorm:"column:reason"`
	CreatedBy  int64     `gorm:"column:created_by"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (Closure) TableName() string { return "reservation_closures" }

type Slot struct {
	ID            int64     `gorm:"column:id;primaryKey"`
	ResourceID    int64     `gorm:"column:resource_id"`
	StartAt       time.Time `gorm:"column:start_at"`
	EndAt         time.Time `gorm:"column:end_at"`
	Capacity      int       `gorm:"column:capacity"`
	ReservedCount int       `gorm:"column:reserved_count"`
	Status        string    `gorm:"column:status"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (Slot) TableName() string { return "reservation_slots" }

type UserDayLock struct {
	ID              int64     `gorm:"column:id;primaryKey"`
	RootUserID      int64     `gorm:"column:root_user_id;uniqueIndex:uk_reservation_user_day"`
	ReservationDate time.Time `gorm:"column:reservation_date;type:date;uniqueIndex:uk_reservation_user_day"`
}

func (UserDayLock) TableName() string { return "reservation_user_day_locks" }

type Booking struct {
	ID          int64      `gorm:"column:id;primaryKey"`
	RootUserID  int64      `gorm:"column:root_user_id"`
	UserID      int64      `gorm:"column:user_id"`
	SlotID      int64      `gorm:"column:slot_id"`
	Status      string     `gorm:"column:status"`
	CheckinCode string     `gorm:"column:checkin_code;uniqueIndex"`
	CheckedAt   *time.Time `gorm:"column:checked_at"`
	CancelledAt *time.Time `gorm:"column:cancelled_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
	Slot        Slot       `gorm:"foreignKey:SlotID"`
}

func (Booking) TableName() string { return "reservations" }
