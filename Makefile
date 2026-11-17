.PHONY: init up down check lint generate test test-integration test-restore e2e

init:
	python3 scripts/init-env.py
up:
	docker compose up --build --wait
down:
	docker compose down
generate:
	cd backend && go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
	cd frontend && pnpm generate:api
lint:
	cd backend && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run ./...
test:
	cd backend && go test ./...
test-integration:
	docker compose -p etheyl-tests -f compose.test.yaml up -d --wait
	endpoint=$$(docker compose -p etheyl-tests -f compose.test.yaml port test-rustfs 9000) && \
		cd backend && TEST_DATABASE_URL=postgres://etheyl_test:etheyl_test@127.0.0.1:5433/etheyl_test?sslmode=disable TEST_S3_ENDPOINT="$$endpoint" go test -race -count=1 ./...
check: test lint
	cd frontend && pnpm check && pnpm build
e2e:
	cd frontend && pnpm test:e2e

test-restore:
	python3 scripts/restore-drill.py
