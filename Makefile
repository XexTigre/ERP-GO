.PHONY: fmt check test integration build db-up preview

COMPOSE = docker compose --project-directory . --env-file .env -f deploy/compose.yaml

fmt:
	gofmt -w cmd internal migrations test

check:
	test -z "$$(gofmt -l cmd internal migrations test)"
	go vet ./...

test:
	go test -race -count=1 ./...

integration:
	go test -tags=integration -race -count=1 ./...

build:
	go build -o bin/erp ./cmd/erp
	go build -o bin/erpctl ./cmd/erpctl

db-up:
	$(COMPOSE) up -d db

preview:
	python3 -m http.server 8090 --directory internal/web/static
