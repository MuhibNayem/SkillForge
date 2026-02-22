.PHONY: proto-gen proto-install build test migrate-create migrate-up migrate-down frontend-install frontend-build frontend-dev

SERVICES := auth user tenant course content enrollment assessment certification
DB_URL ?= "postgres://learnhub:password@localhost:5432/learnhub?sslmode=disable"

# ============================================================
# Protobuf Generation
# ============================================================
# Prerequisites: protoc, protoc-gen-go, protoc-gen-go-grpc
# Install them with: make proto-install

proto-install:
	@echo "Installing protobuf tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Ensure 'protoc' is installed: apt install protobuf-compiler"
	@echo "Ensure GOPATH/bin is in PATH: export PATH=\$$PATH:\$$(go env GOPATH)/bin"

proto-gen:
	@echo "Generating protobuf Go code from server/proto/*.proto → server/shared/pb/..."
	@cd server && export PATH=$$PATH:$$(go env GOPATH)/bin && \
	for proto in proto/*.proto; do \
		echo "  → $$proto"; \
		protoc \
			--go_out=shared --go_opt=module=github.com/amnayem/skillforge/shared \
			--go-grpc_out=shared --go-grpc_opt=module=github.com/amnayem/skillforge/shared \
			-I. -I./proto "$$proto"; \
	done
	@echo "Done. Generated code is in server/shared/pb/*/"

# ============================================================
# Backend (Go Microservices)
# ============================================================

build:
	@cd server && for svc in $(SERVICES); do \
		echo "Building $$svc..." && \
		(cd services/$$svc && go build -o /dev/null ./cmd/); \
	done
	@echo "All services built OK"

test:
	@cd server && for svc in $(SERVICES); do \
		echo "=== Testing $$svc ===" && \
		(cd services/$$svc && go test -v ./...); \
	done

tidy:
	@cd server && for dir in shared $(addprefix services/,$(SERVICES)); do \
		echo "Tidying $$dir..." && (cd $$dir && go mod tidy); \
	done

# ============================================================
# Frontend (Svelte 5 + pnpm)
# ============================================================

frontend-install:
	cd client && pnpm install

frontend-dev:
	cd client && pnpm dev

frontend-build:
	cd client && pnpm build

# ============================================================
# Database Migrations
# ============================================================

# Usage: make migrate-create name=init_schema
migrate-create:
	@mkdir -p server/db/migrations
	@migrate create -ext sql -dir server/db/migrations -seq $(name)

migrate-up:
	@migrate -path server/db/migrations -database $(DB_URL) up

migrate-down:
	@migrate -path server/db/migrations -database $(DB_URL) down

