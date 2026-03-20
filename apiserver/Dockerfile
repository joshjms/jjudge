FROM golang:1.25 AS builder

WORKDIR /workspace

# Copy workspace config so local replace directives resolve
COPY go.work go.work.sum ./

# Copy all workspace modules (go.work requires all to be present)
COPY api/ ./api/
COPY apiserver/ ./apiserver/
COPY grader/ ./grader/
COPY worker/ ./worker/

# Download dependencies (go.work is picked up automatically)
RUN cd apiserver && go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/apiserver ./apiserver/

FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder /out/apiserver /usr/local/bin/apiserver

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/apiserver"]
CMD ["server"]
