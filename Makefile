APP := elevator-iot-diagnosis-service
IMAGE ?= elevator-iot-diagnosis-service:latest

.PHONY: build test race vet fmt run docker-build clean

build:
	CGO_ENABLED=0 go build -trimpath -o bin/$(APP) .

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

run:
	go run .

docker-build:
	docker build -t $(IMAGE) .

clean:
	rm -rf bin data
