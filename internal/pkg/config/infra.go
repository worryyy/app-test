package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitLogger(cfg *Config) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	if cfg != nil && cfg.Logging.Level != "" {
		if err := level.UnmarshalText([]byte(cfg.Logging.Level)); err != nil {
			return nil, fmt.Errorf("parse log level: %w", err)
		}
	}

	logCfg := zap.NewProductionConfig()
	logCfg.Level = zap.NewAtomicLevelAt(level)
	logCfg.EncoderConfig.TimeKey = "time"
	logCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	outputPaths := []string{"stdout"}
	if cfg != nil && cfg.Logging.FilePath != "" {
		dir := filepath.Dir(cfg.Logging.FilePath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create log directory: %w", err)
		}
		outputPaths = append(outputPaths, cfg.Logging.FilePath)
	}
	logCfg.OutputPaths = outputPaths
	logCfg.ErrorOutputPaths = outputPaths

	logger, err := logCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}
	return logger, nil
}

func InitMySQL(cfg *Config) (*gorm.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get mysql sql db: %w", err)
	}
	if cfg.MySQL.MaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.MySQL.MaxLifetime) * time.Millisecond)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return db, nil
}

func InitMongo(cfg *Config) (*mongo.Database, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		return nil, fmt.Errorf("connect mongodb: %w", err)
	}
	if err := client.Ping(context.Background(), nil); err != nil {
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}
	return client.Database(cfg.Mongo.Database), nil
}

func InitRedis(cfg *Config) (*redis.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return client, nil
}

func InitRabbitMQ(cfg *Config) (*amqp.Connection, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	conn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		return nil, fmt.Errorf("connect rabbitmq: %w", err)
	}
	return conn, nil
}
