package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	MySQL      MySQLConfig      `mapstructure:"mysql"`
	Mongo      MongoConfig      `mapstructure:"mongo"`
	Redis      RedisConfig      `mapstructure:"redis"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	COS        COSConfig        `mapstructure:"cos"`
	WX         WXConfig         `mapstructure:"wx"`
	RabbitMQ   RabbitMQConfig   `mapstructure:"rabbitmq"`
	Custom     CustomConfig     `mapstructure:"custom"`
	JW         JWConfig         `mapstructure:"jw"`
	Encryption EncryptionConfig `mapstructure:"encryption"`
	Admin      AdminConfig      `mapstructure:"admin"`
	Logging    LoggingConfig    `mapstructure:"logging"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type MySQLConfig struct {
	DSN         string `mapstructure:"dsn"`
	MaxLifetime int    `mapstructure:"max_lifetime"`
}

type MongoConfig struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type RabbitMQConfig struct {
	URL string `mapstructure:"url"`
}

type JWTConfig struct {
	Secret              string `mapstructure:"secret"`
	TokenMinutes        int    `mapstructure:"token_minutes"`
	RefreshTokenMinutes int    `mapstructure:"refresh_token_minutes"`
	Issue               string `mapstructure:"issue"`
}

type COSConfig struct {
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	BucketName      string `mapstructure:"bucket_name"`
	Region          string `mapstructure:"region"`
	BaseURL         string `mapstructure:"base_url"`
	BaseCDN         string `mapstructure:"base_cdn"`
	Compress        string `mapstructure:"compress"`
	CompressBucket  string `mapstructure:"compress_bucket_name"`
	CompressBaseCDN string `mapstructure:"compress_base_cdn"`
}

type WXConfig struct {
	AppID  string `mapstructure:"appid"`
	Secret string `mapstructure:"secret"`
}

type CustomConfig struct {
	DefaultAvatar          string `mapstructure:"default_avatar"`
	DefaultAnonymousAvatar string `mapstructure:"default_anonymous_avatar"`
	PageSize               int    `mapstructure:"page_size"`
	MaxFileSizeMB          int    `mapstructure:"max_file_size_mb"`
}

type JWConfig struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
}

type EncryptionConfig struct {
	Key string `mapstructure:"key"`
}

type AdminConfig struct {
	PowerSign int `mapstructure:"power_sign"`
}

type LoggingConfig struct {
	Level    string `mapstructure:"level"`
	FilePath string `mapstructure:"file_path"`
}

func Load(configDir string) *Config {
	v := viper.New()
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)

	v.SetConfigName("application")
	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Errorf("read base config: %w", err))
	}

	profile := os.Getenv("APP_PROFILE")
	if profile == "" {
		profile = "dev"
	}

	v.SetConfigName("application-" + profile)
	if err := v.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(fmt.Errorf("read profile config (%s): %w", profile, err))
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("unmarshal config: %w", err))
	}

	return &cfg
}
