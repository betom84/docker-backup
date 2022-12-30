BUILD_DIR=./build
TARGET=$(BUILD_DIR)/docker-backup
SRC=$(shell find . -name "*.go" -not -path "*/vendor/*")

.PHONY: test clean pre-build build docker

default: all

all: clean test fmt

fmt: 
	@test -z $(shell gofmt -l $(SRC)) || (gofmt -d $(SRC); exit 1)

test: pre-build
	go test -v ./...

clean:
	@if [ -d $(BUILD_DIR) ]; then rm -rf $(BUILD_DIR); fi;
	@if [ -d "./vendor" ]; then rm -rf "./vendor"; fi;
	go clean

pre-build:
	@test -d $(BUILD_DIR) || mkdir -p $(BUILD_DIR)
	go mod tidy
	go mod vendor

build: clean pre-build
	go build -v -o $(TARGET)

docker: clean pre-build
	@test ! -z $(DOCKER_REGISTRY) || (echo "DOCKER_REGISTRY not set"; exit 1)
	docker build -f Dockerfile -t docker-backup:latest .
	docker image tag docker-backup:latest $(DOCKER_REGISTRY)/docker-backup:latest
	docker image push $(DOCKER_REGISTRY)/docker-backup