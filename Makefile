.PHONY: build test test-integration lint cover docker helm-lint run

build:
	go build ./...

test:
	go test ./...

test-integration:
	go test -tags=integration ./tests/integration/...

lint:
	gofmt -l .
	go vet ./...

cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

docker:
	docker build --build-arg CMD=server -t pos-production-service:server .
	docker build --build-arg CMD=worker -t pos-production-service:worker .

helm-lint:
	helm lint charts/production-service

run:
	go run ./cmd/server & \
	go run ./cmd/worker
