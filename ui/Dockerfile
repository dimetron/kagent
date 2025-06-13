### STAGE 1: base image
ARG BASE_IMAGE_REGISTRY=cgr.dev
FROM --platform=linux/$BUILDARCH $BASE_IMAGE_REGISTRY/chainguard/wolfi-base:latest AS base-os
ARG TARGETPLATFORM
ARG TARGETARCH
ARG LOCALARCH
ARG TOOLS_BUN_VERSION
ARG PROXY_HTTP
ARG NO_PROXY
ARG NPM_REGISTRY


# add CA certificates before installing other packages
ADD FullAmdocsCA.crt /etc/ssl/certs/FullAmdocsCA.crt
RUN cat /etc/ssl/certs/FullAmdocsCA.crt >> /etc/ssl/certs/ca-certificates.crt

ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8
RUN echo "Installing $LOCALARCH -> $TARGETARCH"   \
    && export PKS="curl openssl unzip bash git ca-certificates-bundle nginx supervisor" \
    && export https_proxy=$PROXY_HTTP  \
    && apk update   || apk update      \
    && apk add $PKS || apk add $PKS

# Use host machine's npm registry (can be overridden with --build-arg NPM_REGISTRY)
ENV DO_NOT_TRACK=1
ENV NEXT_TELEMETRY_DISABLED=1
ENV BUN_INSTALL_CACHE_DIR=/cache/bun
ENV BUN_INSTALL=/usr/local/bun
ENV NVM_DIR=/usr/local/nvm
ENV NODE_ENV=production
ENV CYPRESS_INSTALL_BINARY=0
ENV NODE_PATH=/cache/node_modules
ENV NODE_TLS_REJECT_UNAUTHORIZED=0
ENV PATH=$BUN_INSTALL/bin:$PATH

# Install Bun (uses official install script)
# brew install oven-sh/bun/bun
RUN --mount=type=cache,target=/cache/bun,rw         \
    mkdir -p $BUN_INSTALL  /app/ui /app/ui/public   \
    && export https_proxy=$PROXY_HTTP     \
    && curl -fsSL https://bun.sh/install | bash -s "bun-v$TOOLS_BUN_VERSION" \
    && bun --version

WORKDIR /app/ui

COPY package*.json .
RUN --mount=type=cache,target=/cache/bun,rw                                   \
    echo "Using npm registry: $NPM_REGISTRY -> $TARGETARCH"                   \
    && export https_proxy=$PROXY_HTTP                                         \
    && export no_proxy=$NO_PROXY                                              \
    && echo "registry=$NPM_REGISTRY" > .npmrc                                 \
    bun install --registry=$NPM_REGISTRY

#copy the rest of the files
COPY . .

RUN --mount=type=cache,target=/cache/bun,rw                                   \
    --mount=type=cache,target=/app/ui/.next/cache                             \
    --mount=type=cache,target=/app/ui/.node_modules                           \
    echo "Using npm registry: $NPM_REGISTRY -> $TARGETARCH"                   \
    && export no_proxy=$NO_PROXY                                              \
    && export http_proxy=$PROXY_HTTP                                          \
    && export https_proxy=$PROXY_HTTP                                         \
    && echo "registry=$NPM_REGISTRY" > .npmrc                                 \
    && bun pm ls --all                                                        \
    && bun install --registry=$NPM_REGISTRY                                   \
    && echo "Bun dependencies hash:"                                          \
    && bun pm hash                                                            \
    && bun run build --no-lint 2>&1 || bun run build                          \
    && echo "Bun build completed"

### STAGE 3: final
FROM $BASE_IMAGE_REGISTRY/chainguard/wolfi-base:latest AS final
ARG TARGETPLATFORM
ARG TARGETARCH
ARG LOCALARCH
ARG PROXY_HTTP
ARG NO_PROXY

# add CA certificates before installing other packages
ADD FullAmdocsCA.crt /etc/ssl/certs/FullAmdocsCA.crt
RUN cat /etc/ssl/certs/FullAmdocsCA.crt >> /etc/ssl/certs/ca-certificates.crt

ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8
RUN RUN echo "Installing -> $TARGETARCH"    \
    && export PKS="curl openssl unzip bash git ca-certificates-bundle nginx supervisor" \
    && export https_proxy=$PROXY_HTTP       \
    && apk update   || apk update           \
    && apk add $PKS || apk add $PKS

ARG TOOLS_BUN_VERSION
ENV BUN_INSTALL=/usr/local/bun
ENV PATH=$BUN_INSTALL/bin:$PATH
# Install Bun in native arch for running (uses official install script)
# brew install oven-sh/bun/bun
RUN --mount=type=cache,target=/cache/bun,rw \
    mkdir -p $BUN_INSTALL  \
    && export https_proxy=$PROXY_HTTP \
    && curl -fsSL https://bun.sh/install | bash -s "bun-v$TOOLS_BUN_VERSION" \
    && bun --version

RUN mkdir -p /app/ui/public /run/nginx/ /var/lib/nginx/tmp/ /var/lib/nginx/logs/  \
    && addgroup -g 1001    nginx                            \
    && adduser  -u 1001 -G nginx -s /bin/bash -D nextjs     \
    && adduser  -u 1002 -G nginx -s /bin/bash -D nginx      \
    && chown    -vR nextjs:nginx /app/ui                    \
    && chown    -vR nextjs:nginx /run/nginx                 \
    && chown    -vR nextjs:nginx /var/lib/nginx/

WORKDIR /app
COPY conf/nginx.conf /etc/nginx/nginx.conf
COPY conf/supervisord.conf /etc/supervisor/conf.d/supervisord.conf

WORKDIR /app/ui
COPY --from=base-os /app/ui/next.config.ts ./
COPY --from=base-os /app/ui/public ./public
COPY --from=base-os /app/ui/package.json ./package.json
COPY --from=base-os --chown=nextjs:nginx /app/ui/.next/standalone ./
COPY --from=base-os --chown=nextjs:nginx /app/ui/.next/static ./.next/static

# Ensure correct permissions
RUN chown -R nextjs:nginx /app/ui && \
    chmod -R 755 /app

EXPOSE 80

LABEL org.opencontainers.image.source=https://github.com/kagent-dev/kagent
LABEL org.opencontainers.image.description="Kagent app is the UI and apiserver for running agents."
LABEL org.opencontainers.image.authors="Kagent Creators 🤖"

USER nextjs

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]