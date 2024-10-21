FROM golang:1.23.2 AS builder

WORKDIR /app

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o scality-cosi-driver ./cmd/scality-cosi-driver

FROM gcr.io/distroless/static:latest
COPY --from=builder /app/scality-cosi-driver /scality-cosi-driver
ENTRYPOINT ["/scality-cosi-driver"]
