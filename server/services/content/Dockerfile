FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY shared/ ./shared/
COPY services/content/ ./services/content/

RUN cd services/content && go mod download
RUN cd services/content && CGO_ENABLED=0 GOOS=linux go build -o /bin/content-service ./cmd/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/content-service /bin/content-service
EXPOSE 9090
ENTRYPOINT ["/bin/content-service"]
