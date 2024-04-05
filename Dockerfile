# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.22 as builder

ARG GOARCH=''

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

COPY hack hack

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    go mod download

# Copy the go source
COPY main.go main.go
COPY metrics/ metrics/
# Needed for version extraction by go build
COPY .git/ .git/

ARG TARGETOS
ARG TARGETARCH

# Build
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on go build -ldflags="-s -w" -a -o prometheus-exporter main.go

FROM debian:bullseye-slim
WORKDIR /

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN update-ca-certificates

COPY --from=builder /workspace/prometheus-exporter .

EXPOSE 8080
ENTRYPOINT ["/prometheus-exporter"]
