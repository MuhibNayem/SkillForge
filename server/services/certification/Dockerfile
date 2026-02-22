FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY shared/ ./shared/
COPY services/certification/ ./services/certification/

RUN cd services/certification && go mod download
RUN cd services/certification && CGO_ENABLED=0 GOOS=linux go build -o /bin/certification-service ./cmd/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/certification-service /bin/certification-service
EXPOSE 9090
ENTRYPOINT ["/bin/certification-service"]
