FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY shared/ ./shared/
COPY services/analytics/ ./services/analytics/

RUN cd services/analytics && go mod download
RUN cd services/analytics && CGO_ENABLED=0 GOOS=linux go build -o /bin/analytics-service ./cmd/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/analytics-service /bin/analytics-service
EXPOSE 9090
ENTRYPOINT ["/bin/analytics-service"]
