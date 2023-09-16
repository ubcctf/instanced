############################
# STEP 1 build binary
############################
FROM golang:alpine AS builder
RUN apk update && apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
# Fetch dependencies.
RUN go mod download
COPY ./* ./
# Build the binary.
RUN go get -v && GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/app
############################
# STEP 2 build image
############################
FROM scratch
# Copy ca-certs from builder since we need https
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Copy static binary
COPY --from=builder /go/bin/app /go/bin/app
# Run the binary.
ENTRYPOINT ["/go/bin/app"]