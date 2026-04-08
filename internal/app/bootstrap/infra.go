package bootstrap

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

	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/snowflake"
)

type Options struct {
	ConfigDir     string
	WithRabbitMQ  bool
	SnowflakeNode int64
}

type Infra struct {
	Config      *config.Config
	Logger      *zap.Logger
	MySQL       *gorm.DB
	Mongo       *mongo.Database
	MongoClient *mongo.Client
	Redis       *redis.Client
	RabbitMQ    *amqp.Connection
}

func LoadInfrastructure(opts Options) (*Infra, error) {
	if opts.ConfigDir == "" {
		return nil, fmt.Errorf("config dir is empty")
	}
	if opts.SnowflakeNode == 0 {
		opts.SnowflakeNode = 1
	}

	_, _ = time.LoadLocation("Asia/Shanghai")

	infra := &Infra{
		Config: config.Load(opts.ConfigDir),
	}

	logger, err := initLogger(infra.Config)
	if err != nil {
		return nil, err
	}
	infra.Logger = logger

	ok := false
	defer func() {
		if !ok {
			infra.Close(context.Background())
		}
	}()

	if err := snowflake.Init(opts.SnowflakeNode); err != nil {
		return nil, fmt.Errorf("init snowflake: %w", err)
	}

	db, err := initMySQL(infra.Config)
	if err != nil {
		return nil, err
	}
	infra.MySQL = db

	mongoClient, mongoDB, err := initMongo(infra.Config)
	if err != nil {
		return nil, err
	}
	infra.MongoClient = mongoClient
	infra.Mongo = mongoDB

	rds, err := initRedis(infra.Config)
	if err != nil {
		return nil, err
	}
	infra.Redis = rds

	if opts.WithRabbitMQ {
		conn, err := initRabbitMQ(infra.Config)
		if err != nil {
			return nil, err
		}
		infra.RabbitMQ = conn
	}

	ok = true
	return infra, nil
}

func (i *Infra) Close(ctx context.Context) {
	if i == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if i.RabbitMQ != nil {
		if err := i.RabbitMQ.Close(); err != nil {
			i.warn("close rabbitmq connection failed", err)
		}
	}
	if i.Redis != nil {
		if err := i.Redis.Close(); err != nil {
			i.warn("close redis failed", err)
		}
	}
	if i.MongoClient != nil {
		if err := i.MongoClient.Disconnect(ctx); err != nil {
			i.warn("disconnect mongodb failed", err)
		}
	}
	if i.MySQL != nil {
		sqlDB, err := i.MySQL.DB()
		if err != nil {
			i.warn("get mysql sql db failed", err)
		} else if err := sqlDB.Close(); err != nil {
			i.warn("close mysql failed", err)
		}
	}
	if i.Logger != nil {
		if err := i.Logger.Sync(); err != nil {
			i.warn("sync logger failed", err)
		}
	}
}

func (i *Infra) warn(message string, err error) {
	if err == nil {
		return
	}
	if i == nil || i.Logger == nil {
		return
	}
	i.Logger.Warn(message, zap.Error(err))
}

func initLogger(cfg *config.Config) (*zap.Logger, error) {
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

func initMySQL(cfg *config.Config) (*gorm.DB, error) {
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

func initMongo(cfg *config.Config) (*mongo.Client, *mongo.Database, error) {
	if cfg == nil {
		return nil, nil, fmt.Errorf("config is nil")
	}

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		return nil, nil, fmt.Errorf("connect mongodb: %w", err)
	}
	if err := client.Ping(context.Background(), nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, fmt.Errorf("ping mongodb: %w", err)
	}
	return client, client.Database(cfg.Mongo.Database), nil
}

func initRedis(cfg *config.Config) (*redis.Client, error) {
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

func initRabbitMQ(cfg *config.Config) (*amqp.Connection, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	conn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		return nil, fmt.Errorf("connect rabbitmq: %w", err)
	}
	return conn, nil
}
