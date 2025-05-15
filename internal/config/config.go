package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env             string `env:"ENV" env-default:"prod"`
	Port            int    `env:"PORT" env-default:"50055"`
	AuthServiceAddr string `env:"AUTH_SERVICE_ADDR" env-required:"true"`
	DB              DBConfig
	Minio           MinioConfig
	// Redis           RedisConfig `env:"REDIS"`
}

type DBConfig struct {
	Host       string `env:"DB_HOST" env-default:"localhost"`
	Port       int    `env:"DB_PORT" env-default:"27017"`
	User       string `env:"DB_USER" env-default:"mongo"`
	Password   string `env:"DB_PASSWORD" env-required:"true"`
	Database   string `env:"DB_DATABASE" env-default:"users"`
	Collection string `env:"DB_COLLECTION" env-default:"users"`
}

type MinioConfig struct {
	Endpoint   string `env:"MINIO_ENDPOINT" env-default:"localhost:9000"`
	User       string `env:"MINIO_ROOT_USER" env-default:"minio"`
	Password   string `env:"MINIO_ROOT_PASSWORD" env-required:"true"`
	BucketName string `env:"MINIO_BUCKET_NAME" env-default:"users"`
	IsUseSsl   bool   `env:"MINIO_USE_SSL" env-default:"false"`
}

// type RedisConfig struct {
// 	Addr     string `env:"REDIS_ADDR" env-default:"localhost:6379"`
// 	User     string `env:"REDIS_USER" env-default:"root"`
// 	Password string `env:"REDIS_USER_PASSWORD" env-default:"root"`
// 	DB       int    `env:"REDIS_DB" env-default:"0"`
// }

func MustLoad() *Config {
	path := fetchPath()
	cfg, err := Load(path)
	if err != nil {
		panic(err)
	}
	return cfg
}

func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", path)
	}

	cfg := &Config{}

	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

func fetchPath() string {
	var path string

	flag.StringVar(&path, "config", "", "path to config file")
	flag.Parse()

	if path == "" {
		path = os.Getenv("CONFIG_PATH")
	}

	return path
}
