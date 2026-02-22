# LearnHub LMS Microservices Architecture

Based on `ARCHITECTURE.md`, the platform consists of **17 microservices** across four domains.

## Server Structure (Go 1.25 / gRPC)

Each microservice is an **independently deployable unit** with its own `go.mod`, `cmd/main.go`, and `Dockerfile`. Shared code (logging, config, errors, protobuf) lives in a `shared/` module referenced via Go `replace` directives and a `go.work` workspace.

```
server/
├── go.work                         # Go workspace (ties modules for local dev)
├── shared/                         # Shared module
│   ├── go.mod
│   ├── pkg/                        # logger, config, errors
│   └── pb/                         # Generated protobuf Go code
├── proto/                          # .proto source definitions
├── services/
│   ├── auth/                       # Each service = own go.mod + Dockerfile
│   │   ├── go.mod
│   │   ├── cmd/main.go
│   │   ├── handler.go
│   │   └── Dockerfile
│   ├── user/
│   ├── course/
│   ├── content/
│   └── enrollment/
└── db/migrations/
```

## Service Catalog

### Phase 1 — Core Services (6)

| # | Service | Domain | Database | Responsibility |
|---|---------|--------|----------|----------------|
| 1 | **Auth** | Core | PostgreSQL + Redis | Authentication, JWT, RBAC |
| 2 | **User** | Core | PostgreSQL | Profiles, groups, permissions |
| 3 | **Tenant** | Core | PostgreSQL | Multi-tenancy, billing |
| 4 | **Course** | Learning | PostgreSQL | Course CRUD, modules, lessons |
| 5 | **Content** | Learning | MongoDB + MinIO | Media uploads, SCORM/xAPI |
| 6 | **Enrollment** | Learning | PostgreSQL | Enrollments, progress tracking |

### Phase 2 — Growth Services (5)

| # | Service | Domain | Database | Responsibility |
|---|---------|--------|----------|----------------|
| 7 | **Assessment** | Learning | PostgreSQL | Quizzes, grading |
| 8 | **Certification** | Learning | PostgreSQL | Certificate generation |
| 9 | **Notification** | Engagement | Kafka + Redis | Email, push, in-app alerts |
| 10 | **Analytics** | Platform | Elasticsearch | Reports, dashboards |
| 11 | **Search** | Platform | Elasticsearch | Full-text search |

### Phase 3 — Scale Services (6)

| # | Service | Domain | Database | Responsibility |
|---|---------|--------|----------|----------------|
| 12 | **AI** | Platform | PostgreSQL + Vector DB | Recommendations, adaptive paths |
| 13 | **Discussion** | Engagement | MongoDB | Forums, comments |
| 14 | **Gamification** | Engagement | PostgreSQL | Badges, leaderboards, points |
| 15 | **Live Class** | Engagement | — | Zoom/Teams integration |
| 16 | **Integration** | Platform | PostgreSQL | Webhooks, connectors |
| 17 | **Audit** | Platform | PostgreSQL | Compliance logging |
