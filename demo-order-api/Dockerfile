FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o /api ./cmd/api

FROM alpine:3.21

RUN apk add --no-cache ca-certificates
COPY --from=builder /api /api

EXPOSE 8080
ENTRYPOINT ["/api"]
