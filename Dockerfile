# Use the official Golang image as a builder
FROM golang:1.21-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
# Inject version info if available (e.g., via goreleaser or build args)
ARG VERSION=unknown
ARG COMMIT=unknown
ARG DATE=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o gitlab-mcp-server cmd/gitlab-mcp-server/main.go

# Start fresh from a smaller image
FROM alpine:latest

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/gitlab-mcp-server .

# Expose port (if needed for non-stdio communication later, otherwise informational)
# EXPOSE 8080 

# Set environment variables (defaults, can be overridden)
# GitLab Personal Access Token (PAT), Group or Project token for API authentication. Required.
ENV GITLAB_TOKEN=""
# Base URL for GitLab instance (e.g., "https://gitlab.example.com"). Defaults to "https://gitlab.com" if empty. Optional.
ENV GITLAB_HOST=""
# Comma-separated list of toolsets to enable (e.g., "issues,merge_requests"). Defaults to "all". Optional.
ENV GITLAB_TOOLSETS="all"
# Set to "1" or "true" to enable dynamic toolset discovery by the MCP host. Defaults to "0" (false). Optional.
ENV GITLAB_DYNAMIC_TOOLSETS="0"

# Command to run the executable using stdio communication
ENTRYPOINT ["./gitlab-mcp-server", "stdio"]

# Optionally add flags to ENTRYPOINT if needed later, e.g.:
# ENTRYPOINT ["./gitlab-mcp-server", "stdio", "--toolsets", "$GITLAB_TOOLSETS", "--gitlab-host", "$GITLAB_HOST"]
# Note: Using env vars is often preferred for container secrets/config. 