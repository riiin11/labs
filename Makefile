.PHONY: generate migrate migrate-down run lint test test-integration

generate:
	go tool oapi-codegen -generate types,chi-server -package api \
		-o internal/generated/api.gen.go contracts/openapi/trip-service.openapi.yaml
	go generate ./...

migrate:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

run:
	go run ./cmd/service

lint:
	golangci-lint run

test:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

test-integration:
	go test -race -tags=integration ./internal/repository/...