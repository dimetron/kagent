### STAGE 1: base image
ARG BASE_IMAGE_REGISTRY=cgr.dev
FROM --platform=linux/$BUILDARCH $BASE_IMAGE_REGISTRY/chainguard/wolfi-base:latest AS base-os

ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

RUN export https_proxy=http://127.0.0.1:3128 \
    && apk update \
    && apk add curl openssl unzip bash git ca-certificates-bundle nginx supervisor strace \
    && rm -rf /var/cache/apk/*

ARG TOOLS_BUN_VERSION
ARG TARGETARCH

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
    && export https_proxy=http://127.0.0.1:3128     \
    && curl -fsSL https://bun.sh/install | bash -s "bun-v$TOOLS_BUN_VERSION" \
    && bun --version

WORKDIR /app/ui

#copy the package.json
#COPY package*.json .
#RUN --mount=type=cache,target=/cache/bun,rw             \
#    bun install

#copy the rest of the files
COPY . .

RUN --mount=type=cache,target=/cache/bun,rw                                   \
    echo "Installing dependencies Using npm registry: http://127.0.0.1:3128"  \
    && export no_proxy=localhost,127.0.0.1                                    \
    && export https_proxy=http://127.0.0.1:3128                               \
    && echo "registry=http://127.0.0.1:4873" > .npmrc                         \
    && bun pm migrate                                                         \
    #&& bun pm ls --all | tee | gzip | base64 -w0                             \
    && bun install --registry=http://127.0.0.1:4873                           \
    && echo "Bun dependencies hash:"                                          \
    && bun pm hash                                                            \
    && bun run build --no-lint 2>&1 || bun run build

### STAGE 3: final
FROM $BASE_IMAGE_REGISTRY/chainguard/wolfi-base:latest AS final

ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

RUN export https_proxy=http://127.0.0.1:3128 \
    && apk update \
    && apk add curl openssl unzip bash git ca-certificates nginx supervisor \
    && rm -rf /var/cache/apk/*

ARG TOOLS_BUN_VERSION
ENV BUN_INSTALL=/usr/local/bun
ENV PATH=$BUN_INSTALL/bin:$PATH
# Install Bun in native arch for running (uses official install script)
# brew install oven-sh/bun/bun
RUN --mount=type=cache,target=/cache/bun,rw \
    mkdir -p $BUN_INSTALL  \
    && export https_proxy=http://127.0.0.1:3128 \
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