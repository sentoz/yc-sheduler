FROM alpine:3.21 AS certs

# hadolint ignore=DL3018
RUN apk add --no-cache ca-certificates && update-ca-certificates


FROM alpine:3.21 AS tzdata

# Install timezone data
RUN apk add --no-cache tzdata


FROM scratch

# Required for HTTPS calls (Yandex Cloud APIs).
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

# Required for timezone support (IANA timezone database).
COPY --from=tzdata /usr/share/zoneinfo /usr/share/zoneinfo
ENV ZONEINFO=/usr/share/zoneinfo

# Expect prebuilt linux/amd64 binary from `make release` at build/yc-scheduler-linux-amd64.
COPY --chmod=755 build/yc-scheduler-linux-amd64 /yc-scheduler

USER 65532:65532
ENTRYPOINT ["/yc-scheduler"]
