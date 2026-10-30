# build context is the repository root
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/forge ./cmd/forge

FROM hashicorp/terraform:1.17.0 AS terraform
ENV TF_PLUGIN_CACHE_DIR=/opt/forge/terraform/plugins
COPY infra /opt/forge/infra
RUN mkdir -p $TF_PLUGIN_CACHE_DIR \
    && for stack in /opt/forge/infra/stacks/*/; do \
         terraform -chdir=$stack init -backend=false -input=false && rm -rf $stack/.terraform; \
       done

FROM gcr.io/distroless/static-debian13:nonroot AS api
USER nonroot:nonroot
COPY --from=build /out/forge /usr/local/bin/forge
ENTRYPOINT ["forge"]

FROM api AS worker
COPY --from=terraform /bin/terraform /usr/local/bin/terraform
COPY --from=terraform /opt/forge /opt/forge
COPY deploy/terraform/terraformrc /opt/forge/terraformrc
ENV TF_CLI_CONFIG_FILE=/opt/forge/terraformrc \
    TERRAFORM_DIR=/opt/forge/infra
