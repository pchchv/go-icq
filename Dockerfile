FROM golang:1.25.5-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o go_icq ./cmd/server

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/go_icq /app/

EXPOSE 5190 8080 9898 1088

CMD ["/app/go_icq"]
