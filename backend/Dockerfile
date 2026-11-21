FROM golang:1.27.4-alpine@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/etheyl ./cmd/etheyl

FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
# The statically linked server needs the CA bundle and zoneinfo at runtime.
RUN apk add --no-cache ca-certificates-bundle tzdata zlib=1.4-r0 \
    && addgroup -g 10001 app && adduser -D -u 10001 -G app app \
    && apk del --no-network apk-tools
COPY --from=build /out/etheyl /usr/local/bin/etheyl
USER 10001:10001
EXPOSE 8081
ENTRYPOINT ["etheyl"]
CMD ["api"]
