FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY shared/ ./shared/
COPY services/notification/ ./services/notification/

RUN cd services/notification && go mod download
RUN cd services/notification && CGO_ENABLED=0 GOOS=linux go build -o /bin/notification-service ./cmd/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/notification-service /bin/notification-service
EXPOSE 9090
ENTRYPOINT ["/bin/notification-service"]
