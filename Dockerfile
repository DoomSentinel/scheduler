FROM golang:alpine AS builder

# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0

# Move to working directory /build
WORKDIR /build

# Copy the code into the container
COPY . .

RUN go mod tidy

# Build the application
RUN go build -o application main.go

ARG RABTAP_VERSION=1.25

RUN apk --no-cache add --update curl unzip ca-certificates && update-ca-certificates

RUN curl -L https://github.com/jandelgado/rabtap/releases/download/v${RABTAP_VERSION}/rabtap-v${RABTAP_VERSION}-linux-amd64.zip --output rabtap.zip && \
    unzip rabtap.zip -d download && \
    mv download/bin/rabtap-linux-amd64 rabtap && \
    chmod +x rabtap

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
RUN cp /build/application .
RUN cp /build/rabtap .

# Build a small image
FROM alpine:3.12

COPY --from=builder /dist/application /
COPY --from=builder /dist/rabtap rabtap
COPY config.example.yaml config.yaml
COPY build/wait-for-rabbit.sh wait-for-rabbit.sh

# Command to run
CMD ["/application"]
