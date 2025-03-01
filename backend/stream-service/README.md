# Stream Service

This is the Stream Service, a Go-based microservice designed to handle stream creation and other stream-related actions. The service exposes REST endpoints for client interactions and uses Protocol Buffers (gRPC) for communication with other microservices, such as the database service.

## Features
- **REST API**: Handles client requests for stream-related actions.
- **gRPC Communication**: Interacts with other services (e.g., database service) via Protocol Buffers.
- **Stream Management**: Supports operations like stream creation, deletion, and updates.

## Getting Started

### Prerequisites
- Go (1.22+ recommended)
- Protocol Buffers Compiler (`protoc`)
- A running instance of the database service

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/clementus360/stream-service.git
   cd stream-service
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Compile Protocol Buffers (if needed):
   ```bash
   protoc --go_out=. --go-grpc_out=. proto/*.proto
   ```