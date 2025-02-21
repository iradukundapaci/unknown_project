# Use the official Golang image for building
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

# Set environment variables for Go
ENV GO111MODULE=on \
    CGO_ENABLED=0

# Set the working directory
WORKDIR /app

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the stream service with proper architecture targeting
ARG TARGETOS TARGETARCH
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o stream-service ./main.go

# Use a minimal image for the final build
FROM --platform=$TARGETPLATFORM alpine:3.18

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Set environment variables
ENV GIN_MODE=release

# Set the working directory
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/stream-service .

# Expose the gRPC server port and REST API port
EXPOSE 50051 8080

# Run the binary
CMD ["./stream-service"]