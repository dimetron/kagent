#https://www.docker.com/blog/faster-multi-platform-builds-dockerfile-cross-compilation-guide/
ARG DOCKER_REGISTRY="illin4261.corp.amdocs.com:28090"
ARG DOCKER_REGISTRY_GOOGLE="illin4261.corp.amdocs.com:28090"
ARG TOOLS_UV_VERSION=0.7.2
ARG LOCALPLATFORM

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
    unzip                     \
    && update-ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && ln -sf /usr/bin/python3 /usr/bin/python

FROM base-os AS base

# Install nvm
ENV NEXT_TELEMETRY_DISABLED=1
ENV BUN_INSTALL=/usr/local/bun
ENV BUN_INSTALL_CACHE_DIR=/cache/bun
ENV NVM_DIR=/usr/local/nvm
ENV NODE_ENV=production
ENV NODE_PATH=$NVM_DIR/v$TOOLS_NODE_VERSION/lib/node_modules
ENV PATH=$NVM_DIR/versions/node/v$TOOLS_NODE_VERSION/bin:$BUN_INSTALL/bin:$PATH

# Install python
ENV UV_CACHE_DIR=/app/python/.cache/uv
RUN export https_proxy=http://10.232.233.70:8080  \
    && uv python install 3.12 \
    && uv version \
    && uv cache dir

# Install bun
RUN export https_proxy=http://10.232.233.70:8080  \
    && curl -fsSL https://bun.sh/install | bash   \
    && ln -sf $BUN_INSTALL/bin/bun /usr/bin/bun   \
    && bun --version

### STAGE 2: build + tools
FROM --platform=$TARGETPLATFORM base AS python-builder
ARG PROXY
ARG TARGETARCH

ARG TOOLS_ISTIO_VERSION
ARG TOOLS_ARGO_CD_VERSION

ENV https_proxy=http://10.232.233.70:8080

#
WORKDIR /app/

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
    && rm get_helm.sh

### STAGE 3: final
FROM base AS final

COPY --from=python-builder /usr/local/bin/kubectl /usr/local/bin/kubectl
COPY --from=python-builder /usr/local/bin/istioctl /usr/local/bin/istioctl
COPY --from=python-builder /usr/local/bin/helm /usr/local/bin/helm
COPY --from=python-builder /usr/local/bin/kubectl-argo-rollouts /usr/local/bin/kubectl-argo-rollouts

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1        \
    PIP_NO_CACHE_DIR=1        \
    UV_LINK_MODE=copy         \
    https_proxy=""            \
    PATH="/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/sbin:/bin:/root/.cargo/bin"

# Create groups and users
RUN groupadd -g 1002 pythongroup \
    && useradd  -u 1002 -g pythongroup -s /bin/bash -m python \
    && mkdir -p /app/python /root/.cache/uv \
    && chown -R python:pythongroup /app/python

# Set up Python backend
WORKDIR /app/python
COPY pyproject.toml .
COPY .python-version .
COPY uv.lock .
COPY README.md .
RUN --mount=type=cache,target=/root/.cache/uv uv cache dir && uv venv .venv && uv pip install hatchling
RUN --mount=type=cache,target=/root/.cache/uv uv sync --frozen
COPY src src
RUN chown -R python:pythongroup /app/python

# Generate tools and agents
RUN mkdir -p /root/.autogenstudio/configs
RUN --mount=type=cache,target=/root/.cache/uv uv run tool_gen -o /root/.autogenstudio/configs

# Ensure correct permissions
RUN chown -R python:pythongroup /app/python && \
    chmod -R 755 /app

RUN mkdir -p /app/python/.cache/uv && \
    chown -R python:pythongroup /app/python/.cache

EXPOSE 8081

LABEL org.opencontainers.image.source=https://github.com/kagent-dev/kagent
LABEL org.opencontainers.image.description="Kagent app is the apiserver for running agents."

CMD ["uv", "run", "kagent-engine", "serve", "--host", "0.0.0.0", "--port", "8081"]
