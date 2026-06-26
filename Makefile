# Makefile for terraform-provider-virtualbox
BINARY_NAME=terraform-provider-virtualbox
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

default: build

.PHONY: build
build:
	go build -o $(BINARY_NAME) .

.PHONY: install
install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/bryanbelanger/virtualbox/0.1.0/$(GOOS)_$(GOARCH)/
	cp $(BINARY_NAME) ~/.terraform.d/plugins/registry.terraform.io/bryanbelanger/virtualbox/0.1.0/$(GOOS)_$(GOARCH)/

.PHONY: test
test:
	go test -v ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint: vet
	staticcheck ./...

.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: tidy
tidy:
	go mod tidy
	go mod verify

.PHONY: build-all
build-all: build test vet

.PHONY: dev-install
dev-install: tidy build test
	@echo "Provider built successfully. To install locally, run 'make install'"