# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /go-salesforce-emulator ./cmd/go-salesforce-emulator

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /

COPY --from=builder /go-salesforce-emulator /go-salesforce-emulator

EXPOSE 8080

ENTRYPOINT ["/go-salesforce-emulator"]
CMD ["-port", "8080"]
