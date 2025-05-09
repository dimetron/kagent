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

ENV NEXT_TELEMETRY_DISABLED=1
ENV UV_CACHE_DIR=/app/python/.cache/uv
ENV BUN_INSTALL_CACHE_DIR=/cache/bun
ENV BUN_INSTALL=/usr/local/bun
ENV NVM_DIR=/usr/local/nvm
ENV NODE_ENV=production
ENV NODE_PATH=$NVM_DIR/v$TOOLS_NODE_VERSION/lib/node_modules
ENV PATH=$NVM_DIR/versions/node/v$TOOLS_NODE_VERSION/bin:$BUN_INSTALL/bin:$PATH

# Install python
RUN --mount=type=cache,target=/app/python/.cache/uv,rw  \
    export https_proxy=http://10.232.233.70:8080  \
    && uv python install 3.12 \
    && uv version \
    && uv cache dir

# Install bun
RUN --mount=type=cache,target=/cache/bun,rw \
    export https_proxy=http://10.232.233.70:8080  \
    && curl -fsSL https://bun.sh/install | bash   \
    && ln -sf $BUN_INSTALL/bin/bun /usr/bin/bun   \
    && bun --version

### STAGE 2: build + tools
FROM --platform=$TARGETPLATFORM base AS python-builder
ARG PROXY
ARG TARGETARCH

ARG TOOLS_ISTIO_VERSION
ARG TOOLS_ARGO_CD_VERSION

RUN apt-get update && apt-get install -y --no-install-recommends \
    bsdmainutils              \
    build-essential           \
    ca-certificates           \
    conntrack                 \
    curl                      \
    gettext-base              \
    git                       \
    gnupg                     \
    gnupg2                    \
    htop                      \
    iperf3                    \
    iproute2                  \
    iptables                  \
    iputils-ping              \
    jq                        \
    knot-dnsutils             \
    libffi-dev                \
    libssl-dev                \
    lsof                      \
    net-tools                 \
    netcat-openbsd            \
    python3                   \
    python3-dev               \
    python3-pip               \
    python3-venv              \
    sshuttle                  \
    tcpdump                   \
    wget                      \
    zsh                       \
    && curl -fsSL https://get.docker.com -o get-docker.sh  \
    && sh get-docker.sh       \
    && apt-get clean          \
    && rm -rf /var/log/*log   \
    && rm -rf /var/lib/apt/lists/*

### STAGE 3 add golang
FROM builder AS node-python-golang
ARG TOOLS_GO_VERSION
ARG TOOLS_K9S_VERSION
ARG TOOLS_KIND_VERSION
ARG TOOLS_ISTIO_VERSION
ARG TOOLS_ARGO_CD_VERSION
ARG TOOLS_KUBECTL_VERSION
ARG PROXY="http://10.232.233.70:8080"
ARG LOCALPLATFORM

#ENVIRONMENT VARIABLES
ENV DEBIAN_FRONTEND=noninteractive
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

### STAGE 3 add tools
FROM --platform=$BUILDPLATFORM node-python-golang AS tools-node-python-golang

# Install kubectl argo plugin
RUN curl -o /usr/local/bin/kubectl-argo-rollouts -L https://github.com/argoproj/argo-rollouts/releases/${TOOLS_ARGO_CD_VERSION}/download/kubectl-argo-rollouts-linux-${ARCH} \
    && chmod +x /usr/local/bin/kubectl-argo-rollouts

#Install Helm
RUN curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 \
    && chmod 700 get_helm.sh \
    && ./get_helm.sh \
    && rm get_helm.sh

# Install Istioctl
RUN curl -L https://istio.io/downloadIstio | ISTIO_VERSION=${TOOLS_ISTIO_VERSION} TARGET_ARCH=${ARCH} sh - \
    && mv istio-${TOOLS_ISTIO_VERSION}/bin/istioctl /usr/local/bin/istioctl \
    && rm -rf istio-$TOOLS_ISTIO_VERSION

# Install kind
RUN curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.27.0/kind-$(uname)-${ARCH} \
    && chmod +x ./kind \
    && mv ./kind /usr/local/bin/

# Install k9s
RUN curl -LO https://github.com/derailed/k9s/releases/download/v${TOOLS_K9S_VERSION}/k9s_Linux_${ARCH}.tar.gz \
    && tar -xvf k9s_Linux_${ARCH}.tar.gz \
    && mv k9s /usr/local/bin/ \
    && chmod +x /usr/local/bin/k9s \
    && rm -f k9s_Linux_${ARCH}.tar.gz

# Install kubectl \
RUN curl -Lo /usr/local/bin/kubectl "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/${ARCH}/kubectl" \
    && chmod +x /usr/local/bin/kubectl

RUN sh -c "$(wget https://raw.github.com/robbyrussell/oh-my-zsh/master/tools/install.sh -O -)" \
    sh -c "$(curl -fsSL https://starship.rs/install.sh)" -- --yes \
    && mkdir -p /root/.config \
    && git config --global core.excludesfile /root/.gitignore_global \
    && echo "**/.idea/" > /root/.gitignore_global

ENTRYPOINT ["zsh"]