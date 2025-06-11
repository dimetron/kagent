### STAGE 1: base image
ARG BASE_IMAGE_REGISTRY=cgr.dev
FROM $BASE_IMAGE_REGISTRY/chainguard/wolfi-base:latest AS base-os

ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

# add CA certificates before installing other packages
ADD FullAmdocsCA.crt /etc/ssl/certs/FullAmdocsCA.crt
RUN cat /etc/ssl/certs/FullAmdocsCA.crt >> /etc/ssl/certs/ca-certificates.crt

RUN export https_proxy=http://127.0.0.1:3128  \
    && apk update || apk update               \
    && apk add curl openssl unzip bash git ca-certificates jq

### STAGE 2: python
FROM base-os AS python-os
ARG TOOLS_PYTHON_VERSION=3.12

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
RUN export https_proxy=http://127.0.0.1:3128  \
    && apk update || apk update               \
    && apk add python3 uv nodejs npm

ENV GIT_LFS_SKIP_SMUDGE=1
ENV UV_LINK_MODE=copy
ENV UV_PROJECT_ENVIRONMENT="/app/python/.venv"

ENV BUN_INSTALL=/usr/local/bun
ENV BUN_INSTALL_CACHE_DIR=/app/python/.cache
ENV PATH=$BUN_INSTALL/bin:/app/python/bin/:$PATH

RUN addgroup -g 1001 pythongroup && \
    adduser  -u 1001 -G pythongroup -s /bin/bash -D python -h /app/python/ && \
    mkdir -p /tmp /app/python/.autogenstudio/configs                       && \
    chown -vR python:pythongroup /app


### STAGE 3: tools
FROM python-os AS python-builder
ARG TARGETARCH

ARG TOOLS_BUN_VERSION
ARG TOOLS_HELM_VERSION
ARG TOOLS_ISTIO_VERSION
ARG TOOLS_ARGO_CD_VERSION

RUN mkdir -p /app/python/downloads /app/python/bin/
WORKDIR /app/python/downloads

RUN export https_proxy=http://127.0.0.1:3128   \
    && curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/$TARGETARCH/kubectl" \
    && chmod +x kubectl

RUN export https_proxy=http://127.0.0.1:3128   \
    && curl -Lo /app/python/downloads/kubectl-argo-rollouts -L https://github.com/argoproj/argo-rollouts/releases/$TOOLS_ARGO_CD_VERSION/download/kubectl-argo-rollouts-linux-$TARGETARCH \
    && chmod +x /app/python/downloads/kubectl-argo-rollouts

RUN export https_proxy=http://127.0.0.1:3128                                                                    \
    && curl -L https://istio.io/downloadIstio | ISTIO_VERSION=$TOOLS_ISTIO_VERSION TARGET_ARCH=$TARGETARCH sh - \
    && mv istio-*/bin/istioctl /app/python/downloads/                                                           \
    && rm -rf istio-*

# Install Helm
RUN export https_proxy=http://127.0.0.1:3128                                                            \
    && curl -Lo helm.tar.gz https://get.helm.sh/helm-v${TOOLS_HELM_VERSION}-linux-${TARGETARCH}.tar.gz  \
    && tar -xvf helm.tar.gz                                                                             \
    && mv linux-${TARGETARCH}/helm /app/python/downloads/helm                                           \
    && chmod +x /app/python/downloads/helm

### STAGE 4: final
FROM python-os AS final
ARG TARGETARCH

USER python
WORKDIR /app/python

RUN --mount=type=cache,target=/app/python/.cache,rw     \
    echo "Installing Python $TOOLS_PYTHON_VERSION..."   \
    && export https_proxy=http://127.0.0.1:3128         \
    && uv venv .venv                                    \
    && uv python install $TOOLS_PYTHON_VERSION          \
    && uv cache dir                                     \
    && uv version

COPY --from=python-builder --chown=python:pythongroup /app/python/downloads/kubectl               /app/python/bin/kubectl
COPY --from=python-builder --chown=python:pythongroup /app/python/downloads/istioctl              /app/python/bin/istioctl
COPY --from=python-builder --chown=python:pythongroup /app/python/downloads/helm                  /app/python/bin/helm
COPY --from=python-builder --chown=python:pythongroup /app/python/downloads/kubectl-argo-rollouts /app/python/bin/kubectl-argo-rollouts

COPY --chown=python:pythongroup pyproject.toml  .
COPY --chown=python:pythongroup .python-version .
COPY --chown=python:pythongroup uv.lock .
COPY --chown=python:pythongroup src src
COPY --chown=python:pythongroup README.md .

# update permissions and ownership
RUN --mount=type=cache,target=/app/python/.cache                  \
    echo "Update permissions and ownership with readonly"         \
    && chmod -R 777 /app/python/bin/*                             \
    && chown -R python:pythongroup /app/ || :

# Install dependencies
RUN --mount=type=cache,target=/app/python/.cache                  \
    --mount=type=bind,source=pyproject.toml,target=pyproject.toml \
    export https_proxy=http://127.0.0.1:3128                      \
    && uv venv .venv && uv sync --locked                          \
    && uv pip install hatchling hatch setuptools

# Generate configs
RUN --mount=type=cache,target=/app/python/.cache                  \
    --mount=type=bind,source=pyproject.toml,target=pyproject.toml \
    export https_proxy=http://127.0.0.1:3128                      \
    && mkdir -p /app/python/.autogenstudio/configs                \
    && uv run tool_gen -o /app/python/.autogenstudio/configs

# Test if the tool is working and fetch all dependencies
RUN --mount=type=cache,target=/app/python/.cache                  \
    --mount=type=bind,source=pyproject.toml,target=pyproject.toml \
    export https_proxy=http://127.0.0.1:3128                      \
    && uv run kagent-engine --help

# Install Kubescape
RUN export https_proxy=http://127.0.0.1:3128                                                            \
    && curl -Lo install.sh https://raw.githubusercontent.com/kubescape/kubescape/master/install.sh      \
    && chmod +x install.sh                        \
    && /bin/bash install.sh  2>&1

RUN mv /app/python/.kubescape/bin/kubescape bin/kubescape  \
    && kubescape download artifacts || :                   \
    && echo -e "\033[0m"

ENV UV_OFFLINE=1

#set custom certificates
ENV CERT_PATH='/app/python/amdcerts.pem'
COPY --chown=python:pythongroup amdcerts.pem $CERT_PATH
ENV REQUESTS_CA_BUNDLE=$CERT_PATH
ENV SSL_CERT_FILE=$CERT_PATH

EXPOSE 8081

LABEL org.opencontainers.image.source=https://github.com/kagent-dev/kagent
LABEL org.opencontainers.image.description="Kagent app is the apiserver for running agents."
LABEL org.opencontainers.image.authors="Kagent Creators 🤖"

CMD ["uv", "run", "kagent-engine", "serve", "--host", "0.0.0.0", "--port", "8081", "--appdir" , "/app/python/.autogenstudio"]