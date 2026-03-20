FROM golang:1.25 AS go-builder

WORKDIR /workspace

# Copy workspace config so local replace directives resolve
COPY go.work go.work.sum ./

# Copy all workspace modules (go.work requires all to be present)
COPY api/ ./api/
COPY apiserver/ ./apiserver/
COPY grader/ ./grader/
COPY worker/ ./worker/

# Download dependencies (go.work is picked up automatically)
RUN cd worker && go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/worker ./worker/

# Build lime binary
FROM gcc:14 AS lime-builder

WORKDIR /lime
COPY lime/ ./
RUN mkdir -p build && make

# Runtime image
FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    uidmap \
    && rm -rf /var/lib/apt/lists/*

# ubuntu:24.04 already ships with uid/gid 1000 as the 'ubuntu' user.
# Add subuid/subgid ranges for the ubuntu user so newuidmap/newgidmap work.
RUN echo 'ubuntu:100000:65536' >> /etc/subuid && \
    echo 'ubuntu:100000:65536' >> /etc/subgid

# Rootfs is downloaded at container startup via entrypoint.sh.
RUN mkdir -p /rootfs && chown 1000:1000 /rootfs

WORKDIR /app
COPY --from=go-builder /out/worker /usr/local/bin/worker
COPY --from=lime-builder /lime/build/lime /usr/local/bin/lime
COPY worker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Create work directories owned by uid 1000 (ubuntu)
RUN mkdir -p /tmp/judge/submissions /tmp/judge/work /tmp/judge/overlayfs /tmp/judge/rootfs \
    && chown -R 1000:1000 /tmp/judge

USER ubuntu

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
