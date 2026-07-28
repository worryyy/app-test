package marketplace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMySQLConcurrentOrdersKeepSingleActiveOrder(t *testing.T) {
	dsn := os.Getenv("ECAMPUS_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("ECAMPUS_TEST_MYSQL_DSN is not set")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	base := time.Now().UnixNano() / 1000
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 7, 28, 10, 0, 0, 0, location)
	category := Category{ID: base, Name: fmt.Sprintf("mysql-%d", base), CommissionRateBps: 500, Status: StatusActive, CreatedAt: now, UpdatedAt: now}
	item := Item{ID: base + 1, SellerRootUserID: base + 10, SellerUserID: base + 11, CategoryID: category.ID, Title: "single item", Description: "mysql concurrency", Condition: "good", PriceCents: 999, Images: []string{"0123456789abcdef0123456789abcdef"}, DeliveryLocation: "campus", Status: ItemPublished, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&category).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Where("item_id = ?", item.ID).Delete(&Order{})
		db.Delete(&Item{}, item.ID)
		db.Delete(&Category{}, category.ID)
	})

	service := NewService(db, NewGateway("test"), nil)
	service.now = func() time.Time { return now }
	var successes atomic.Int64
	errorsChannel := make(chan error, 8)
	var wait sync.WaitGroup
	for i := int64(0); i < 8; i++ {
		wait.Add(1)
		go func(offset int64) {
			defer wait.Done()
			buyerRootUserID := base + 100 + offset
			_, err := service.CreateOrder(context.Background(), buyerRootUserID, buyerRootUserID, CreateOrderReq{ItemID: fmt.Sprintf("%d", item.ID)})
			if err == nil {
				successes.Add(1)
				return
			}
			if !errors.Is(err, ErrItemUnavailable) {
				errorsChannel <- err
			}
		}(i)
	}
	wait.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		t.Errorf("unexpected order error: %v", err)
	}
	if successes.Load() != 1 {
		t.Fatalf("successful orders = %d, want 1", successes.Load())
	}
	var count int64
	if err := db.Model(&Order{}).Where("item_id = ?", item.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("order count = %d, want 1", count)
	}
	var stored Item
	if err := db.Where("id = ?", item.ID).Take(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != ItemReserved || stored.ReservedOrderID == nil {
		t.Fatalf("stored item = %+v", stored)
	}
}
