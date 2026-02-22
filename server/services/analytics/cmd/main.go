package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	analytics "github.com/amnayem/skillforge/services/analytics"
	analyticspb "github.com/amnayem/skillforge/shared/pb/analytics"
	"github.com/amnayem/skillforge/shared/pkg/auth"
	"github.com/amnayem/skillforge/shared/pkg/config"
	"github.com/amnayem/skillforge/shared/pkg/logger"
	es "github.com/elastic/go-elasticsearch/v8"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()
	logger.Init(cfg.Env)
	log := logger.Get()
	log.Info().Msg("Starting Analytics Service...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Elasticsearch
	esAddr := getEnv("ELASTICSEARCH_URL", "http://elasticsearch:9200")
	esCfg := es.Config{Addresses: []string{esAddr}}
	esClient, err := es.NewClient(esCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Elasticsearch client")
	}

	repo := analytics.NewElasticsearchRepository(esClient)
	handler := analytics.NewHandler(repo)

	// Kafka consumer
	kafkaBrokers := strings.Split(getEnv("KAFKA_BROKERS", "kafka:29092"), ",")
	consumer := analytics.NewAnalyticsConsumer(kafkaBrokers, "analytics-service", repo)
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
	analyticspb.RegisterAnalyticsServiceServer(grpcServer, handler)
	reflection.Register(grpcServer)

	go func() {
		log.Info().Str("port", cfg.Port).Msg("Analytics Service gRPC server listening")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("Failed to serve gRPC")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down Analytics Service...")
	grpcServer.GracefulStop()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
