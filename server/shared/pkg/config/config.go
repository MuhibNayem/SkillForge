package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Env       string
	Port      string
	DBUrl     string
	RedisAddr string
	RedisPass string
	MongoURI  string
	MinioURL  string
	KafkaUrl  string
	JWTSecret string
}

func LoadConfig() *Config {
	_ = godotenv.Load()
	return &Config{
		Env:       getEnv("APP_ENV", "development"),
		Port:      getEnv("PORT", "9090"),
		DBUrl:     getEnv("DB_URL", "postgres://learnhub:password@localhost:5432/learnhub?sslmode=disable"),
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass: getEnv("REDIS_PASS", ""),
		MongoURI:  getEnv("MONGO_URI", "mongodb://admin:password@localhost:27017"),
		MinioURL:  getEnv("MINIO_URL", "http://localhost:9000"),
		KafkaUrl:  getEnv("KAFKA_URL", "localhost:9092"),
		JWTSecret: getEnv("JWT_SECRET", "super-secret-key-for-dev"),
	}
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
