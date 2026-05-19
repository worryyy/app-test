package metrics

import (
	"context"
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

type CheckFunc func(context.Context) error

type DependencyChecks struct {
	MySQL    CheckFunc
	MongoDB  CheckFunc
	Redis    CheckFunc
	RabbitMQ CheckFunc
}

type dependencyCheck struct {
	name  string
	check CheckFunc
}

var errDependencyUnavailable = errors.New("dependency unavailable")

func NewDependencyChecks(mysql *gorm.DB, mongoClient *mongo.Client, redisClient *redis.Client, rabbitMQ *amqp.Connection) DependencyChecks {
	return DependencyChecks{
		MySQL:    mysqlCheck(mysql),
		MongoDB:  mongoCheck(mongoClient),
		Redis:    redisCheck(redisClient),
		RabbitMQ: rabbitMQCheck(rabbitMQ),
	}
}

func (d DependencyChecks) all() []dependencyCheck {
	return []dependencyCheck{
		{name: "mysql", check: d.MySQL},
		{name: "mongodb", check: d.MongoDB},
		{name: "redis", check: d.Redis},
		{name: "rabbitmq", check: d.RabbitMQ},
	}
}

func mysqlCheck(db *gorm.DB) CheckFunc {
	return func(ctx context.Context) error {
		if db == nil {
			return errDependencyUnavailable
		}
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.PingContext(ctx)
	}
}

func mongoCheck(client *mongo.Client) CheckFunc {
	return func(ctx context.Context) error {
		if client == nil {
			return errDependencyUnavailable
		}
		return client.Ping(ctx, nil)
	}
}

func redisCheck(client *redis.Client) CheckFunc {
	return func(ctx context.Context) error {
		if client == nil {
			return errDependencyUnavailable
		}
		return client.Ping(ctx).Err()
	}
}

func rabbitMQCheck(conn *amqp.Connection) CheckFunc {
	return func(context.Context) error {
		if conn == nil || conn.IsClosed() {
			return errDependencyUnavailable
		}
		return nil
	}
}
