package api

import (
	"encoding/json"
	"io"
	"net/http"

	grpcclient "github.com/clementus360/stream-service/grpc"
	"github.com/clementus360/stream-service/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteStream(grpcClient *grpcclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := logrus.New()

		// Read and parse the request body with a limit to prevent large payload attacks
		body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
		if err != nil {
			logger.Errorf("Failed to read request body: %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Unmarshal the JSON into a GetStreamRequest message
		var req proto.DeleteStreamRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Errorf("Invalid request format: %v", err)
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		// Ensure that only authorized users can delete streams (optional)
		// You can use a middleware to validate user authentication here

		// Call gRPC to get the stream info
		streamResponse, err := grpcClient.Client.DeleteStream(r.Context(), &req)
		if err != nil {
			// Parse the gRPC error to extract detailed error information
			if errStatus, ok := status.FromError(err); ok {
				errorMsg := errStatus.Message()

				// Map gRPC status codes to HTTP status codes
				statusCode := http.StatusInternalServerError
				switch errStatus.Code() {
				case codes.InvalidArgument:
					statusCode = http.StatusBadRequest
				case codes.NotFound:
					statusCode = http.StatusNotFound
				case codes.PermissionDenied:
					statusCode = http.StatusForbidden
				case codes.Unauthenticated:
					statusCode = http.StatusUnauthorized
				case codes.FailedPrecondition:
					statusCode = http.StatusPreconditionFailed
				}

				// Create a structured error response
				errorResponse := map[string]interface{}{
					"status":  "error",
					"code":    statusCode,
					"message": "Failed to delete stream",
					"details": errorMsg,
				}

				logger.Errorf("Failed to delete stream via grpc: %v", err)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(errorResponse)
				return
			}

			// Fallback error handling
			logger.Errorf("Failed to delete stream: %v", err)
			http.Error(w, "Failed to delete stream", http.StatusInternalServerError)
			return
		}

		// Respond with the created stream as JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		successResponse := map[string]interface{}{
			"status":  "success",
			"message": "Stream deleted successfully",
		}
		if err := json.NewEncoder(w).Encode(successResponse); err != nil {
			logger.Errorf("Failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}

		logger.Infof("Deleted stream: %v", streamResponse)

	}
}
