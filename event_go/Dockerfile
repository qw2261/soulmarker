# syntax=docker/dockerfile:1
FROM node:22.14.0-alpine3.21 AS frontend

WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

FROM golang:1.25.13-alpine3.24 AS backend

ENV GOPROXY=https://goproxy.cn,direct

ARG BUILD_VERSION=dev
ARG BUILD_COMMIT=unknown
ARG BUILD_TIME=unknown

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /web/dist /build/web/dist

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X github.com/qw2261/soulmarker/event_go/internal/buildinfo.Version=${BUILD_VERSION} -X github.com/qw2261/soulmarker/event_go/internal/buildinfo.Commit=${BUILD_COMMIT} -X github.com/qw2261/soulmarker/event_go/internal/buildinfo.BuildTime=${BUILD_TIME}" \
    -o event-go ./cmd/event-go/

FROM alpine:3.19.1

RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S app -G app

WORKDIR /app
COPY --from=backend /build/event-go .
COPY --from=backend /build/web/dist /app/web/dist

RUN mkdir -p /app/data && chown -R app:app /app/data
USER app

VOLUME ["/app/data"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO /dev/null http://127.0.0.1:8080/healthz || exit 1

CMD ["./event-go"]
