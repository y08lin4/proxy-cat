# Stage 1: Build frontend
FROM node:22-alpine AS frontend-builder

RUN corepack enable && corepack prepare pnpm@11.5.3 --activate

WORKDIR /src/frontend

COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY frontend/ ./
RUN pnpm run build

# Stage 2: Build Go binary
FROM golang:1.22-alpine AS go-builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /proxy-cat ./cmd/proxy-cat

# Stage 3: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=go-builder /proxy-cat .
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist

ENV FRONTEND_DIR=/app/frontend/dist

EXPOSE 8080

ENTRYPOINT ["./proxy-cat", \
    "--headless", \
    "--no-system-proxy", \
    "--port", "8080", \
    "--mihomo-binary", "/usr/local/bin/mihomo"]
