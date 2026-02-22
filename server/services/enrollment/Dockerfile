FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY shared/ ./shared/
COPY services/enrollment/ ./services/enrollment/

RUN cd services/enrollment && go mod download
RUN cd services/enrollment && CGO_ENABLED=0 GOOS=linux go build -o /bin/enrollment-service ./cmd/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/enrollment-service /bin/enrollment-service
EXPOSE 9090
ENTRYPOINT ["/bin/enrollment-service"]
