.PHONY: all build run test clean fmt

all: build

build:
	go build -o bin/tsuzuri ./cmd/tsuzuri

run:
	go run ./cmd/tsuzuri

test:
	go test -v ./...

fmt:
	gofmt -w .

clean:
	rm -rf bin/
