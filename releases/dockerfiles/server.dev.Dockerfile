FROM golang:1.25-trixie

# Development image: built once for tool dependencies. Application source is
# mounted by docker-compose.dev.yml and compiled by `go run`, so code changes
# never require rebuilding this image.
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates gcc libsqlite3-dev openssh-client rsync kubernetes-client \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app
