.PHONY: up down logs test tf-test lint generate token

SQLC_VERSION := 1.31.1

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f api worker

test:
	cd backend && go test -race ./...

tf-test:
	@for dir in infra/modules/* infra/stacks/*; do \
		echo "== $$dir"; \
		terraform -chdir=$$dir init -backend=false -input=false >/dev/null && terraform -chdir=$$dir test || exit 1; \
	done

lint:
	cd backend && go vet ./...
	terraform fmt -check -recursive infra

generate:
	docker run --rm -u $$(id -u):$$(id -g) -v $(CURDIR)/backend:/src -w /src sqlc/sqlc:$(SQLC_VERSION) generate

# access token for the local "dev" user, e.g. TOKEN=$(make -s token)
token:
	@curl -s http://keycloak.localhost:8080/realms/forge/protocol/openid-connect/token \
		-d grant_type=password -d client_id=forge-web -d client_secret=forge-web-local-secret \
		-d username=dev -d password=dev -d scope=openid | jq -r .access_token
