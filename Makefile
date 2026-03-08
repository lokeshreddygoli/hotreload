.PHONY: all build install test test-race demo clean deps help

BINARY    := hotreload
BUILD_DIR := ./bin

all: deps build

## deps: tidy and download dependencies
deps:
	go mod tidy
	go mod download

## build: compile the hotreload binary
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) .
	@echo "Built: $(BUILD_DIR)/$(BINARY)"

## install: install to GOPATH/bin
install:
	go install .

## test: run all tests
test:
	go test ./... -v -count=1

## test-race: run tests with the race detector
test-race:
	go test -race ./... -count=1

## demo: build and run against testserver — visit http://localhost:8080
demo: build
	@echo ""
	@echo "  Server → http://localhost:8080"
	@echo "  Edit testserver/main.go and save to trigger a reload"
	@echo ""
	$(BUILD_DIR)/$(BINARY) \
		--root ./testserver \
		--build "go build -o ./bin/testserver ./testserver" \
		--exec  "./bin/testserver"

## clean: remove build artifacts
clean:
	rm -rf $(BUILD_DIR)

## help: list available targets
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
