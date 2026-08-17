FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/akoflow-observer ./cmd/akoflow-observer
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/akoflow-observer ./cmd/akoflow-observer

FROM scratch
COPY --from=build /out/akoflow-observer /akoflow-observer
ENTRYPOINT ["/akoflow-observer"]
