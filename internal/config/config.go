package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL  string
	KafkaBrokers []string
	KafkaTopic   string
	KafkaGroup   string
	HTTPAddr     string
	CacheTTL     time.Duration
}

func LoadFromEnv() (*Config, error) {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbPass == "" || dbName == "" {
		return nil, errors.New("database configuration missing (DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME)")
	}

	dsn := "postgres://" + dbUser + ":" + dbPass + "@" + dbHost + ":" + dbPort + "/" + dbName + "?sslmode=disable"

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}

	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	if kafkaTopic == "" {
		kafkaTopic = "orders"
	}

	kafkaGroup := os.Getenv("KAFKA_GROUP")
	if kafkaGroup == "" {
		kafkaGroup = "order-service-group"
	}

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	ttlSec := 600
	if v := os.Getenv("CACHE_TTL_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			ttlSec = parsed
		}
	}

	return &Config{
		DatabaseURL:  dsn,
		KafkaBrokers: []string{kafkaBrokers},
		KafkaTopic:   kafkaTopic,
		KafkaGroup:   kafkaGroup,
		HTTPAddr:     httpAddr,
		CacheTTL:     time.Duration(ttlSec) * time.Second,
	}, nil
}
