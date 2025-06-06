### STAGE 1: base image
ARG BASE_IMAGE_REGISTRY=cgr.dev
FROM $BASE_IMAGE_REGISTRY/chainguard/wolfi-base:latest AS base-os
ARG TOOLS_BUN_VERSION
ARG TARGETARCH
ARG PROXY

ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8
ENV http_proxy=http://10.232.233.70:8080
ENV https_proxy=http://10.232.233.70:8080
ENV HTTP_PROXY=http://10.232.233.70:8080
ENV HTTPS_PROXY=http://10.232.233.70:8080
ENV no_proxy="localhost,127.0.0.1,172.17.0.0/12,192.168.0.0/16,.svc,.default,.local,.internal,fs.amdocs.com,.amdocs.com,.amdocs.aws,.sock,bitbucket.corp.amdocs.com,.corp.amdocs.com,docker-registry-proxy.corp.amdocs.com"

RUN export https_proxy=http://10.232.233.70:8080 \
    && apk update \
    && apk add curl bash openssl ca-certificates nginx supervisor \
    && rm -rf /var/cache/apk/*

ENV NEXT_TELEMETRY_DISABLED=1
ENV BUN_INSTALL_CACHE_DIR=/cache/bun
ENV BUN_INSTALL=/usr/local/bun
ENV NVM_DIR=/usr/local/nvm
ENV NODE_ENV=production
ENV CYPRESS_INSTALL_BINARY=0
ENV NODE_PATH=/cache/node_modules
ENV PATH=$BUN_INSTALL/bin:$PATH

# Install Bun (uses official install script)
# brew install oven-sh/bun/bun
RUN --mount=type=cache,target=/cache/bun,rw \
    mkdir -p $BUN_INSTALL  /app/ui \
    && curl -fsSL https://bun.sh/install | bash -s "bun-v$TOOLS_BUN_VERSION" \
    && bun --version

WORKDIR /app/ui

#copy the package.json
#COPY package*.json .
#COPY bunfig.toml   .
#RUN --mount=type=cache,target=/cache/bun,rw             \
#    bun install --verbose

#copy the rest of the UI files
COPY ./ .
RUN --mount=type=cache,target=/cache/bun,rw             \
    bun install --verbose

RUN --mount=type=cache,target=/cache/bun,rw             \
    export https_proxy=http://10.232.233.70:8080        \
    && mkdir -p /app/ui/public                          \
    && bun run build --no-lint  --debug  &2>1

### STAGE 3: final
FROM $BASE_IMAGE_REGISTRY/chainguard/wolfi-base:latest AS final
ARG TOOLS_BUN_VERSION
ARG TARGETARCH
ARG PROXY

ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8
ENV http_proxy=http://10.232.233.70:8080
ENV https_proxy=http://10.232.233.70:8080
ENV no_proxy="localhost,127.0.0.1,172.17.0.0/12,192.168.0.0/16,.svc,.default,.local,.internal,fs.amdocs.com,.amdocs.com,.amdocs.aws,.sock,bitbucket.corp.amdocs.com,.corp.amdocs.com,docker-registry-proxy.corp.amdocs.com"

RUN export https_proxy=http://10.232.233.70:8080 \
    && apk update \
    && apk add curl bash openssl ca-certificates nginx supervisor \
    && rm -rf /var/cache/apk/*

ENV NEXT_TELEMETRY_DISABLED=1
ENV BUN_INSTALL_CACHE_DIR=/cache/bun
ENV BUN_INSTALL=/usr/local/bun
ENV NVM_DIR=/usr/local/nvm
ENV NODE_ENV=production
ENV CYPRESS_INSTALL_BINARY=0
ENV NODE_PATH=/cache/node_modules
ENV PATH=$BUN_INSTALL/bin:$PATH

# Install Bun (uses official install script)
# brew install oven-sh/bun/bun
RUN --mount=type=cache,target=/cache/bun,rw \
    mkdir -p $BUN_INSTALL  /app/ui \
    && curl -fsSL https://bun.sh/install | bash -s "bun-v$TOOLS_BUN_VERSION" \
    && bun --version

RUN mkdir -p /app/ui /run/nginx /var/lib/nginx/tmp/     \
    && addgroup -g 1001    nginx                        \
    && adduser  -u 1001 -G nginx -s /bin/bash -D nextjs \
    && adduser  -u 1002 -G nginx -s /bin/bash -D nginx  \
    && chown    -vR nextjs:nginx /app/ui

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