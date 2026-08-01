# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.25.11-alpine3.24 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
ARG BUILD_BY=docker

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build -buildvcs=false -trimpath -tags=netgo \
    -ldflags="-s -w \
    -X github.com/justjundana/govman/internal/version.Version=${VERSION} \
    -X github.com/justjundana/govman/internal/version.Commit=${COMMIT} \
    -X github.com/justjundana/govman/internal/version.Date=${DATE} \
    -X github.com/justjundana/govman/internal/version.BuildBy=${BUILD_BY}" \
    -o /out/govman ./cmd/govman

FROM alpine:3.24.1

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

LABEL org.opencontainers.image.title="govman" \
      org.opencontainers.image.description="Cross-platform Go version manager" \
      org.opencontainers.image.url="https://github.com/justjundana/govman" \
      org.opencontainers.image.source="https://github.com/justjundana/govman" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.vendor="justjundana" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${DATE}" \
      org.opencontainers.image.revision="${COMMIT}"

RUN apk add --no-cache ca-certificates \
    && adduser -D -s /bin/sh govman \
    && mkdir -p /home/govman/.govman/bin \
        /home/govman/.govman/cache \
        /home/govman/.govman/versions \
        /home/govman/.govman/downloads \
    && chown -R govman:govman /home/govman/.govman \
    && chmod 0700 /home/govman/.govman \
    && chmod 0755 /home/govman/.govman/bin

COPY --from=builder /out/govman /usr/local/bin/govman

USER govman
WORKDIR /home/govman

ENV HOME=/home/govman \
    PATH=/home/govman/.govman/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/usr/local/bin/govman", "--version"]

ENTRYPOINT ["/usr/local/bin/govman"]
CMD ["--help"]
