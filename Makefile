.PHONY: up down logs test lint generate token

SQLC_VERSION := 1.31.1

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f api worker

test:
	cd backend && go test -race ./...

lint:
	cd backend && go vet ./...

generate:
	docker run --rm -u $$(id -u):$$(id -g) -v $(CURDIR)/backend:/src -w /src sqlc/sqlc:$(SQLC_VERSION) generate

# access token for the local "dev" user, e.g. TOKEN=$(make -s token)
token:
	@curl -s http://localhost:8080/realms/forge/protocol/openid-connect/token \
		-d grant_type=password -d client_id=forge-web -d client_secret=forge-web-local-secret \
		-d username=dev -d password=dev -d scope=openid | jq -r .access_token
