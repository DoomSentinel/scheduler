FROM golang:alpine AS builder

# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0

# Move to working directory /build
WORKDIR /build

# Copy the code into the container
COPY . .

RUN go mod download

# Build the application
RUN go build -o application main.go

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
RUN cp /build/application .

# Build a small image
FROM scratch

COPY --from=builder /dist/application /
COPY config.example.yaml config.yaml

# Command to run
ENTRYPOINT ["/application"]
