FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o bin/api ./cmd/api

FROM alpine

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/bin/api ./api
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080 9090

CMD ["./api"]