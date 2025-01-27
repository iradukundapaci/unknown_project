package main

import (
	"fmt"
	"github.com/clerkinc/clerk-sdk-go/clerk"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/clerkinc/clerk-sdk-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/Josy-coder/user-service/internal/clients"
	"github.com/Josy-coder/user-service/internal/config"
	"github.com/Josy-coder/user-service/internal/ports"
	"github.com/Josy-coder/user-service/internal/service"
	pb "github.com/Josy-coder/user-service/proto/user/v1"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Clerk client
	clerkClient, err := clerk.NewClient(cfg.ClerkSecretKey)
	if err != nil {
		log.Fatalf("Failed to create Clerk client: %v", err)
	}

	// Initialize database client
	dbClient, err := clients.NewDBServiceClient(cfg.Services.DBServiceURL)
	if err != nil {
		log.Fatalf("Failed to create database service client: %v", err)
	}
	defer dbClient.Close()

	// Initialize comment client
	commentClient, err := clients.NewCommentServiceClient(cfg.Services.CommentServiceURL)
	if err != nil {
		log.Fatalf("Failed to create comment service client: %v", err)
	}
	defer commentClient.Close()

	// Initialize stream client
	streamClient, err := clients.NewStreamServiceClient(cfg.Services.StreamServiceURL)
	if err != nil {
		log.Fatalf("Failed to create stream service client: %v", err)
	}
	defer streamClient.Close()

	// Initialize auth client
	authClient := clients.NewAuthClient(clerkClient)

	// Initialize metrics client
	metricsClient := clients.NewMetricsClient("user_service")

	// Initialize user service
	userService := service.NewUserService(dbClient, clerkClient, commentClient, streamClient)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	userServer := ports.NewGRPCServer(userService)
	pb.RegisterUserServiceServer(grpcServer, userServer)

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
