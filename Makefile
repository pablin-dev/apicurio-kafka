.PHONY: build fmt lint test

build:
	go build -o apireg cmd/main.go

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

test:
	ginkgo -v ./cmd