# ─────────────────────────────────────────────────────────────────────────────
# ExchangeOS Worker — outbox dispatch loop + (future) Kafka consumers.
# Built with `-tags kafka` so the publisher uses franz-go (not the noop default).
# Distroless multi-stage, multi-arch — same shape as the api image.
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

# Build with kafka tag: selects pkg/outbox/kafka (franz-go) publisher.
# Without the tag, the publisher is a noop that only logs — useful for tests.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -tags kafka \
      -ldflags="-s -w \
        -X main.version=${VERSION} \
        -X main.commit=${COMMIT} \
        -X main.buildDate=${BUILD_DATE}" \
      -o /out/exchangeos-worker \
      ./cmd/worker

# ─── Stage 2: runtime ────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:${DISTROLESS_TAG}

WORKDIR /app

COPY --from=builder /out/exchangeos-worker /app/exchangeos-worker

USER nonroot:nonroot

# No EXPOSE — worker has no public surface. Future: prometheus :9090 for metrics.
ENV EXCHANGEOS_ENV=dev

ENTRYPOINT ["/app/exchangeos-worker"]
