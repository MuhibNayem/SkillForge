package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	content "github.com/amnayem/skillforge/services/content"
	"github.com/amnayem/skillforge/shared/pb/contentpb"
	"github.com/amnayem/skillforge/shared/pkg/config"
	dbpkg "github.com/amnayem/skillforge/shared/pkg/db"
	"github.com/amnayem/skillforge/shared/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()
	logger.Init(cfg.Env)
	log := logger.Get()
	log.Info().Msg("Starting Content Service...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mongoClient, err := dbpkg.ConnectMongo(ctx, cfg.MongoURI)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to MongoDB")
	}
	defer mongoClient.Disconnect(ctx)

	mongoDB := mongoClient.Database("learnhub_content")
	repo := content.NewMongoRepository(mongoDB, cfg.MinioURL)
	handler := content.NewHandler(repo, cfg.MinioURL)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Port))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to listen")
	}

	grpcServer := grpc.NewServer()
	contentpb.RegisterContentServiceServer(grpcServer, handler)
	reflection.Register(grpcServer)

	go func() {
		log.Info().Str("port", cfg.Port).Msg("Content Service gRPC server listening")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("Failed to serve gRPC")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down...")
	grpcServer.GracefulStop()
}
