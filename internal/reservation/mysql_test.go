package reservation

import (
	"context"
	"errors"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/snowflake"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMySQLConcurrentBookingDoesNotOversell(t *testing.T) {
	dsn := os.Getenv("ECAMPUS_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("ECAMPUS_TEST_MYSQL_DSN is not set")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := snowflake.Init(1); err != nil {
		t.Fatal(err)
	}
	base := time.Now().UnixNano() / 1000
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 7, 28, 10, 0, 0, 0, location)
	venue := Venue{ID: base, Name: "mysql-concurrency", Status: StatusActive, AdvanceDays: 7, SlotMinutes: 60, DailyLimit: 2, CancelBeforeMinutes: 120, CreatedAt: now, UpdatedAt: now}
	resource := Resource{ID: base + 1, VenueID: venue.ID, Name: "capacity-three", Capacity: 3, Status: StatusActive, CreatedAt: now, UpdatedAt: now}
	slot := Slot{ID: base + 2, ResourceID: resource.ID, StartAt: now.Add(4 * time.Hour), EndAt: now.Add(5 * time.Hour), Capacity: 3, Status: SlotOpen, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&venue).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&resource).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&slot).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Where("slot_id = ?", slot.ID).Delete(&Booking{})
		db.Where("root_user_id >= ? AND root_user_id < ?", base+100, base+200).Delete(&UserDayLock{})
		db.Delete(&Slot{}, slot.ID)
		db.Delete(&Resource{}, resource.ID)
		db.Delete(&Venue{}, venue.ID)
	})

	service := NewService(db, nil)
	service.now = func() time.Time { return now }
	var successes atomic.Int64
	errorsChannel := make(chan error, 8)
	var wait sync.WaitGroup
	for i := int64(0); i < 8; i++ {
		wait.Add(1)
		go func(offset int64) {
			defer wait.Done()
			rootUserID := base + 100 + offset
			_, err := service.CreateBooking(context.Background(), rootUserID, rootUserID, CreateBookingReq{SlotID: strconv.FormatInt(slot.ID, 10)})
			if err == nil {
				successes.Add(1)
				return
			}
			if !errors.Is(err, ErrSlotUnavailable) {
				errorsChannel <- err
			}
		}(i)
	}
	wait.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		t.Errorf("unexpected booking error: %v", err)
	}
	if successes.Load() != 3 {
		t.Fatalf("successful bookings = %d, want 3", successes.Load())
	}
	var stored Slot
	if err := db.Where("id = ?", slot.ID).Take(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ReservedCount != 3 {
		t.Fatalf("reserved count = %d, want 3", stored.ReservedCount)
	}
}
