FROM node:22-alpine AS frontend

WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /web/dist /build/web/dist

RUN CGO_ENABLED=0 GOOS=linux go build -o event-go ./cmd/event-go/

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=backend /build/event-go .
COPY --from=backend /build/web/dist /app/web/dist

VOLUME ["/app/data"]
EXPOSE 8080

CMD ["./event-go"]
