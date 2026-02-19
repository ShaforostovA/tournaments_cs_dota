FROM golang:1.22-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download || true

RUN apk add --no-cache curl

COPY . .

EXPOSE 8080

CMD ["go", "run", "./cmd/server"]
