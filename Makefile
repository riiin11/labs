include .env
export

.PHONY: generate migrate migrate-down run lint test test-integration

generate:
	go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen \
		-generate types,chi-server -package api \
		-o internal/generated/api/api.gen.go contracts/openapi/trip-service.openapi.yaml
	go generate ./...

migrate:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

run:
	go run ./cmd/trip-service

lint:
	golangci-lint run

test:
	go test -coverprofile coverage.out ./...
	go tool cover -func coverage.out | tail -1

test-integration:
	go test -tags=integration ./internal/repository/...