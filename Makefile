.PHONY: fmt check test integration build

fmt:
	gofmt -w cmd internal

check:
	test -z "$$(gofmt -l cmd internal)"
	go vet ./...

test:
	go test -race -count=1 ./...

integration:
	go test -tags=integration -race -count=1 ./...

build:
	go build -o bin/erp ./cmd/erp
