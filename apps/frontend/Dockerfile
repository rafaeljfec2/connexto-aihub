# syntax=docker/dockerfile:1

ARG NODE_VERSION=24.22.1
ARG PNPM_VERSION=12.16.1

FROM node:${NODE_VERSION}-bookworm-slim AS base

ARG PNPM_VERSION

ENV PNPM_HOME=/pnpm
ENV PATH="${PNPM_HOME}:${PATH}"

RUN corepack enable \
  && corepack prepare "pnpm@${PNPM_VERSION}" --activate

FROM base AS deps

WORKDIR /app

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/frontend/package.json apps/frontend/package.json

RUN pnpm install --frozen-lockfile --allow-build esbuild --filter @connexto/frontend...

FROM deps AS builder

COPY apps/frontend apps/frontend

RUN pnpm --filter @connexto/frontend build

FROM nginx:1.27-alpine AS runtime

ENV AIHUB_BACKEND_UPSTREAM=http://backend:8080

RUN rm -f /etc/nginx/conf.d/default.conf

COPY --from=builder /app/apps/frontend/dist /usr/share/nginx/html
COPY apps/frontend/nginx.conf.template /etc/nginx/templates/default.conf.template

EXPOSE 80

HEALTHCHECK --interval=15s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:80/nginx-health >/dev/null || exit 1
