# ─────────────────────────────────────────────────────────────────────────────
# ExchangeOS Migrator — applies CRDB migrations + loads seeds.
# Bundles SQL files into the image so it's self-contained at runtime.
# ─────────────────────────────────────────────────────────────────────────────

ARG GO_VERSION=1.26.11
ARG DISTROLESS_TAG=nonroot

# ─── Stage 1: builder ────────────────────────────────────────────────────────
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /src

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
      -o /out/exchangeos-migrator \
      ./cmd/migrator

# ─── Stage 2: runtime ────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:${DISTROLESS_TAG}

WORKDIR /app

COPY --from=builder /out/exchangeos-migrator /app/exchangeos-migrator
# Migrations + seeds bundled so the container is self-contained.
COPY migrations /app/migrations
COPY seeds      /app/seeds

USER nonroot:nonroot

ENV EXCHANGEOS_MIGRATIONS_SOURCE=file:///app/migrations \
    EXCHANGEOS_SEEDS_DIR=/app/seeds

ENTRYPOINT ["/app/exchangeos-migrator"]
CMD ["up"]
