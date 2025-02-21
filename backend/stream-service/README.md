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

### Configuration

Set environment variables to configure the service. Example:

```bash
export STREAM_SERVICE_PORT=8080
export DATABASE_SERVICE_ADDRESS="localhost:50051"
export LOG_LEVEL="info"
```

You can also use a `.env` file to manage these variables.

### Running the Service

Start the service with:
```bash
go run main.go
```

### API Endpoints

#### Base URL
```
http://localhost:8080
```

#### Endpoints

| Method | Endpoint         | Description               |
|--------|------------------|---------------------------|
| POST   | `/stream`       | Create a new stream       |
| GET    | `/stream/:id`   | Retrieve a stream by ID   |
| PUT    | `/stream/:id`   | Update stream details     |
| DELETE | `/stream/:id`   | Delete a stream           |

### gRPC Functions

The service defines several gRPC methods for communication with the database and other microservices. Refer to the `proto` files for more details.

### Development

#### Running Tests

To run the test suite:
```bash
go test ./...
```

#### Code Formatting

Format your code before committing:
```bash
go fmt ./...
```

### Contributing

Contributions are welcome! Please submit a pull request or open an issue to discuss your ideas.

### License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
