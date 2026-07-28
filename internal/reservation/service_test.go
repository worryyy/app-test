package reservation

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type notificationRecord struct {
	rootUserID int64
	eventType  string
	resourceID string
}

type notifierStub struct{ records []notificationRecord }

func (n *notifierStub) NotifyReservation(_ context.Context, rootUserID int64, eventType, _, _, resourceID string) error {
	n.records = append(n.records, notificationRecord{rootUserID: rootUserID, eventType: eventType, resourceID: resourceID})
	return nil
}

func TestBookingCapacityOverlapAndDailyLimit(t *testing.T) {
	svc, db, now := reservationTestService(t)
	seedReservationInventory(t, db, now)

	first, err := svc.CreateBooking(context.Background(), 101, 100, CreateBookingReq{SlotID: "21"})
	if err != nil {
		t.Fatalf("create first booking: %v", err)
	}
	if first.ID == "" || first.Status != BookingReserved {
		t.Fatalf("first booking = %+v", first)
	}
	if _, err := svc.CreateBooking(context.Background(), 201, 200, CreateBookingReq{SlotID: "21"}); !errors.Is(err, ErrSlotUnavailable) {
		t.Fatalf("capacity error = %v", err)
	}
	if _, err := svc.CreateBooking(context.Background(), 101, 100, CreateBookingReq{SlotID: "22"}); !errors.Is(err, ErrBookingConflict) {
		t.Fatalf("overlap error = %v", err)
	}
	if _, err := svc.CreateBooking(context.Background(), 101, 100, CreateBookingReq{SlotID: "23"}); err != nil {
		t.Fatalf("create second booking: %v", err)
	}
	if _, err := svc.CreateBooking(context.Background(), 101, 100, CreateBookingReq{SlotID: "24"}); !errors.Is(err, ErrDailyLimit) {
		t.Fatalf("daily limit error = %v", err)
	}
}

func TestCancelDeadlineAndInventoryRelease(t *testing.T) {
	svc, db, now := reservationTestService(t)
	seedReservationInventory(t, db, now)
	booking, err := svc.CreateBooking(context.Background(), 101, 100, CreateBookingReq{SlotID: "21"})
	if err != nil {
		t.Fatal(err)
	}

	svc.now = func() time.Time { return now.Add(2 * time.Hour) }
	if err := svc.CancelBooking(context.Background(), 100, booking.ID); !errors.Is(err, ErrCancelDeadline) {
		t.Fatalf("deadline error = %v", err)
	}
	svc.now = func() time.Time { return now.Add(2*time.Hour - time.Second) }
	if err := svc.CancelBooking(context.Background(), 100, booking.ID); err != nil {
		t.Fatalf("cancel booking: %v", err)
	}
	var slot Slot
	if err := db.Where("id = ?", 21).Take(&slot).Error; err != nil {
		t.Fatal(err)
	}
	if slot.ReservedCount != 0 {
		t.Fatalf("reserved count = %d", slot.ReservedCount)
	}
	if err := svc.CancelBooking(context.Background(), 100, booking.ID); !errors.Is(err, ErrBookingNotFound) {
		t.Fatalf("second cancel error = %v", err)
	}
}

func TestCheckinAndNoShowJobsAreIdempotent(t *testing.T) {
	svc, db, now := reservationTestService(t)
	seedReservationInventory(t, db, now)
	notifier := &notifierStub{}
	svc.SetNotifier(notifier)
	endedSlot := Slot{ID: 25, ResourceID: 11, StartAt: now.Add(-2 * time.Hour), EndAt: now.Add(-time.Hour), Capacity: 1, ReservedCount: 1, Status: SlotOpen, CreatedAt: now, UpdatedAt: now}
	bookings := []Booking{
		{ID: 31, RootUserID: 100, UserID: 101, SlotID: 25, Status: BookingReserved, CheckinCode: "check-once", CreatedAt: now, UpdatedAt: now},
		{ID: 32, RootUserID: 200, UserID: 201, SlotID: 25, Status: BookingReserved, CheckinCode: "no-show", CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&endedSlot).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&bookings).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.Checkin(context.Background(), "check-once"); err != nil {
		t.Fatalf("check in: %v", err)
	}
	if err := svc.Checkin(context.Background(), "check-once"); !errors.Is(err, ErrCheckinInvalid) {
		t.Fatalf("second checkin error = %v", err)
	}
	if err := svc.RunDueJobs(context.Background()); err != nil {
		t.Fatalf("run jobs: %v", err)
	}
	if err := svc.RunDueJobs(context.Background()); err != nil {
		t.Fatalf("run jobs twice: %v", err)
	}
	var noShow Booking
	if err := db.Where("id = ?", 32).Take(&noShow).Error; err != nil {
		t.Fatal(err)
	}
	if noShow.Status != BookingNoShow {
		t.Fatalf("no-show status = %s", noShow.Status)
	}
	if len(notifier.records) != 1 || notifier.records[0].eventType != "reservation.noShow" || notifier.records[0].rootUserID != 200 {
		t.Fatalf("notifications = %+v", notifier.records)
	}
}

