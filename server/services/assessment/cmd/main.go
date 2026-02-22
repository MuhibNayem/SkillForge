package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/amnayem/skillforge/services/assessment"
	assessmentpb "github.com/amnayem/skillforge/shared/pb/assessment"
	"github.com/amnayem/skillforge/shared/pkg/auth"
	"github.com/amnayem/skillforge/shared/pkg/config"
	"github.com/amnayem/skillforge/shared/pkg/db"
	"github.com/amnayem/skillforge/shared/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()
	logger.Init(cfg.Env)
	log := logger.Get()
	log.Info().Msg("Starting Assessment Service...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.ConnectPostgres(ctx, cfg.DBUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to PostgreSQL")
	}
	defer pool.Close()

	repo := assessment.NewPostgresRepository(pool)
	handler := assessment.NewHandler(repo)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Port))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to listen")
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(auth.UnaryInterceptor(cfg.JWTSecret)),
	)
	assessmentpb.RegisterAssessmentServiceServer(grpcServer, handler)
	reflection.Register(grpcServer)

	go func() {
		log.Info().Str("port", cfg.Port).Msg("Assessment Service gRPC server listening")
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
