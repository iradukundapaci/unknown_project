package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/clementus360/stream-service/api"
	"github.com/clementus360/stream-service/config"
	grpcclient "github.com/clementus360/stream-service/grpc"
	"github.com/clementus360/stream-service/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Define the logger for error and eventt logging
	logger := logrus.New()

	// load environment variables
	config.LoadEnv()

	// Get the port number from environment variables and set the fallback port
	PORT := os.Getenv("PORT")
	if PORT == "" {
		PORT = "8080"
	}

	grpcPORT := os.Getenv("gRPC_PORT")

	// start a grpc client with context to handle grpc connections
	ctx := context.Background()
	grpcClient, err := grpcclient.NewClient(ctx)
	if err != nil {
		logger.Fatalf("Failed to initialize gRPC client: %v", err)
	}
	logger.Info("grpc client initialized successfully")
	defer grpcClient.Close()

	// define route handlers
	router := http.NewServeMux()
	router.HandleFunc("POST /v1/api/stream", api.CreateStream(grpcClient))
	router.HandleFunc("GET /v1/api/stream", api.RetrieveStream(grpcClient))
	router.HandleFunc("PATCH /v1/api/stream", api.UpdateStream(grpcClient))
	router.HandleFunc("DELETE /v1/api/stream", api.DeleteStream(grpcClient))
	router.HandleFunc("GET /v1/api/streams", api.ListStream(grpcClient))

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPORT))
	if err != nil {
		logger.Fatalf("Failed to listen: %v", err)
	}

	var opts []grpc.ServerOption

	// create a gRPC server instance
	grpcServer := grpc.NewServer(opts...)

	streamService := &grpcclient.StreamServiceServer{
		GrpcClient: *grpcClient,
	}
	reflection.Register(grpcServer)

	proto.RegisterStreamServiceServer(grpcServer, streamService)

	// Handle graceful shutdown
	go func() {
		logger.Infof("gRPC server is listening on port %s", grpcPORT)
		if err := grpcServer.Serve(listener); err != nil {
			logger.Fatalf("Failed to serve: %v", err)
		}
	}()

	// define the server before starting
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", PORT),
		Handler: router,
	}

	// start the server inside a goroutine
	go func() {
		logger.Info(fmt.Sprintf("Server running at PORT: %v", PORT))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	logger.Info("Shutting down server...")
	if err := server.Shutdown(context.Background()); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	grpcServer.GracefulStop()
	logger.Info("Server exiting")
}