func TestClosureCancelsBookingsAndReleasesInventory(t *testing.T) {
	svc, db, now := reservationTestService(t)
	seedReservationInventory(t, db, now)
	notifier := &notifierStub{}
	svc.SetNotifier(notifier)
	booking, err := svc.CreateBooking(context.Background(), 101, 100, CreateBookingReq{SlotID: "21"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.CreateClosure(context.Background(), 9, ClosureReq{
		ResourceID: "11",
		StartAt:    now.Add(3 * time.Hour).Format(time.RFC3339),
		EndAt:      now.Add(5 * time.Hour).Format(time.RFC3339),
		Reason:     "maintenance",
	}); err != nil {
		t.Fatalf("create closure: %v", err)
	}
	var got Booking
	if err := db.Where("id = ?", mustReservationID(t, booking.ID)).Take(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != BookingClosureCancelled {
		t.Fatalf("booking status = %s", got.Status)
	}
	var slot Slot
	if err := db.Where("id = ?", 21).Take(&slot).Error; err != nil {
		t.Fatal(err)
	}
	if slot.Status != SlotClosed || slot.ReservedCount != 0 {
		t.Fatalf("closed slot = %+v", slot)
	}
	if len(notifier.records) != 1 || notifier.records[0].eventType != "reservation.closure" {
		t.Fatalf("notifications = %+v", notifier.records)
	}
}

func reservationTestService(t *testing.T) (*Service, *gorm.DB, time.Time) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Venue{}, &Resource{}, &WeeklyRule{}, &Closure{}, &Slot{}, &UserDayLock{}, &Booking{}); err != nil {
		t.Fatal(err)
	}
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 7, 28, 10, 0, 0, 0, location)
	svc := NewService(db, nil)
	svc.location = location
	svc.now = func() time.Time { return now }
	return svc, db, now
}

func seedReservationInventory(t *testing.T, db *gorm.DB, now time.Time) {
	t.Helper()
	venue := Venue{ID: 1, Name: "体育馆", Status: StatusActive, AdvanceDays: 7, SlotMinutes: 60, DailyLimit: 2, CancelBeforeMinutes: 120, CreatedAt: now, UpdatedAt: now}
	resources := []Resource{
		{ID: 11, VenueID: 1, Name: "一号场", Capacity: 1, Status: StatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: 12, VenueID: 1, Name: "二号场", Capacity: 1, Status: StatusActive, CreatedAt: now, UpdatedAt: now},
	}
	slots := []Slot{
		{ID: 21, ResourceID: 11, StartAt: now.Add(4 * time.Hour), EndAt: now.Add(5 * time.Hour), Capacity: 1, Status: SlotOpen, CreatedAt: now, UpdatedAt: now},
		{ID: 22, ResourceID: 12, StartAt: now.Add(4*time.Hour + 30*time.Minute), EndAt: now.Add(5*time.Hour + 30*time.Minute), Capacity: 1, Status: SlotOpen, CreatedAt: now, UpdatedAt: now},
		{ID: 23, ResourceID: 12, StartAt: now.Add(5 * time.Hour), EndAt: now.Add(6 * time.Hour), Capacity: 1, Status: SlotOpen, CreatedAt: now, UpdatedAt: now},
		{ID: 24, ResourceID: 12, StartAt: now.Add(6 * time.Hour), EndAt: now.Add(7 * time.Hour), Capacity: 1, Status: SlotOpen, CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&venue).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&resources).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&slots).Error; err != nil {
		t.Fatal(err)
	}
}

func mustReservationID(t *testing.T, raw string) int64 {
	t.Helper()
	value, err := parseID(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
