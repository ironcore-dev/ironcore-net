# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.24 AS builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

COPY hack hack

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY api/ api/
COPY apimachinery/ apimachinery/
COPY apinetlet/ apinetlet/
COPY client-go/ client-go/
COPY cmd/ cmd/
COPY internal/ internal/
COPY metalnetlet/ metalnetlet/
COPY networkid/ networkid/
COPY utils/ utils/

ARG TARGETOS
ARG TARGETARCH

RUN mkdir bin

# Build
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on go build -ldflags="-s -w" -a -o bin/apiserver ./cmd/apiserver && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on go build -ldflags="-s -w" -a -o bin/controller-manager ./cmd/controller-manager && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on go build -ldflags="-s -w" -a -o bin/apinetlet-manager ./cmd/apinetlet && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on go build -ldflags="-s -w" -a -o bin/metalnetlet-manager ./cmd/metalnetlet

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot AS apiserver
WORKDIR /
COPY --from=builder /workspace/bin/apiserver /apiserver
USER 65532:65532

ENTRYPOINT ["/apiserver"]

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot AS controller-manager
WORKDIR /
COPY --from=builder /workspace/bin/controller-manager /manager
USER 65532:65532

ENTRYPOINT ["/manager"]

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot AS apinetlet-manager
WORKDIR /
COPY --from=builder /workspace/bin/apinetlet-manager /manager
USER 65532:65532

ENTRYPOINT ["/manager"]

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot AS metalnetlet-manager
WORKDIR /
COPY --from=builder /workspace/bin/metalnetlet-manager /manager
USER 65532:65532

ENTRYPOINT ["/manager"]
