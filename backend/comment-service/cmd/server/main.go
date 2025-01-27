package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/Josy-coder/comment-service/internal/clients"
	"github.com/Josy-coder/comment-service/internal/config"
	"github.com/Josy-coder/comment-service/internal/ports"
	"github.com/Josy-coder/comment-service/internal/service"
	pb "github.com/Josy-coder/comment-service/proto/comment/v1"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize service clients
	dbClient, err := clients.NewDBServiceClient(cfg.Services.DBServiceURL)
	if err != nil {
		log.Fatalf("Failed to create database service client: %v", err)
	}

	userClient, err := clients.NewUserServiceClient(cfg.Services.UserServiceURL)
	if err != nil {
		log.Fatalf("Failed to create user service client: %v", err)
	}

	streamClient, err := clients.NewStreamServiceClient(cfg.Services.StreamServiceURL)
	if err != nil {
		log.Fatalf("Failed to create stream service client: %v", err)
	}

	// Initialize comment service
	commentService := service.NewCommentService(dbClient, userClient, streamClient)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	commentServer := ports.NewGRPCServer(commentService)
	pb.RegisterCommentServiceServer(grpcServer, commentServer)

	// Enable reflection for development purposes
	if cfg.Server.Env == "development" {
		reflection.Register(grpcServer)
	}

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Received shutdown signal")
		grpcServer.GracefulStop()
	}()

	log.Printf("Starting gRPC server on port %d", cfg.Server.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
