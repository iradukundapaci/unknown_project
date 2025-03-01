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

func UpdateStream(grpcClient *grpcclient.Client) http.HandlerFunc {
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

		// Unmarshal the JSON into a UpdateStreamRequest message
		var req proto.UpdateStreamRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Errorf("Invalid request format: %v", err)
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		// Call gRPC to update the stream info
		streamResponse, err := grpcClient.Client.UpdateStream(r.Context(), &req)
		if err != nil {
			// Parse the gRPC error to extract validation errors
			if errStatus, ok := status.FromError(err); ok {
				// Check if it's an InvalidArgument error
				if errStatus.Code() == codes.InvalidArgument {
					errorMsg := errStatus.Message()
					// Create a structured error response
					errorResponse := map[string]interface{}{
						"status":  "error",
						"code":    http.StatusBadRequest,
						"message": "Validation failed",
						"details": errorMsg,
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(errorResponse)
					return
				} else if errStatus.Code() == codes.NotFound {
					errMsg := errStatus.Message()

					errorResponse := map[string]interface{}{
						"status":  "error",
						"code":    http.StatusNotFound,
						"message": "Stream not found",
						"details": errMsg,
					}

					logger.Errorf("Failed to update stream info: %v", err)

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(errorResponse)
					return
				}
			}
			// If not a validation error, return generic error
			logger.Errorf("Failed to update stream info: %v", err)
			http.Error(w, "Failed to update stream info", http.StatusInternalServerError)
			return
		}

		// Respond with the updated stream as JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // Changed from StatusFound to StatusOK
		if err := json.NewEncoder(w).Encode(streamResponse); err != nil {
			logger.Errorf("Failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}
