#https://www.docker.com/blog/faster-multi-platform-builds-dockerfile-cross-compilation-guide/
ARG DOCKER_REGISTRY="illin4261.corp.amdocs.com:28090"
ARG DOCKER_REGISTRY_GOOGLE="illin4261.corp.amdocs.com:28090"
ARG TOOLS_UV_VERSION=0.7.2

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
    git                             \
    git-lfs                         \
    ca-certificates                 \
    curl                            \
    gnupg                           \
    wget                            \
    unzip                           \
    && update-ca-certificates       \
    && rm -rf /var/lib/apt/lists/*  \
    && ln -sf /usr/bin/python3 /usr/bin/python

FROM base-os AS base
ENV NEXT_TELEMETRY_DISABLED=1

### STAGE 2: build + tools
FROM --platform=$TARGETPLATFORM base AS python-builder

ARG TARGETARCH
ARG TOOLS_ISTIO_VERSION
ARG TOOLS_ARGO_CD_VERSION

WORKDIR /app/downloads
ENV https_proxy=http://10.232.233.70:8080

# Install kubectl
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/$TARGETARCH/kubectl" \
    && chmod +x kubectl \
    && mv kubectl /usr/local/bin/

# Install kubectl argo plugin
RUN curl -Lo /usr/local/bin/kubectl-argo-rollouts -L https://github.com/argoproj/argo-rollouts/releases/$TOOLS_ARGO_CD_VERSION/download/kubectl-argo-rollouts-linux-$TARGETARCH \
    && chmod +x /usr/local/bin/kubectl-argo-rollouts

# Install Istio
RUN curl -L https://istio.io/downloadIstio | ISTIO_VERSION=$TOOLS_ISTIO_VERSION TARGET_ARCH=$TARGETARCH sh - \
    && mv istio-$TOOLS_ISTIO_VERSION/bin/istioctl /usr/local/bin/istioctl \
    && rm -rf istio-$TOOLS_ISTIO_VERSION

# Install Helm
RUN curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 \
    && chmod 700 get_helm.sh \
    && ./get_helm.sh \
    && rm get_helm.sh \

# Install Kubescape
RUN curl -Ls https://raw.githubusercontent.com/kubescape/kubescape/master/install.sh | bash

##################### STAGE 3: final
FROM base-os AS final

COPY --from=python-builder /usr/local/bin/helm /usr/local/bin/helm
COPY --from=python-builder /usr/local/bin/kubectl /usr/local/bin/kubectl
COPY --from=python-builder /usr/local/bin/istioctl /usr/local/bin/istioctl
COPY --from=python-builder /usr/local/bin/kubectl-argo-rollouts /usr/local/bin/kubectl-argo-rollouts
#COPY --from=python-builder /usr/local/bin/kubescape /usr/local/bin/kubescape


ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
ENV GIT_LFS_SKIP_SMUDGE=1
ENV UV_LINK_MODE=copy
ENV UV_PROJECT_ENVIRONMENT="/app/python/.venv"
ENV https_proxy=""

# Set up Python backend
WORKDIR /app/python

# Create groups and users
RUN echo "Creating app user" \
    && groupadd -g 1001 pythongroup                               \
    && useradd  -u 1001 -g pythongroup -s /bin/bash -m python     \
    && chown -R python:pythongroup /app/python

# Install python
RUN --mount=type=cache,target=/app/python/.cache,rw  \
    export https_proxy=http://10.232.233.70:8080     \
    && uv venv .venv                                 \
    && uv python install 3.12                        \
    && uv cache dir                                  \
    && uv version

COPY pyproject.toml .
COPY .python-version .
COPY uv.lock .
COPY README.md .
COPY src src

# Install dependencies
RUN --mount=type=cache,target=/app/python/.cache                  \
    --mount=type=bind,source=pyproject.toml,target=pyproject.toml \
    export https_proxy=http://10.232.233.70:8080                  \
    && uv venv .venv                                              \
    && uv sync --locked --no-install-project --no-editable

# Generate configs
RUN --mount=type=cache,target=/app/python/.cache                  \
    --mount=type=bind,source=pyproject.toml,target=pyproject.toml \
    export https_proxy=http://10.232.233.70:8080                  \
    && mkdir -p /app/python/.autogenstudio/configs                \
    && uv run tool_gen -o /app/python/.autogenstudio/configs

RUN --mount=type=cache,target=/app/python/.cache                  \
    --mount=type=bind,source=pyproject.toml,target=pyproject.toml \
    export https_proxy=http://10.232.233.70:8080                  \
    && uv run kagent-engine --help

# Ensure correct permissions
RUN chown -R python:pythongroup /app/python &&                    \
    chmod -vR 755 /app/python/.venv/bin

EXPOSE 8081
ENV UV_OFFLINE=1

COPY amdcerts.pem /
ENV CERT_PATH = '/amdcerts.pem'
ENV REQUESTS_CA_BUNDLE=$CERT_PATH
ENV SSL_CERT_FILE=$CERT_PATH

LABEL org.opencontainers.image.source=https://github.com/kagent-dev/kagent
LABEL org.opencontainers.image.description="Kagent app is the apiserver for running agents."
LABEL org.opencontainers.image.authors="Kagent Creators 🤖"

CMD ["uv", "run", "kagent-engine", "serve", "--host", "0.0.0.0", "--port", "8081"]
