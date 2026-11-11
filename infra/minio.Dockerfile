# The latest community security release is source-only; build its exact tag.
FROM golang:1.27.4-alpine@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf AS build
RUN apk add --no-cache git
RUN CGO_ENABLED=0 GOBIN=/out go install -ldflags='-s -w' github.com/minio/minio@RELEASE.2025-10-15T17-29-55Z
FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
RUN apk add --no-cache ca-certificates tzdata zlib=1.4-r0 && addgroup -g 10001 minio && adduser -D -u 10001 -G minio minio && mkdir /data && chown 10001:10001 /data
COPY --from=build /out/minio /usr/local/bin/minio
USER 10001:10001
EXPOSE 9000
ENTRYPOINT ["minio"]
CMD ["server", "/data", "--console-address", ":9001"]
