FROM debian:trixie AS simgrid-builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    cmake \
    g++ \
    libsimgrid-dev \
    ninja-build \
    nlohmann-json3-dev \
    pkg-config \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /build
COPY simgrid-runner ./simgrid-runner
RUN cmake -S simgrid-runner -B output -G Ninja -DCMAKE_BUILD_TYPE=Release \
 && cmake --build output --parallel

FROM golang:1.25-trixie AS go-builder

WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends gcc libsqlite3-dev \
 && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /output/akoflow-server ./cmd/server

FROM debian:trixie-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libsimgrid4.0 \
    libsqlite3-0 \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=go-builder /output/akoflow-server /usr/local/bin/akoflow-server
COPY --from=simgrid-builder /build/output/akoflow-simgrid-runner /usr/local/bin/akoflow-simgrid-runner

RUN mkdir -p /app/storage
EXPOSE 8080
CMD ["akoflow-server"]
