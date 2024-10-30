FROM golang:1.23.2 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o scality-cosi-driver ./cmd/scality-cosi-driver

FROM gcr.io/distroless/static:latest
COPY --from=builder /app/scality-cosi-driver /scality-cosi-driver
ENTRYPOINT ["/scality-cosi-driver"]
