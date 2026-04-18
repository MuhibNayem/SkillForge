package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	discussion "github.com/learnhub/lms/server/services/discussion/internal/handler"
	"github.com/learnhub/lms/server/services/discussion/internal/repository"
	"github.com/learnhub/lms/server/shared/pb/discussion"
	"github.com/learnhub/lms/server/shared/pkg/config"
	"github.com/learnhub/lms/server/shared/pkg/db"
	"github.com/learnhub/lms/server/shared/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()
	logger.Init(cfg.Env)
	log := logger.Get()
	log.Info().Msg("Starting Discussion Service...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to MongoDB for discussions (flexible schema for threads/replies)
	mongoClient, err := db.ConnectMongoDB(ctx, cfg.MongoDBURI)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to MongoDB")
	}
	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Error().Err(err).Msg("Error disconnecting MongoDB")
		}
	}()

	// Connect to Redis for caching and real-time features
	rdb, err := db.ConnectRedis(ctx, cfg.RedisAddr, cfg.RedisPass, 0)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer rdb.Close()

	// Create repository and handler
	repo := repository.NewMongoRepository(mongoClient, rdb)
	handler := discussion.NewHandler(repo)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Port))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to listen")
	}

	grpcServer := grpc.NewServer()
	discussionpb.RegisterDiscussionServiceServer(grpcServer, handler)
	reflection.Register(grpcServer)

	go func() {
		log.Info().Str("port", cfg.Port).Msg("Discussion Service gRPC server listening")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("Failed to serve gRPC")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down Discussion Service...")
	grpcServer.GracefulStop()
}
