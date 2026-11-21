FROM golang:1.27.4-alpine@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf AS build
WORKDIR /src
COPY infra/rustfs-healthcheck.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/rustfs-healthcheck rustfs-healthcheck.go

FROM rustfs/rustfs:1.0.6@sha256:ee717072290bec9b11abfc0365f8310b00df1cd9b3639943edae47bdef05044a
USER root
COPY --from=build /out/rustfs-healthcheck /usr/local/bin/rustfs-healthcheck
# Keep HTTPS support for the upstream bootstrap readiness check without curl.
RUN grep -Fq 'curl -fsS "$BOOTSTRAP_HEALTH_URL"' /etc/rustfs/entrypoint.sh \
    && sed -i 's/curl -fsS "\$BOOTSTRAP_HEALTH_URL"/rustfs-healthcheck "$BOOTSTRAP_HEALTH_URL"/' /etc/rustfs/entrypoint.sh \
    && apk add --no-cache ca-certificates-bundle \
    && apk del --no-network apk-tools ca-certificates coreutils curl
USER rustfs
