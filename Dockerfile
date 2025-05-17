FROM golang:1.18-alpine AS builder

# Install necessary build dependencies
RUN apk --no-cache add ca-certificates git openssh-client

# Create a non-root user for building
RUN adduser -D -u 10001 lessonuser

# Copy only necessary files for building
COPY go.mod go.sum /go/src/lessoncraft/
WORKDIR /go/src/lessoncraft

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . /go/src/lessoncraft/

# Generate SSH key with proper permissions
RUN ssh-keygen -N "" -t rsa -f /etc/ssh/ssh_host_rsa_key >/dev/null && \
    chmod 600 /etc/ssh/ssh_host_rsa_key

# Build the application with security flags
RUN CGO_ENABLED=0 go build -a -installsuffix nocgo -ldflags="-w -s" -o /go/bin/lessoncraft .


FROM alpine:3.18

# Add CA certificates and create necessary directories
RUN apk --no-cache add ca-certificates && \
    mkdir -p /app/pwd && \
    adduser -D -u 10001 lessonuser && \
    chown -R lessonuser:lessonuser /app

# Copy the binary and SSH key from the builder stage
COPY --from=builder /go/bin/lessoncraft /app/lessoncraft
COPY --from=builder /etc/ssh/ssh_host_rsa_key /etc/ssh/ssh_host_rsa_key

# Set proper permissions
RUN chmod 600 /etc/ssh/ssh_host_rsa_key && \
    chown lessonuser:lessonuser /etc/ssh/ssh_host_rsa_key && \
    chmod 755 /app/lessoncraft

# Switch to non-root user
USER lessonuser
WORKDIR /app

# Add health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:3000/health || exit 1

# Run the application
CMD ["./lessoncraft"]

EXPOSE 3000
