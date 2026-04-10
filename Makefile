# Binary name
BINARY_NAME=code-gehirn

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOVET=$(GOCMD) vet
GOMOD=$(GOCMD) mod

# Main entry point
MAIN_FILE=main.go

.PHONY: all build clean test run fmt vet tidy install help

all: build

build:
	@echo "Building..."
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_FILE)

run: build
	./$(BINARY_NAME)

test:
	$(GOTEST) -v ./...

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

fmt:
	$(GOCMD) fmt ./...

vet:
	$(GOVET) ./...

tidy:
	$(GOMOD) tidy

install: build
	mkdir -p $(HOME)/bin
	cp $(BINARY_NAME) $(HOME)/bin/$(BINARY_NAME)
	mkdir -p $(HOME)/.config/code-gehirn
	test -f $(HOME)/.config/code-gehirn/config.yaml || cp config.yaml.example $(HOME)/.config/code-gehirn/config.yaml

help:
	@echo "Usage:"
	@echo "  make build    - Build the binary"
	@echo "  make run      - Build and run the binary"
	@echo "  make test     - Run tests"
	@echo "  make clean    - Remove binary"
	@echo "  make fmt      - Format code"
	@echo "  make vet      - Run go vet"
	@echo "  make tidy     - Tidy go modules"
	@echo "  make install  - Install binary to ~/bin and create ~/.config/code-gehirn/config.yaml"
