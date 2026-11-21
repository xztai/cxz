FROM node:26.13.1-alpine@sha256:6bd9ab43c0ec0190ca7b2d193906cb8b7edf6e7f4afa43ddeccd6452710bffb5 AS base
WORKDIR /app

FROM base AS build
RUN npm install --global pnpm@12.18.0
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile --ignore-scripts
COPY frontend/ ./
RUN pnpm build
RUN pnpm prune --prod --ignore-scripts

FROM base AS runtime
ENV NODE_ENV=production HOST=0.0.0.0 PORT=3000
# Node carries its own TLS library. Retain the system CA bundle when removing apk.
RUN apk add --no-cache ca-certificates-bundle zlib=1.4-r0 \
    && apk del --no-network apk-tools \
    && rm -r /usr/local/lib/node_modules/npm /usr/local/bin/npm /usr/local/bin/npx
COPY --from=build --chown=node:node /app/build ./build
COPY --from=build --chown=node:node /app/package.json ./package.json
COPY --from=build --chown=node:node /app/node_modules ./node_modules
USER node
EXPOSE 3000
CMD ["node", "build"]
