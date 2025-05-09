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
    git-lfs                   \
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
ENV UV_CACHE_DIR=/app/python/.cache
ENV BUN_INSTALL_CACHE_DIR=/cache/bun
ENV BUN_INSTALL=/usr/local/bun
ENV NVM_DIR=/usr/local/nvm
ENV NODE_ENV=production
ENV NODE_PATH=$NVM_DIR/v$TOOLS_NODE_VERSION/lib/node_modules
ENV PATH=$NVM_DIR/versions/node/v$TOOLS_NODE_VERSION/bin:$BUN_INSTALL/bin:$PATH

# Install bun
RUN --mount=type=cache,target=/cache/bun,rw             \
    export https_proxy=http://10.232.233.70:8080        \
    && mkdir -p /usr/local/bun                          \
    && curl -fsSL https://bun.sh/install | bash         \
    && ln -sf $BUN_INSTALL/bin/bun /usr/bin/bun         \
    && bun --version

### STAGE 2: build + tools
FROM --platform=$TARGETPLATFORM base AS ui-builder
ARG PROXY
ARG TARGETARCH

#
WORKDIR /app/ui

#install packages
COPY package*.json ./

RUN mkdir -p /app/ui/public
RUN --mount=type=cache,target=/cache/bun,rw             \
    echo "Installing bun packages..."                   \
    && export https_proxy=http://10.232.233.70:8080     \
    && bun install

COPY . .
RUN --mount=type=cache,target=/cache/bun,rw             \
    echo "Running bun build..."                         \
    && export https_proxy=http://10.232.233.70:8080     \
    && bun run build

### STAGE 3: final
FROM base AS final

RUN echo "Installing "                                  \
    && apt-get update                                   \
    && apt-get install -y --no-install-recommends       \
    nginx                                               \
    supervisor                                          \
    && rm -rf /var/lib/apt/lists/*                      \
    && ln -sf /usr/bin/python3 /usr/bin/python

# Create groups and users
RUN mkdir -p /app/ui                                    \
    && groupadd -g 1001 nodejs                          \
    && useradd -u 1001 -g nodejs -s /bin/bash -m nextjs \
    && chown -R nextjs:nodejs /app/ui

# Set up Next.js UI
WORKDIR /app/ui

COPY --from=ui-builder /app/ui/next.config.ts ./
COPY --from=ui-builder /app/ui/public ./public
COPY --from=ui-builder /app/ui/package.json ./package.json
COPY --from=ui-builder --chown=nextjs:nodejs /app/ui/.next/standalone ./
COPY --from=ui-builder --chown=nextjs:nodejs /app/ui/.next/static ./.next/static

# Set up Nginx and Supervisor
WORKDIR /app
COPY conf/nginx.conf /etc/nginx/nginx.conf
COPY conf/supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# Ensure correct permissions
RUN chown -R nextjs:nodejs /app/ui && \
    chmod -R 755 /app

EXPOSE 80

LABEL org.opencontainers.image.description="Kagent app is the UI and apiserver for running agents."
LABEL org.opencontainers.image.source=https://github.com/kagent-dev/kagent
LABEL org.opencontainers.image.authors="Kagent Creators 🤖"

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
