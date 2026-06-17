# syntax=docker/dockerfile:1.7

# ===== Stage 1: frontend builder =====
FROM node:22-alpine AS frontend-builder
WORKDIR /app

RUN corepack enable && corepack prepare pnpm@10.32.1 --activate

# 先复制 workspace 配置以利用缓存
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml* ./
COPY admin/package.json ./admin/
COPY visitor-sdk/package.json ./visitor-sdk/

RUN pnpm install --frozen-lockfile

# 复制源码并构建
COPY admin/ ./admin/
COPY visitor-sdk/ ./visitor-sdk/
RUN pnpm --filter @marketing-monitor/admin build && \
    pnpm --filter @marketing-monitor/visitor-sdk build

# ===== Stage 2: Go builder =====
FROM golang:1.25-alpine AS go-builder
WORKDIR /app

RUN apk add --no-cache git ca-certificates

# 先复制 go.mod 利用缓存
COPY server/go.mod server/go.sum* ./server/
RUN cd server && go mod download

# 复制源码
COPY server/ ./server/

# 复制前端构建产物到 embed 位置
COPY --from=frontend-builder /app/admin/dist ./server/cmd/server/embedded/admin
COPY --from=frontend-builder /app/visitor-sdk/dist ./server/cmd/server/embedded/sdk
COPY landing/ ./server/cmd/server/embedded/landing

# 构建 release 二进制
RUN cd server && \
    CGO_ENABLED=0 GOOS=linux go build \
      -ldflags="-s -w -X main.version=docker -X main.commit=${GIT_COMMIT:-unknown}" \
      -tags release \
      -o /out/server \
      ./cmd/server

# ===== Stage 3: runtime =====
FROM alpine:3.19 AS runtime
RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=go-builder /out/server /app/server

USER app
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=2s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/server"]
