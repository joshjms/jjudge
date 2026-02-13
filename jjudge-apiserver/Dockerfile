FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/apiserver .

FROM golang:1.25

WORKDIR /app
COPY --from=builder /out/apiserver /usr/local/bin/apiserver

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/apiserver"]
CMD ["server"]
