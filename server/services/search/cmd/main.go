package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	search "github.com/amnayem/skillforge/services/search"
	searchpb "github.com/amnayem/skillforge/shared/pb/search"
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
	log.Info().Msg("Starting Search Service...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Elasticsearch
	esAddr := getEnv("ELASTICSEARCH_URL", "http://elasticsearch:9200")
	esCfg := es.Config{Addresses: []string{esAddr}}
	esClient, err := es.NewClient(esCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Elasticsearch client")
	}

	repo := search.NewElasticsearchRepository(esClient)
	handler := search.NewHandler(repo)

	// Kafka indexing consumer
	kafkaBrokers := strings.Split(getEnv("KAFKA_BROKERS", "kafka:29092"), ",")
	consumer := search.NewIndexConsumer(kafkaBrokers, "search-service", repo)
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
	searchpb.RegisterSearchServiceServer(grpcServer, handler)
	reflection.Register(grpcServer)

	go func() {
		log.Info().Str("port", cfg.Port).Msg("Search Service gRPC server listening")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("Failed to serve gRPC")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down Search Service...")
	grpcServer.GracefulStop()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
