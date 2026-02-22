FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy shared module first (dependency)
COPY shared/ ./shared/

# Copy this service's module
COPY services/auth/ ./services/auth/

# Download dependencies
RUN cd services/auth && go mod download

# Build the service binary
RUN cd services/auth && CGO_ENABLED=0 GOOS=linux go build -o /bin/auth-service ./cmd/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/auth-service /bin/auth-service
EXPOSE 9090
ENTRYPOINT ["/bin/auth-service"]
