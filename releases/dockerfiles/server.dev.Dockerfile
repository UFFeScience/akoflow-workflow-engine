FROM golang:1.25-trixie AS apptainer-builder

ARG APPTAINER_VERSION=v1.5.3
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential git libseccomp-dev libglib2.0-dev libgpgme-dev \
    libssl-dev libnss3-dev uuid-dev \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /src
RUN git clone --depth 1 --branch ${APPTAINER_VERSION} https://github.com/apptainer/apptainer.git . \
 && ./mconfig --prefix=/usr/local --with-suid \
 && make -C builddir -j"$(nproc)" \
 && make -C builddir install

FROM golang:1.25-trixie

# Development image: built once for tool dependencies. Application source is
# mounted by docker-compose.dev.yml and compiled by `go run`, so code changes
# never require rebuilding this image.
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates gcc libsqlite3-dev libseccomp2 libgpgme11 libfuse3-4 \
    openssh-client rsync kubernetes-client squashfs-tools uidmap \
 && rm -rf /var/lib/apt/lists/*

COPY --from=apptainer-builder /usr/local /usr/local

WORKDIR /app
