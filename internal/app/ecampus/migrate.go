package ecampus

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/agentchat"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
)

const ecampusConfigDir = "configs/ecampus"

func RunMigrations() error {
	cfg := config.Load(ecampusConfigDir)

	db, closeDB, err := openMigrationDB(cfg)
	if err != nil {
		return err
	}
	defer closeDB()

	if err := agentchat.EnsureSchema(db); err != nil {
		return err
	}
	return nil
}

func openMigrationDB(cfg *config.Config) (*gorm.DB, func(), error) {
	if cfg == nil {
		return nil, func() {}, fmt.Errorf("config is nil")
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{})
	if err != nil {
		return nil, func() {}, fmt.Errorf("open mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, func() {}, fmt.Errorf("get mysql sql db: %w", err)
	}
	if cfg.MySQL.MaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.MySQL.MaxLifetime) * time.Millisecond)
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, func() {}, fmt.Errorf("ping mysql: %w", err)
	}

	return db, func() {
		_ = sqlDB.Close()
	}, nil
}
