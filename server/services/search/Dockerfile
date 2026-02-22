FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY shared/ ./shared/
COPY services/search/ ./services/search/

RUN cd services/search && go mod download
RUN cd services/search && CGO_ENABLED=0 GOOS=linux go build -o /bin/search-service ./cmd/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/search-service /bin/search-service
EXPOSE 9090
ENTRYPOINT ["/bin/search-service"]
