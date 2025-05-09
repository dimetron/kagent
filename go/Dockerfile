#https://www.docker.com/blog/faster-multi-platform-builds-dockerfile-cross-compilation-guide/
ARG DOCKER_REGISTRY="illin4261.corp.amdocs.com:28090"
ARG DOCKER_REGISTRY_GOOGLE="illin4261.corp.amdocs.com:28090"
ARG TOOLS_UV_VERSION=0.7.2
ARG LOCALPLATFORM
ARG PROXY="genproxy:8080"
ARG NOPROXY

### STAGE 1: base image
FROM $DOCKER_REGISTRY/astral-sh/uv:${TOOLS_UV_VERSION}-bookworm-slim AS base-os
ARG TOOLS_NODE_VERSION

#ENVIRONMENT VARIABLES
ENV DEBIAN_FRONTEND=noninteractive

RUN echo /etc/apt/apt.conf.d/99genproxy && \
    echo "Acquire::http::Pipeline-Depth 0;"     > /etc/apt/apt.conf.d/99genproxy   && \
    echo "Acquire::http::No-Cache=True;"        >> /etc/apt/apt.conf.d/99genproxy  && \
    echo "Acquire::BrokenProxy=True;"           >> /etc/apt/apt.conf.d/99genproxy  && \
    echo "Acquire::http::Timeout \"120\";"      >> /etc/apt/apt.conf.d/99genproxy  && \
    echo "Acquire::http::Proxy \"http://10.232.233.70:8080\";" >> /etc/apt/apt.conf.d/99proxy   \
    && apt-get update && apt-get install -y --no-install-recommends \
    git                       \
    ca-certificates           \
    curl                      \
    gnupg                     \
    wget                      \
    && update-ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && ln -sf /usr/bin/python3 /usr/bin/python

FROM --platform=linux/$LOCALPLATFORM base-os AS go-base
ARG TOOLS_GO_VERSION=1.24.3
ARG LOCALPLATFORM
ARG GOPROXY="http://proxy.golang.org"
ARG PROXY="http://10.232.233.70:8080"

ENV GO111MODULE=on
ENV GOTOOLCHAIN=local
ENV GONOSUMDB="bitbucket.corp.amdocs.com"
ENV GOPRIVATE="bitbucket.corp.amdocs.com/*"
ENV GOINSECURE="bitbucket.corp.amdocs.com/*"
ENV GOPROXY="http://docker-registry-proxy.corp.amdocs.com:8081/repository/goproxy/,http://proxy.golang.org"
ENV PATH=$PATH:/usr/local/go/bin

RUN --mount=type=cache,target=/cache,rw \
    wget -e https_proxy=$PROXY  -q --show-progress -P /cache https://golang.org/dl/go${TOOLS_GO_VERSION}.linux-${LOCALPLATFORM}.tar.gz \
    && tar     -C /usr/local -xzf /cache/go${TOOLS_GO_VERSION}.linux-${LOCALPLATFORM}.tar.gz

# verify the installation
RUN echo "LOCALPLATFORM=$LOCALPLATFORM" \
    && set \
    && uname -a \
    && go version

FROM --platform=linux/$LOCALPLATFORM go-base AS builder

ARG LOCALPLATFORM
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN --mount=type=cache,target=/root/go/pkg/mod,rw      \
    --mount=type=cache,target=/root/.cache/go-build,rw \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go mod download

# Copy the go source
COPY autogen autogen
COPY controller controller

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN --mount=type=cache,target=/root/go/pkg/mod,rw      \
    --mount=type=cache,target=/root/.cache/go-build,rw \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager controller/cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM $DOCKER_REGISTRY_GOOGLE/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

LABEL org.opencontainers.image.source=https://github.com/kagent-dev/kagent
LABEL org.opencontainers.image.description="Kagent controller is the controller for the Kagent app."

ENTRYPOINT ["/manager"]
