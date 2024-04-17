# Builder stage
FROM golang:1.22.1-alpine AS builder

# Add Maintainer info
LABEL maintainer="Belyakov Nikita"

# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git && apk add --no-cach bash && apk add build-base

# Setup folders
RUN mkdir /app
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o BashAPI ./app

# New stage from scratch for a smaller image
FROM alpine:latest

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/BashAPI .
COPY --from=builder /app/config ./config
COPY --from=builder /app/migrations ./migrations

# Expose port 8000 to the outside world
EXPOSE 8000
EXPOSE 5432

# Command to run the executable
CMD ["./BashAPI"]
