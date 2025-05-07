# Build the manager binary
ARG TOOLS_GO_VERSION=latest
FROM --platform=$BUILDPLATFORM docker-registry-proxy.corp.amdocs.com/golang:${TOOLS_GO_VERSION}-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG PROXY


#proxy env
ENV http_proxy=${PROXY:-http://10.232.233.70:8080}
ENV https_proxy=${PROXY:-http://10.232.233.70:8080}
ENV no_proxy="localhost,127.0.0.1,172.17.0.0/12,192.168.0.0/16,.svc,.default,.local,.internal,fs.amdocs.com,.amdocs.com,.amdocs.aws,.sock,bitbucket.corp.amdocs.com,.corp.amdocs.com,docker-registry-proxy.corp.amdocs.com"

ENV GO111MODULE=on
ENV GOTOOLCHAIN=local
ENV GOPROXY="http://docker-registry-proxy.corp.amdocs.com:8081/repository/goproxy/,http://proxy.golang.org"
ENV PATH=$PATH:/usr/local/go/bin

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN --mount=type=cache,target=/root/go/pkg/mod,rw      \
    --mount=type=cache,target=/root/.cache/go-build,rw \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go mod tidy -v

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
FROM --platform=$BUILDPLATFORM docker-registry-proxy.corp.amdocs.com/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

LABEL org.opencontainers.image.source=https://github.com/kagent-dev/kagent
LABEL org.opencontainers.image.description="Kagent controller is the controller for the Kagent app."

ENTRYPOINT ["/manager"]
