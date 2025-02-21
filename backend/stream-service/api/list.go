package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	grpcclient "github.com/clementus360/stream-service/grpc"
	"github.com/clementus360/stream-service/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListStream(grpcClient *grpcclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := logrus.New()

		// Parse the query parameters
		query := r.URL.Query()

		// Set default values for pagination
		pageSize := int32(10)
		pageNumber := int32(1)

		// Parse the page query parameters
		if p := query.Get("page"); p != "" {
			if parsedPage, err := strconv.Atoi(p); err == nil && parsedPage > 0 {
				pageNumber = int32(parsedPage)
			} else {
				logger.Warnf("Invalid page parameter: %v", p)
			}
		}

		// Parse the page_size query parameter
		if ps := query.Get("page_size"); ps != "" {
			if parsedPageSize, err := strconv.Atoi(ps); err == nil && parsedPageSize > 0 {
				pageSize = int32(parsedPageSize)
			} else {
				logger.Warnf("Invalid page_size parameter: %v", ps)
			}
		}

		// Initialize filter
		filter := &proto.StreamFilter{}

		// Parse text filters
		if title := query.Get("title_contains"); title != "" {
			filter.TitleContains = title
		}
		if desc := query.Get("description_contains"); desc != "" {
			filter.DescriptionContains = desc
		}
		if codec := query.Get("codec"); codec != "" {
			filter.Codec = codec
		}
		if protocol := query.Get("protocol"); protocol != "" {
			filter.Protocol = protocol
		}

		// Parse user_id
		if id := query.Get("user_id"); id != "" {
			if parsedID, err := strconv.Atoi(id); err == nil {
				filter.UserId = int32(parsedID)
			} else {
				logger.Warnf("Invalid user_id parameter: %v", id)
			}
		}

		// Parse view count filters
		if minViews := query.Get("min_view_count"); minViews != "" {
			if parsedViews, err := strconv.Atoi(minViews); err == nil {
				filter.MinViewCount = int32(parsedViews)
			}
		}
		if maxViews := query.Get("max_view_count"); maxViews != "" {
			if parsedViews, err := strconv.Atoi(maxViews); err == nil {
				filter.MaxViewCount = int32(parsedViews)
			}
		}

		// Parse time filters
		if startTime := query.Get("start_time"); startTime != "" {
			filter.StartTime = startTime
		}
		if endTime := query.Get("end_time"); endTime != "" {
			filter.EndTime = endTime
		}
		if endTimeAfter := query.Get("end_time_after"); endTimeAfter != "" {
			filter.EndTimeAfter = endTimeAfter
		}
		if endTimeBefore := query.Get("end_time_before"); endTimeBefore != "" {
			filter.EndTimeBefore = endTimeBefore
		}

		// Parse status filters (can be multiple)
		if statuses := query["status"]; len(statuses) > 0 {
			filter.Status = statuses
		}

		// Create the request
		req := &proto.ListStreamsRequest{
			PageSize:   pageSize,
			PageNumber: pageNumber,
			Filter:     filter,
			SortBy:     query.Get("sort_by"),
			Ascending:  query.Get("ascending") == "true",
		}

		// Call gRPC to list streams
		streamResponse, err := grpcClient.Client.ListStreams(r.Context(), req)
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
				case codes.ResourceExhausted:
					statusCode = http.StatusTooManyRequests
				}

				// Create a structured error response
				errorResponse := map[string]interface{}{
					"status":  "error",
					"code":    statusCode,
					"message": "Failed to list streams",
					"details": errorMsg,
				}

				// For invalid arguments, try to parse specific validation errors
				if errStatus.Code() == codes.InvalidArgument {
					// Try to extract specific validation issues from the error message
					validationIssues := parseValidationErrors(errorMsg)
					if len(validationIssues) > 0 {
						errorResponse["validation_errors"] = validationIssues
					}
				}

				logger.Errorf("Failed to list streams via grpc: %v", err)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(errorResponse)
				return
			}

			// Fallback error handling
			logger.Errorf("Failed to list streams: %v", err)
			http.Error(w, "Failed to list streams", http.StatusInternalServerError)
			return
		}

		// Respond with the list of streams as JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(streamResponse); err != nil {
			logger.Errorf("Failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}

		logger.Infof("Listed streams: %v", streamResponse)
	}
}

// Helper function to parse validation errors from error message
func parseValidationErrors(errorMsg string) []string {
	// Simple parsing: split by commas and periods
	var errors []string

	// Handle common formats like "Field is required, Another field is invalid"
	for _, part := range strings.Split(errorMsg, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			errors = append(errors, part)
		}
	}

	// If we didn't split anything but there are periods
	if len(errors) == 0 && strings.Contains(errorMsg, ". ") {
		for _, part := range strings.Split(errorMsg, ". ") {
			part = strings.TrimSpace(part)
			if part != "" {
				errors = append(errors, part)
			}
		}
	}

	// If we couldn't parse it, just return the whole message
	if len(errors) == 0 && errorMsg != "" {
		errors = append(errors, errorMsg)
	}

	return errors
}
