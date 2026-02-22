FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY shared/ ./shared/
COPY services/course/ ./services/course/

RUN cd services/course && go mod download
RUN cd services/course && CGO_ENABLED=0 GOOS=linux go build -o /bin/course-service ./cmd/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/course-service /bin/course-service
EXPOSE 9090
ENTRYPOINT ["/bin/course-service"]
