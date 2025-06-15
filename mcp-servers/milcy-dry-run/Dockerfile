ARG LOCALARCH
FROM --platform=linux/$LOCALARCH docker-registry-proxy.corp.amdocs.com/chainguard/go:latest AS builder
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETARCH
ARG BUILDARCH
ARG LOCALARCH

ENV GO111MODULE=on
ENV GOTOOLCHAIN=local
ENV GOPROXY="http://docker-registry-proxy.corp.amdocs.com:8081/repository/goproxy"
ENV PATH=$PATH:/usr/local/go/bin

WORKDIR /app

COPY go.mod go.mod
COPY go.sum go.sum
COPY mcp-milcy-dryrun.go mcp-milcy-dryrun.go

RUN set
RUN --mount=type=cache,target=/root/go/pkg/mod,rw         \
    --mount=type=cache,target=/root/.cache/go-build,rw    \
    echo "Downloading go.mod GOARCH=${TARGETARCH} on ${BUILDARCH}..." && \
    CGO_ENABLED=0 GOARCH=${TARGETARCH} go mod tidy                    && \
    CGO_ENABLED=0 GOARCH=${TARGETARCH} go mod download

# Build the Go binary (assuming your main file is named main.go)

RUN --mount=type=cache,target=/root/go/pkg/mod,rw         \
    --mount=type=cache,target=/root/.cache/go-build,rw    \
    echo "Building for GOARCH=${TARGETARCH} on ${BUILDARCH}..." && \
    CGO_ENABLED=0 GOARCH=${TARGETARCH} go build -a -o /mcp-milcy-dryrun

FROM docker-registry-proxy.corp.amdocs.com/distroless/static:nonroot
ARG TARGETPLATFORM
ARG BUILDPLATFORM

WORKDIR /
COPY --from=builder /mcp-milcy-dryrun .

# Expose the port (replace 8080 with your desired port)
EXPOSE 8080

# Command to run the server
CMD ["/mcp-milcy-dryrun"]
