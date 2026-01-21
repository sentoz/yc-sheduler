FROM alpine:3.21 AS certs

RUN apk add --no-cache ca-certificates && update-ca-certificates


FROM scratch

# Required for HTTPS calls (Yandex Cloud APIs).
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

# Expect prebuilt binary from `make build` at build/yc-scheduler.
COPY build/yc-scheduler-linux-amd64 /yc-scheduler

USER 65532:65532
ENTRYPOINT ["/yc-scheduler"]
