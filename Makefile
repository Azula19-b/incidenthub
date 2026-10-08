.PHONY: run build test vet check clean

run:
	go run ./cmd/server

build:
	mkdir -p bin
	go build -o bin/incidenthub ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

check: test vet build

clean:
	rm -rf bin coverage.out
