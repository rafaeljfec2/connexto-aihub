# syntax=docker/dockerfile:1

ARG GO_VERSION=1.25

FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /src

COPY apps/backend/go.mod apps/backend/go.sum ./
RUN go mod download

COPY apps/backend/ ./

# Satisfy //go:embed all:dist when the frontend is served by its own container.
RUN mkdir -p internal/static/dist && : > internal/static/dist/.placeholder

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ai-hub ./cmd/ai-hub

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates curl tzdata \
  && rm -rf /var/lib/apt/lists/* \
  && groupadd --system --gid 1000 aihub \
  && useradd --system --uid 1000 --gid aihub --home-dir /var/lib/aihub --create-home aihub \
  && mkdir -p /var/lib/aihub \
  && chown -R aihub:aihub /var/lib/aihub

COPY --from=builder /out/ai-hub /usr/local/bin/ai-hub
COPY --chmod=755 apps/backend/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

USER aihub
WORKDIR /var/lib/aihub

ENV AIHUB_SERVER_HOST=0.0.0.0 \
  AIHUB_SERVER_PORT=8080 \
  AIHUB_DATA_DIR=/var/lib/aihub \
  AIHUB_SERVE_EMBEDDED_UI=false

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -fsS "http://127.0.0.1:${AIHUB_SERVER_PORT:-8080}/healthz" || exit 1

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

