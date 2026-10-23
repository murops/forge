#!/usr/bin/env bash
# Installs the chart on a kind cluster and creates a repository in dryrun mode.
# Run from the repository root. Set SKIP_BUILD=1 to reuse images already loaded.
set -euo pipefail

cluster=${CLUSTER:-forge-e2e}
tag=${TAG:-e2e}
chart=charts/forge

if ! kind get clusters | grep -qx "$cluster"; then
  kind create cluster --name "$cluster" --wait 60s
fi
kubectl config use-context "kind-$cluster" >/dev/null

if [[ -z "${SKIP_BUILD:-}" ]]; then
  docker build -f backend/Dockerfile --target api -t "ghcr.io/murops/forge-api:$tag" .
  docker build -f backend/Dockerfile --target worker -t "ghcr.io/murops/forge-worker:$tag" .
  docker build -t "ghcr.io/murops/forge-frontend:$tag" frontend
fi
kind load docker-image --name "$cluster" \
  "ghcr.io/murops/forge-api:$tag" "ghcr.io/murops/forge-worker:$tag" "ghcr.io/murops/forge-frontend:$tag"

kubectl create namespace keycloak --dry-run=client -o yaml | kubectl apply -f -
kubectl -n keycloak create configmap forge-realm --from-file=deploy/keycloak/forge-realm.json \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f "$chart/ci/deps.yaml"
kubectl -n forge rollout status deploy/postgres --timeout=2m
kubectl -n keycloak rollout status deploy/keycloak --timeout=5m

helm upgrade --install forge "$chart" -n forge -f "$chart/ci/test-values.yaml" \
  --set image.tag="$tag" --wait --timeout 5m

kubectl -n keycloak port-forward svc/keycloak 18080:8080 >/dev/null &
kubectl -n forge port-forward svc/forge-api 18000:8000 >/dev/null &
trap 'kill $(jobs -p) 2>/dev/null' EXIT
for _ in $(seq 30); do
  curl -sf localhost:18000/readyz && curl -sf -o /dev/null localhost:18080/realms/forge && break
  sleep 1
done

token=$(curl -sf localhost:18080/realms/forge/protocol/openid-connect/token \
  -d grant_type=password -d client_id=forge-web -d client_secret=forge-web-local-secret \
  -d username=dev -d password=dev -d scope=openid | jq -r .access_token)
api() { curl -sf -H "Authorization: Bearer $token" "$@"; }

templates=$(api localhost:18000/api/v1/templates | jq '.templates | length')
echo "templates: $templates"
[[ "$templates" -gt 0 ]]

name="e2e-$(date +%s)"
id=$(api -X POST localhost:18000/api/v1/repositories -H 'Content-Type: application/json' \
  -d "{\"name\":\"$name\",\"template\":\"api-go\"}" | jq -r .id)
echo "created request $id for $name"

for _ in $(seq 30); do
  status=$(api "localhost:18000/api/v1/repositories/$id" | jq -r .status)
  echo "status: $status"
  case $status in
    succeeded) exit 0 ;;
    failed) exit 1 ;;
  esac
  sleep 2
done
echo "request did not finish in time" >&2
kubectl -n forge logs deploy/forge-worker --tail=50 >&2
exit 1
