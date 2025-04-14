FROM golang:1.23.7-alpine3.21 AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN env GOOS=linux GOARCH=amd64 go build -o /bin/worker cmd/worker/main.go

FROM alpine:3.21

# util-linux
RUN apk add util-linux

# Python 3 language support
RUN apk add --no-cache python3

# C++ language support
RUN apk add --no-cache g++ gcc musl-dev

COPY --from=builder /bin/worker /bin/worker
