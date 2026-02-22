FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY shared/ ./shared/
COPY services/tenant/ ./services/tenant/

RUN cd services/tenant && go mod download
RUN cd services/tenant && CGO_ENABLED=0 GOOS=linux go build -o /bin/tenant-service ./cmd/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/tenant-service /bin/tenant-service
EXPOSE 9090
ENTRYPOINT ["/bin/tenant-service"]
