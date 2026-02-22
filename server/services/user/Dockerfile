FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY shared/ ./shared/
COPY services/user/ ./services/user/

RUN cd services/user && go mod download
RUN cd services/user && CGO_ENABLED=0 GOOS=linux go build -o /bin/user-service ./cmd/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/user-service /bin/user-service
EXPOSE 9090
ENTRYPOINT ["/bin/user-service"]
