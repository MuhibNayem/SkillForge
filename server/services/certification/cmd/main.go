package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/amnayem/skillforge/services/certification"
	certificationpb "github.com/amnayem/skillforge/shared/pb/certification"
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
	log.Info().Msg("Starting Certification Service...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.ConnectPostgres(ctx, cfg.DBUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to PostgreSQL")
	}
	defer pool.Close()

	repo := certification.NewPostgresRepository(pool)
	handler := certification.NewHandler(repo, cfg.MinioURL)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Port))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to listen")
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(auth.UnaryInterceptor(cfg.JWTSecret)),
	)
	certificationpb.RegisterCertificationServiceServer(grpcServer, handler)
	reflection.Register(grpcServer)

	go func() {
		log.Info().Str("port", cfg.Port).Msg("Certification Service gRPC server listening")
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
