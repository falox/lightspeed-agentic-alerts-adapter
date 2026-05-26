BINARY_NAME := lightspeed-agentic-alerts-adapter
IMAGE_REPO ?= quay.io/openshift-lightspeed/lightspeed-agentic-alerts-adapter
IMAGE_TAG ?= latest

.PHONY: build test lint image clean

build:
	go build -o bin/$(BINARY_NAME) ./cmd/adapter/

test:
	go test ./... -count=1

lint:
	golangci-lint run ./...

image:
	podman build -t $(IMAGE_REPO):$(IMAGE_TAG) -f Containerfile .

clean:
	rm -rf bin/
