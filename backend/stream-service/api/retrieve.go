package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	grpcclient "github.com/clementus360/stream-service/grpc"
	"github.com/clementus360/stream-service/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type StreamResponse struct {
	ID          int32  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	StreamKey   string `json:"stream_key"`
	Resolution  string `json:"resolution"`
	Bitrate     string `json:"bitrate"`
	FPS         string `json:"fps"`
	Codec       string `json:"codec"`
	Protocol    string `json:"protocol"`
	Status      string `json:"status"`
	UserID      int32  `json:"user_id"`
}

func RetrieveStream(grpcClient *grpcclient.Client) http.HandlerFunc {
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
		var req proto.GetStreamRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Errorf("Invalid request format: %v", err)
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		// Ensure that only authorized users can get stream info (optional)
		// You can use a middleware to validate user authentication here

		// Call gRPC to get the stream info
		streamResponse, err := grpcClient.Client.GetStream(r.Context(), &req)
		if err != nil {
			// Parse the gRPC error to extract detailed error information
			if errStatus, ok := status.FromError(err); ok {
				errorMsg := errStatus.Message()

				// Return appropriate HTTP status based on gRPC status code
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
				}

				// Create a structured error response
				errorResponse := map[string]interface{}{
					"status":  "error",
					"code":    statusCode,
					"message": "Failed to retrieve stream",
					"details": errorMsg,
				}

				logger.Errorf("Failed to get stream info via gRPC: %v", err)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(errorResponse)
				return
			}

			// Fallback error handling if not a standard gRPC status error
			logger.Errorf("Failed to get stream info: %v", err)
			http.Error(w, "Failed to get stream info", http.StatusInternalServerError)
			return
		}

		fmt.Println(streamResponse)

		// Respond with the created stream as JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(streamResponse); err != nil {
			logger.Errorf("Failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}

		logger.Infof("Retrieved stream info: %v", streamResponse)
	}
}
