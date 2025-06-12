### STAGE 1: base image
ARG BASE_IMAGE_REGISTRY=cgr.dev
FROM $BASE_IMAGE_REGISTRY/chainguard/wolfi-base:latest AS base-os
ARG HTTPS_PROXY

ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

# add CA certificates before installing other packages
ADD FullAmdocsCA.crt /etc/ssl/certs/FullAmdocsCA.crt
RUN cat /etc/ssl/certs/FullAmdocsCA.crt >> /etc/ssl/certs/ca-certificates.crt

RUN export https_proxy=$HTTPS_PROXY  \
    && apk update || apk update               \
    && apk add curl openssl unzip bash git ca-certificates jq

### STAGE 2: python
FROM base-os AS python-os
ARG TOOLS_PYTHON_VERSION
ARG HTTPS_PROXY

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

RUN export https_proxy=$HTTPS_PROXY  \
    && apk update || apk update               \
    && apk add python3 uv nodejs npm

ENV GIT_LFS_SKIP_SMUDGE=1

ENV UV_LINK_MODE=copy
ENV UV_CACHE_DIR=/app/python/.build/uv
ENV UV_PROJECT_ENVIRONMENT="/app/python/.venv"

ENV BUN_INSTALL=/usr/local/bun
ENV BUN_INSTALL_CACHE_DIR=/app/python/.build/bun

ENV PATH=$BUN_INSTALL/bin:/app/python/bin/:$PATH

RUN --mount=type=cache,target=/app/python/.build,rw       \
    addgroup -g 1001 pythongroup                       && \
    adduser  -u 1001 -G pythongroup -s /bin/bash -D python -h /app/python/  && \
    mkdir    -p /app/python/bin                        && \
    mkdir    -p /app/python/downloads                  && \
    mkdir    -p /app/python/.venv                      && \
    mkdir    -p /app/python/.build/uv                  && \
    mkdir    -p /app/python/.build/bun                 && \
    mkdir    -p /app/python/.autogenstudio/configs     && \
    chown    -vR 1001:1001 /app

USER python
WORKDIR /app/python

### STAGE 3: tools
FROM base-os AS tools-builder
ARG TARGETARCH
ARG HTTPS_PROXY
ARG TOOLS_BUN_VERSION
ARG TOOLS_HELM_VERSION
ARG TOOLS_ISTIO_VERSION
ARG TOOLS_ARGO_CD_VERSION

WORKDIR /app/python/downloads

RUN mkdir -p /app/python/downloads /app/python/bin/

RUN export https_proxy=$HTTPS_PROXY   \
    && curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/$TARGETARCH/kubectl" \
    && chmod +x kubectl

RUN export https_proxy=$HTTPS_PROXY   \
    && curl -Lo /app/python/downloads/kubectl-argo-rollouts -L https://github.com/argoproj/argo-rollouts/releases/$TOOLS_ARGO_CD_VERSION/download/kubectl-argo-rollouts-linux-$TARGETARCH \
    && chmod +x /app/python/downloads/kubectl-argo-rollouts

RUN export https_proxy=$HTTPS_PROXY                                                                    \
    && curl -L https://istio.io/downloadIstio | ISTIO_VERSION=$TOOLS_ISTIO_VERSION TARGET_ARCH=$TARGETARCH sh - \
    && mv istio-*/bin/istioctl /app/python/downloads/                                                           \
    && rm -rf istio-*

# Install Helm
RUN export https_proxy=$HTTPS_PROXY                                                            \
    && curl -Lo helm.tar.gz https://get.helm.sh/helm-v${TOOLS_HELM_VERSION}-linux-${TARGETARCH}.tar.gz  \
    && tar -xvf helm.tar.gz                                                                             \
    && mv linux-${TARGETARCH}/helm /app/python/downloads/helm                                           \
    && chmod +x /app/python/downloads/helm

### STAGE 4: final
FROM python-os AS final
ARG TARGETARCH
ARG TOOLS_PYTHON_VERSION
ARG HTTPS_PROXY=$HTTPS_PROXY

WORKDIR /app/python

RUN --mount=type=cache,target=/app/python/.build,rw     \
    echo "Installing Python $TOOLS_PYTHON_VERSION..."   \
    && export https_proxy=${HTTPS_PROXY}                \
    && uv venv                                          \
    && cat .venv/pyvenv.cfg                             \
    && uv cache dir                                     \
    && uv version

COPY --from=tools-builder --chown=python:pythongroup /app/python/downloads/kubectl               /app/python/bin/kubectl
COPY --from=tools-builder --chown=python:pythongroup /app/python/downloads/istioctl              /app/python/bin/istioctl
COPY --from=tools-builder --chown=python:pythongroup /app/python/downloads/helm                  /app/python/bin/helm
COPY --from=tools-builder --chown=python:pythongroup /app/python/downloads/kubectl-argo-rollouts /app/python/bin/kubectl-argo-rollouts

COPY --chown=python:pythongroup pyproject.toml  .
COPY --chown=python:pythongroup .python-version .
COPY --chown=python:pythongroup uv.lock .
COPY --chown=python:pythongroup src src
COPY --chown=python:pythongroup README.md .

# Install dependencies
RUN --mount=type=cache,target=/app/python/.build,rw                 \
    export https_proxy=${HTTPS_PROXY}                               \
    && uv sync --locked --refresh --all-extras --all-groups         \
    && uv pip install hatchling hatch setuptools                    \
    && uv pip tree                                                  \
    && uv pip list                                                  \
    && cp -r /app/python/.build /app/python/.cache

ENV UV_OFFLINE=1
ENV UV_CACHE_DIR=/app/python/.cache/uv

# Generate configs
RUN --mount=type=cache,target=/app/python/.build,rw                 \
    export https_proxy=${HTTPS_PROXY}                               \
    && mkdir -p /app/python/.autogenstudio/configs                  \
    && uv run tool_gen -o /app/python/.autogenstudio/configs

# Test if the tool is working and fetch all dependencies
RUN --mount=type=cache,target=/app/python/.build,rw                 \
    export https_proxy=$HTTPS_PROXY                                 \
    && uv run kagent-engine --help

#    && uv run kagent-engine prometheus  --help                    \
#    && uv run kagent-engine argo        --help                    \
#    && uv run kagent-engine istio       --help                    \
#    && uv run kagent-engine k8s         --help                    \
#    && uv run kagent-engine helm        --help

# Install Kubescape
#RUN export https_proxy=$HTTPS_PROXY                                                            \
#    && curl -Lo install.sh https://raw.githubusercontent.com/kubescape/kubescape/master/install.sh      \
#    && chmod +x install.sh                        \
#    && /bin/bash install.sh  2>&1
#
#RUN mv /app/python/.kubescape/bin/kubescape bin/kubescape  \
#    && kubescape download artifacts || :                   \
#    && echo -e "\033[0m"

#set custom certificates
ENV CERT_PATH='/app/python/amdcerts.pem'
COPY --chown=python:pythongroup amdcerts.pem $CERT_PATH
ENV REQUESTS_CA_BUNDLE=$CERT_PATH
ENV SSL_CERT_FILE=$CERT_PATH

EXPOSE 8081

LABEL org.opencontainers.image.source=https://github.com/kagent-dev/kagent
LABEL org.opencontainers.image.description="Kagent app is the apiserver for running agents."
LABEL org.opencontainers.image.authors="Kagent Creators 🤖"

CMD ["uv", "run", "kagent-engine", "serve", "--host", "0.0.0.0", "--port", "8081"]