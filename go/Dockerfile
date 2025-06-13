### STAGE 1: base image
ARG BASE_IMAGE_REGISTRY=cgr.dev
ARG LOCALARCH=amd64

FROM --platform=linux/$LOCALARCH $BASE_IMAGE_REGISTRY/chainguard/go:latest AS builder
ARG TARGETPLATFORM
ARG TARGETARCH
ARG LOCALARCH
ARG BUILDARCH

ENV GO111MODULE=on
ENV GOTOOLCHAIN=local
ENV GOPROXY="http://docker-registry-proxy.corp.amdocs.com:8081/repository/goproxy"
ENV PATH=$PATH:/usr/local/go/bin

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN --mount=type=cache,target=/root/go/pkg/mod,rw      \
    --mount=type=cache,target=/root/.cache/go-build,rw \
     go mod download

# Copy the go source
COPY autogen autogen
COPY controller controller

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN --mount=type=cache,target=/root/go/pkg/mod,rw         \
    --mount=type=cache,target=/root/.cache/go-build,rw    \
    echo "Building for GOOS=linux GOARCH=${TARGETARCH} on ${LOCALARCH}..." && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -a -o manager controller/cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM $BASE_IMAGE_REGISTRY/distroless/static:nonroot
ARG TARGETPLATFORM
ARG TARGETARCH

WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

LABEL org.opencontainers.image.source=https://github.com/kagent-dev/kagent
LABEL org.opencontainers.image.description="Kagent controller is the controller for the Kagent app."

ENTRYPOINT ["/manager"]
