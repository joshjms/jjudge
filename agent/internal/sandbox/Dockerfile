FROM ubuntu:noble-20250404

RUN apt update && apt install -y \
    python3 python3-pip \
    gcc \
    g++ \
    default-jdk \
    openjdk-17-jdk \
    make bash curl

ENV GO_VERSION=1.23.8

RUN curl -fsSL https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz | tar -C /usr/local -xz \
    && ln -s /usr/local/go/bin/go /usr/local/bin/go \
    && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

RUN apt clean && rm -rf /var/lib/apt/lists/*
