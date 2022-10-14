# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.19 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY api/ api/
COPY apinetlet/ apinetlet/
COPY flag/ flag/
COPY onmetal-api-net/ onmetal-api-net/

ARG TARGETOS
ARG TARGETARCH

RUN mkdir bin

# Build
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on go build -ldflags="-s -w" -a -o bin/onmetal-api-net-manager ./onmetal-api-net && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on go build -ldflags="-s -w" -a -o bin/apinetlet-manager ./apinetlet

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot AS onmetal-api-net-manager
WORKDIR /
COPY --from=builder /workspace/bin/onmetal-api-net-manager .
USER 65532:65532

ENTRYPOINT ["/manager"]

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot AS apinetlet-manager
WORKDIR /
COPY --from=builder /workspace/bin/apinetlet-manager manager
USER 65532:65532

ENTRYPOINT ["/manager"]
