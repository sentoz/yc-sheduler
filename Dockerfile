FROM golang:1.25.5-alpine AS builder

# hadolint ignore=DL3018
RUN apk add --no-cache ca-certificates tzdata && update-ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Generate JSON schema required for go:embed (static/schemas/config.json).
RUN go run ./cmd/schema-gen -out static/schemas/config.json

ARG VERSION=v0.0.0
ARG COMMIT=000000
ARG BUILD_TIME=1970-01-01T00:00:00
ARG URL=https://github.com/sentoz/yc-sheduler

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
	go build -buildvcs=false -trimpath \
		-tags "forceposix" \
		-ldflags="-s -w \
			-X 'github.com/sentoz/yc-sheduler/internal/vars.Version=$VERSION' \
			-X 'github.com/sentoz/yc-sheduler/internal/vars.Commit=$COMMIT' \
			-X 'github.com/sentoz/yc-sheduler/internal/vars._buildTime=$BUILD_TIME' \
			-X 'github.com/sentoz/yc-sheduler/internal/vars.URL=$URL'" \
		-o /out/yc-scheduler ./cmd/yc-scheduler


FROM scratch

# Required for HTTPS calls (Yandex Cloud APIs).
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

COPY --from=builder /out/yc-scheduler /yc-scheduler

USER 65532:65532
ENTRYPOINT ["/yc-scheduler"]
