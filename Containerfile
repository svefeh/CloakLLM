# --- Stage 1: Build the Go binary ---
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the dependency files first (optimizes caching)
COPY go.mod ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Compile the binary statically (CGO_ENABLED=0 is required for minimal images)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o cloakllm-daemon ./cmd/cloakllm-daemon

# --- Stage 2: Create the minimal final image ---
FROM alpine:latest

# Install CA certificates (required for secure HTTPS upstream communication)
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy only the compiled binary from the builder stage
COPY --from=builder /app/cloakllm-daemon .

# Expose the default proxy port
EXPOSE 8080

# Run the daemon
CMD ["./cloakllm-daemon"]