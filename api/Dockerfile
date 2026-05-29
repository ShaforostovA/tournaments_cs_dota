# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /out/tournaments-api \
    ./cmd/server

FROM alpine:3.20 AS runtime

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup

COPY --from=builder /out/tournaments-api /app/tournaments-api

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/tournaments-api"]
