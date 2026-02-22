package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	notification "github.com/amnayem/skillforge/services/notification"
	notificationpb "github.com/amnayem/skillforge/shared/pb/notification"
	"github.com/amnayem/skillforge/shared/pkg/auth"
	"github.com/amnayem/skillforge/shared/pkg/config"
	"github.com/amnayem/skillforge/shared/pkg/db"
	"github.com/amnayem/skillforge/shared/pkg/logger"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()
	logger.Init(cfg.Env)
	log := logger.Get()
	log.Info().Msg("Starting Notification Service...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// PostgreSQL
	pool, err := db.ConnectPostgres(ctx, cfg.DBUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to PostgreSQL")
	}
	defer pool.Close()

	// Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})

	repo := notification.NewPostgresRepository(pool)
	handler := notification.NewHandler(repo)

	// Kafka consumer (runs in background goroutine)
	kafkaBrokers := strings.Split(getEnv("KAFKA_BROKERS", "kafka:29092"), ",")
	topics := []string{
		"user.registered",
		"enrollment.created",
		"assessment.graded",
		"course.completed",
		"course.published",
	}
	smtpCfg := notification.SMTPConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     587,
		Username: os.Getenv("SMTP_USER"),
		Password: os.Getenv("SMTP_PASS"),
		From:     getEnv("SMTP_FROM", "noreply@skillforge.io"),
	}
	consumer := notification.NewConsumer(kafkaBrokers, "notification-service", topics, repo, rdb, smtpCfg)
	defer consumer.Close()
	go consumer.Run(ctx)

	// gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Port))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to listen")
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(auth.UnaryInterceptor(cfg.JWTSecret)),
	)
	notificationpb.RegisterNotificationServiceServer(grpcServer, handler)
	reflection.Register(grpcServer)

	go func() {
		log.Info().Str("port", cfg.Port).Msg("Notification Service gRPC server listening")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("Failed to serve gRPC")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down Notification Service...")
	grpcServer.GracefulStop()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
