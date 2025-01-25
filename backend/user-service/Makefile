# Build configuration
BINARY_NAME=comment-service
GO_BUILD_ENV=CGO_ENABLED=0 GOOS=linux GOARCH=amd64
DOCKER_REGISTRY=your-registry
VERSION?=latest

# Protocol buffer configuration
PROTO_DIR=proto
PROTO_OUT=proto/comment/v1

.PHONY: proto
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
	$(PROTO_DIR)/comment/v1/*.proto

.PHONY: build
build:
	${GO_BUILD_ENV} go build -o bin/${BINARY_NAME} cmd/server/main.go

.PHONY: run
run:
	go run cmd/server/main.go

.PHONY: test
test:
	go test -v ./... -cover

.PHONY: mock
mock:
	mockgen -source=internal/domain/comment.go -destination=internal/mocks/comment_mock.go

.PHONY: docker
docker:
	docker build -t ${DOCKER_REGISTRY}/${BINARY_NAME}:${VERSION} .

.PHONY: docker-run
docker-run:
	docker run -p 9091:9091 -p 50053:50053 ${DOCKER_REGISTRY}/${BINARY_NAME}:${VERSION}

.PHONY: clean
clean:
	go clean
	rm -f bin/${BINARY_NAME}

.PHONY: lint
lint:
	golangci-lint run

.PHONY: deps
deps:
	go mod tidy
	go mod verify