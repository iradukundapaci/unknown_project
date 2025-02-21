package api

import (
	"encoding/json"
	"io"
	"net/http"

	grpcclient "github.com/clementus360/stream-service/grpc"
	"github.com/clementus360/stream-service/proto"
	"github.com/clementus360/stream-service/utils"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CreateStream(grpcClient *grpcclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := logrus.New()

		// Read and parse the request body with a limit to prevent large payload attacks
		body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // Limit body size to 10MB
		if err != nil {
			logger.Errorf("Failed to read request body: %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Unmarshal the JSON into a CreateStreamRequest message
		var req proto.CreateStreamRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Errorf("Invalid request format: %v", err)
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		// Generate stream key only if not provided
		if req.StreamKey == "" {
			req.StreamKey = utils.GenerateStreamKey()
		} else {
			// Optionally validate the stream key format
			if len(req.StreamKey) < 10 {
				http.Error(w, "Invalid stream key", http.StatusBadRequest)
				return
			}
		}

		// Ensure that only authorized users can create streams (optional)
		// You can use a middleware to validate user authentication here

		// Call gRPC to create the stream
		streamResponse, err := grpcClient.Client.CreateStream(r.Context(), &req)
		if err != nil {
			// Parse the gRPC error to extract detailed error information
			if errStatus, ok := status.FromError(err); ok {
				errorMsg := errStatus.Message()

				// Map gRPC status codes to HTTP status codes
				statusCode := http.StatusInternalServerError
				switch errStatus.Code() {
				case codes.InvalidArgument:
					statusCode = http.StatusBadRequest
				case codes.AlreadyExists:
					statusCode = http.StatusConflict
				case codes.PermissionDenied:
					statusCode = http.StatusForbidden
				case codes.Unauthenticated:
					statusCode = http.StatusUnauthorized
				case codes.ResourceExhausted:
					statusCode = http.StatusTooManyRequests
				}

				// Create a structured error response
				errorResponse := map[string]interface{}{
					"status":  "error",
					"code":    statusCode,
					"message": "Failed to create stream",
					"details": errorMsg,
				}

				logger.Errorf("Failed to create stream via grpc: %v", errorMsg)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(errorResponse)
				return
			}

			// Fallback error handling
			logger.Errorf("Failed to create stream via grpc: %v", err)
			http.Error(w, "Failed to create stream", http.StatusInternalServerError)
			return
		}

		// Respond with the created stream as JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(streamResponse); err != nil {
			logger.Errorf("Failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}

		logger.Infof("Created stream: %v", streamResponse)
	}
}
