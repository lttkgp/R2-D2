FROM golang:alpine AS builder

# Add build dependencies
# update-ca-certificates will show a warning, which is safe to ignore.
# https://github.com/gliderlabs/docker-alpine/issues/52
RUN apk update && \
    apk add --no-cache make upx ncurses ca-certificates && \
    update-ca-certificates

# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Move to working directory /build
WORKDIR /build

# Copy the code into the container
# We don't need to fetch dependencies since vendor directory is checked in
COPY . .

# Build and compress the application
RUN make build
RUN make pack

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
RUN cp /build/bin/r2-d2 .

# Build a small image
FROM scratch

# Import from Builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /dist/r2-d2 .

# Export necessary port
EXPOSE 8080

# Command to run when starting the container
CMD ["/r2-d2"]