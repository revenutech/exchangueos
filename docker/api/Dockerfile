# ─────────────────────────────────────────────────────────────────────────────
# ExchangeOS API — distroless multi-stage, multi-arch (linux/amd64 + linux/arm64)
# Target image < 50 MB (FX-DOC-001).
# ─────────────────────────────────────────────────────────────────────────────

ARG GO_VERSION=1.25.1
ARG DISTROLESS_TAG=nonroot

# ─── Stage 1: builder ────────────────────────────────────────────────────────
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /src

# Cache deps first
COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE

ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath \
      -ldflags="-s -w \
        -X main.version=${VERSION} \
        -X main.commit=${COMMIT} \
        -X main.buildDate=${BUILD_DATE}" \
      -o /out/exchangeos-api \
      ./cmd/api && \
    go build -trimpath -ldflags="-s -w" \
      -o /out/healthcheck \
      ./cmd/healthcheck

# ─── Stage 2: runtime ────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:${DISTROLESS_TAG}

WORKDIR /app

COPY --from=builder /out/exchangeos-api /app/exchangeos-api
COPY --from=builder /out/healthcheck /app/healthcheck

# Distroless nonroot UID/GID = 65532
USER nonroot:nonroot

EXPOSE 8084 9084

ENV EXCHANGEOS_ENV=dev \
    EXCHANGEOS_HTTP_PORT=8084 \
    EXCHANGEOS_GRPC_PORT=9084

ENTRYPOINT ["/app/exchangeos-api"]
