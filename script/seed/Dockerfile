# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /seed ./script/seed/

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /seed /seed

ENTRYPOINT ["/seed"]
