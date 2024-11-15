# Stage 1: Build the Go application and install Delve
FROM golang:1.23.3 AS builder

# Install Delve for debugging
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Set the working directory
WORKDIR /app

# Copy Go modules and install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app with debugging flags
RUN CGO_ENABLED=0 GOOS=linux go build -gcflags "all=-N -l" -o scality-cosi-driver ./cmd/scality-cosi-driver

# Stage 2: Create a minimal container with the built binary and Delve
FROM golang:1.23.3

WORKDIR /app

# Copy the Go binary and Delve debugger from the builder stage
COPY --from=builder /go/bin/dlv /dlv
COPY --from=builder /app/scality-cosi-driver /app/scality-cosi-driver

# Expose Delve debugger port
EXPOSE 2345

# Run the Go app with Delve in headless mode
CMD ["/dlv", "exec", "/app/scality-cosi-driver", "--headless", "--listen=:2345", "--api-version=2", "--accept-multiclient", "--log"]
