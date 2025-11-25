FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make
WORKDIR /app
COPY go.mod go.sum ./

RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o /app/bin/server \
    -ldflags="-s -w" \
    ./cmd/server/main.go

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o /app/bin/migrate \
    -ldflags="-s -w" \
    ./cmd/migrate/main.go

FROM alpine:latest AS runtime
RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 1000 appuser && adduser -u 1000 -D -G appuser appuser
WORKDIR /app

COPY --from=builder /app/bin/server /app/server
COPY --from=builder /app/bin/migrate /app/migrate
COPY --from=builder /app/config /app/config
COPY --from=builder /app/migrations /app/migrations

RUN chown -R appuser:appuser /app
USER appuser

FROM runtime AS server
EXPOSE ${PORT}
ENTRYPOINT ["/app/server"]

FROM runtime AS migration 
ENTRYPOINT ["/app/migrate"]