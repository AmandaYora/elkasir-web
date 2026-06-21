# syntax=docker/dockerfile:1
#
# ONE-CONTAINER image: the SPA (apps/web) is built statically and EMBEDDED into the
# Go binary (apps/api), so a single process serves both the web SPA (root) and the
# API (/api/v1). MySQL is NOT here — it runs at host/OS level (see docker-compose.yml).
#
# Build context = repo root:  docker build -t elkasir-app:latest -f infra/docker/Dockerfile .

# ── Stage 1: build the static SPA (Node) ──
FROM node:22-alpine AS web
WORKDIR /repo
# Manifests first (npm cache layer). apps/api is a Go module, not an npm workspace.
COPY package.json package-lock.json ./
COPY apps/web/package.json ./apps/web/package.json
COPY packages/api-contract/package.json ./packages/api-contract/package.json
COPY packages/shared/package.json ./packages/shared/package.json
RUN npm ci
COPY apps/web ./apps/web
COPY packages ./packages
RUN npm run build -w @elkasir/web
# Output: /repo/apps/web/dist (index.html + assets)

# ── Stage 2: build the Go binary + embed the SPA ──
FROM golang:1.26 AS api
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum* ./
RUN go mod download
COPY apps/api ./
# Copy the built SPA into the embed dir; go:embed bakes it into the binary.
COPY --from=web /repo/apps/web/dist ./internal/webui/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ── Stage 3: distroless runtime (small, nonroot) ──
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=api /out/api /app/api
# Migrations are also embedded in the binary (go:embed) — `api migrate up` works from
# this image. The copy below keeps them on disk too for external tooling/inspection.
COPY --from=api /src/db/migrations /app/db/migrations
EXPOSE 8081
# Self-check via the binary's healthcheck subcommand (no shell/curl in distroless).
HEALTHCHECK --interval=30s --timeout=3s --start-period=20s --retries=3 \
  CMD ["/app/api", "healthcheck"]
USER nonroot:nonroot
ENTRYPOINT ["/app/api"]
