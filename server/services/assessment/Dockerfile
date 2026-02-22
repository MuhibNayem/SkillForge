FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY shared/ ./shared/
COPY services/assessment/ ./services/assessment/

RUN cd services/assessment && go mod download
RUN cd services/assessment && CGO_ENABLED=0 GOOS=linux go build -o /bin/assessment-service ./cmd/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/assessment-service /bin/assessment-service
EXPOSE 9090
ENTRYPOINT ["/bin/assessment-service"]
