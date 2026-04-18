module github.com/learnhub/lms/server/services/discussion

go 1.19

require (
github.com/learnhub/lms/server/shared v0.0.0
github.com/redis/go-redis/v9 v9.5.1
go.mongodb.org/mongo-driver v1.13.4
google.golang.org/grpc v1.62.0
)

replace github.com/learnhub/lms/server/shared => ../../shared
