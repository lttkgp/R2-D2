FROM golang:alpine AS builder

# Add build dependencies
RUN apk update && apk add make ncurses

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

# Build the application
RUN make build

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
RUN cp /build/bin/r2-d2 .

# Build a small image
FROM scratch

COPY --from=builder /dist/r2-d2 .

# Export necessary port
EXPOSE 8080

# Command to run when starting the container
CMD ["/r2-d2"]