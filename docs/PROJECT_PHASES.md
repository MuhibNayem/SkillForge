# LearnHub LMS — Project Phases

> A comprehensive, phase-by-phase implementation roadmap for the LearnHub LMS platform.
> Technology: Go 1.25 (gRPC), Svelte 5, PostgreSQL, MongoDB, Redis, Kafka, Kong, MinIO.
> Architecture: Per-service `go.mod` modules with Go Workspaces, event-driven via Kafka, multi-tenant (schema-per-tenant).

---

## Phase 1: Foundation (Months 1-6) — Core Platform MVP 🔧

**Goal:** Establish the base infrastructure and deliver the minimum viable learning experience: a user can register, an instructor can create a course, and a student can enroll and view content.

> **Status Legend:** ✅ Done — 🔧 Scaffolded (proto + handler, needs real DB) — 🔄 In Progress — ⬜ Not Started

### 1.1 Infrastructure & Repository ✅

| Item | Details |
|------|---------|
| **Monorepo** | `client/` (Svelte 5), `server/` (Go), `docs/`, `.github/` |
| **Go Modules** | Per-service `go.mod` with `shared/` module; `go.work` workspace |
| **Docker Compose** | PostgreSQL 15, MongoDB 7, Redis 7, Kafka (Confluent 7.5), Kong 3.5, MinIO, Zookeeper |
| **CI/CD** | GitHub Actions — matrix build per service, frontend lint/build |
| **Protobuf** | Central `.proto` definitions in `server/proto/`, generated Go code in `shared/pb/` |
| **Migrations** | `golang-migrate` with Makefile targets (`migrate-create`, `migrate-up`, `migrate-down`) |

### 1.2 API Gateway (Kong) ✅

- Declarative config (`kong.yaml`) with `grpc-gateway` plugin for REST ↔ gRPC translation
- Global plugins: CORS, Rate Limiting (500 req/min), request logging
- Per-service routing: `/api/v1/auth/*`, `/api/v1/users/*`, `/api/v1/courses/*`
- JWT validation plugin for protected routes (validates tokens issued by Auth Service)

### 1.3 Core Microservices (6 services) ✅

> All 6 services have: `.proto` definitions, generated gRPC code, **Repository interface + PostgreSQL/MongoDB implementation**, handler with DI, `cmd/main.go` with real DB connections, `go.mod`, Dockerfile. **23 unit tests passing.**

#### Auth Service ✅
- **Responsibility:** Authentication, authorization, session management
- **Database:** PostgreSQL (shared schema) + Redis (sessions)
- **Key APIs:**
  - `POST /auth/register` — Email/password registration with bcrypt hashing
  - `POST /auth/login` — JWT access + refresh token issuance (HS256, 24h expiry)
  - `POST /auth/refresh` — Refresh token rotation
  - `POST /auth/forgot-password` / `POST /auth/reset-password`
  - `GET /auth/me` — Current user from JWT claims
  - `POST /auth/mfa/enable` / `POST /auth/mfa/verify` — TOTP-based MFA
- **RBAC Roles:** `super_admin`, `tenant_admin`, `instructor`, `student`
- **Events Published:** `user.registered`, `user.logged_in`

#### User Service ✅
- **Responsibility:** User profiles, groups, permissions
- **Database:** PostgreSQL (tenant schema)
- **Key APIs:**
  - `GET/POST /users` — List/create users (admin)
  - `GET/PUT/DELETE /users/{id}` — Profile CRUD
  - `POST /users/bulk-import` — CSV/Excel bulk user import
  - `GET/POST /users/{id}/groups` — Group membership management
- **Events Published:** `user.updated`, `user.deleted`

#### Tenant Service ✅
- **Responsibility:** Multi-tenancy, onboarding, billing metadata
- **Database:** PostgreSQL (shared schema)
- **Key APIs:**
  - `POST /tenants` — Create new tenant (provisions schema)
  - `GET /tenants/{id}` — Tenant config and metadata
  - `PUT /tenants/{id}/settings` — Theme, branding, feature flags
- **Multi-tenancy Strategy:** Schema-per-tenant in PostgreSQL; tenant resolved from JWT `tenant_id` claim
- **Events Published:** `tenant.created`, `tenant.updated`

#### Course Service ✅
- **Responsibility:** Course CRUD, module/lesson structuring, publishing workflow
- **Database:** PostgreSQL (tenant schema)
- **Key APIs:**
  - `GET/POST /courses` — List/create courses
  - `GET/PUT/DELETE /courses/{id}` — Course CRUD
  - `POST /courses/{id}/modules` — Add module to course
  - `POST /courses/{id}/modules/{mid}/lessons` — Add lesson to module
  - `PUT /courses/{id}/publish` — Publish course (draft → active)
- **Data Model:** Course → Modules (ordered) → Lessons (ordered, type: video/text/quiz)
- **Events Published:** `course.created`, `course.published`

#### Content Service ✅
- **Responsibility:** Media upload, storage, transcoding metadata
- **Database:** MongoDB (metadata) + MinIO (file storage)
- **Key APIs:**
  - `POST /content/upload` — Multipart file upload → MinIO bucket
  - `GET /content/{id}` — Retrieve content metadata + presigned download URL
  - `DELETE /content/{id}` — Soft-delete content
- **Supported Formats:** Video (MP4, WebM), Audio (MP3), Documents (PDF), Images (PNG, JPG)
- **MinIO Buckets:** `tenant-{id}/courses/{course_id}/` prefix-based isolation
- **Events Published:** `content.uploaded`, `content.deleted`

#### Enrollment Service ✅
- **Responsibility:** Course enrollment, progress tracking, completion
- **Database:** PostgreSQL (tenant schema)
- **Key APIs:**
  - `POST /enrollments` — Enroll student in course
  - `GET /enrollments?user_id=X` — List user's enrollments
  - `PUT /enrollments/{id}/progress` — Update lesson progress (% complete)
  - `GET /enrollments/{id}/progress` — Get detailed progress per module/lesson
- **Completion Logic:** Course marked complete when all required lessons reach 100%
- **Events Published:** `enrollment.created`, `course.completed`

### 1.4 Frontend Web Application (Svelte 5) 🔧

#### Design System
- Built with **Tailwind CSS v4** (pnpm)
- Dark mode with glassmorphism design, responsive breakpoints (mobile-first)
- Component library: Button, Input, Card, DataTable — **still needed:** Toast, Modal, Sidebar as reusable components

#### Authentication Pages ✅
- Login page with email/password (MFA code input — ⬜)
- Registration page with role selection (Student/Instructor)
- Forgot/Reset password flow — ⬜

#### Instructor Dashboard 🔧
- **Course Builder:** Dynamic module/lesson creation with type selection (drag-and-drop ordering — ⬜)
- **Content Uploader:** ⬜ (needs MinIO integration)
- **Student Progress View:** ⬜ (needs Enrollment Service DB wiring)

#### Student Dashboard 🔧
- **Course Catalog:** Searchable grid with category filters ✅ (static mock data, needs API integration)
- **Course Player:** ⬜ (needs Content Service + HLS.js)
- **My Courses:** Progress bars with enrolled courses ✅ (static mock data, needs API integration)

### 1.5 Testing Strategy (Phase 1) 🔧
- **Backend:** 23 Go unit tests passing (handler-level with mock repository DI); integration tests with testcontainers — ⬜
- **Frontend:** Vitest + Playwright — ⬜
- **Target:** >80% unit test coverage for business logic

> **Remaining for full Phase 1:**
> 1. Frontend API integration (replace static mock data with fetch calls)
> 2. Course player (HLS.js video, markdown renderer)
> 3. Integration tests (testcontainers)
> 4. `docker-compose up` → full E2E verification

---

## Phase 2: Growth (Months 7-12) — Expanding Core Features ⬜

**Goal:** Enrich the learning experience with assessments, certifications, notifications, analytics, search, and initial AI capabilities. Prepare for mobile.

### 2.1 New Microservices (5 services)

#### Assessment Service ⬜
- **Responsibility:** Quiz creation, assignment management, auto/manual grading
- **Database:** PostgreSQL (tenant schema)
- **Key APIs:**
  - `POST /assessments` — Create quiz/assignment (linked to a lesson)
  - `GET /assessments/{id}` — Retrieve assessment with questions
  - `POST /assessments/{id}/submissions` — Student submits answers
  - `GET /assessments/{id}/submissions/{sid}/grade` — Get grade
- **Question Types:** Multiple choice, true/false, short answer, essay, file upload
- **Auto-Grading:** MC/TF graded instantly; essay/file queued for instructor review
- **Events Published:** `assessment.submitted`, `assessment.graded`

#### Certification Service ⬜
- **Responsibility:** Certificate generation, template management, verification
- **Database:** PostgreSQL (tenant schema)
- **Key APIs:**
  - `POST /certificates/templates` — Create certificate template (instructor)
  - `GET /certificates?user_id=X` — List user's certificates
  - `GET /certificates/{id}/verify` — Public verification endpoint
- **Generation:** Triggered by `course.completed` Kafka event; PDF with unique verification code
- **Templates:** Customizable name, course, date, instructor signature fields

#### Notification Service ⬜
- **Responsibility:** Multi-channel notifications (email, in-app, push)
- **Database:** PostgreSQL + Redis (delivery status)
- **Architecture:** Pure Kafka consumer — no REST API (event-driven only)
- **Consumed Events:** `user.registered`, `course.published`, `assessment.graded`, `enrollment.created`, `course.completed`
- **Channels:** SMTP (email via SendGrid/SES), WebSocket (in-app), FCM (push — Phase 2.3)
- **User Preferences:** Per-user notification settings (opt-in/out per channel per event type)

#### Analytics Service ⬜
- **Responsibility:** Learning analytics, executive dashboards, engagement metrics
- **Database:** Elasticsearch (aggregations)
- **Key APIs:**
  - `GET /analytics/courses/{id}/overview` — Enrollment count, completion rate, avg score
  - `GET /analytics/users/{id}/learning` — Time spent, courses completed, streaks
  - `GET /analytics/tenant/dashboard` — DAU/MAU, top courses, revenue metrics
- **Data Pipeline:** Kafka consumer → transform → Elasticsearch indices
- **Dashboards:** Pre-built queries for Student, Instructor, Admin, Executive personas

#### Search Service ⬜
- **Responsibility:** Full-text search across courses, content, and users
- **Database:** Elasticsearch
- **Key APIs:**
  - `GET /search?q=X&type=course|content|user` — Global search with type filters
  - `GET /search/suggestions?q=X` — Autocomplete suggestions
- **Indexing:** Kafka consumer listens for `course.created`, `course.updated`, `content.uploaded` events
- **Features:** Fuzzy matching, highlighting, faceted filters (category, difficulty, rating)

### 2.2 AI Service V1 (Z.AI GLM-4.7-Flash + LangChain) ⬜

> **Reference:** [Z.AI LangChain Integration Guide](https://docs.z.ai/guides/develop/langchain/introduction)

- **LLM Provider:** [Z.AI](https://z.ai) — GLM-4.6-Flash model
  - MoE (Mixture of Experts) architecture, 200K token context window
  - Optimized for coding, agentic workflows, and text generation
  - **OpenAI-compatible API** — accessed via `langchain-openai` package (NOT `langchain-community`)
  - API Base URL: `https://api.z.ai/api/paas/v4/`
  - API Key env var: `ZAI_API_KEY` (obtain from [Z.AI API Keys](https://z.ai/manage-apikey/apikey-list))

- **Install Dependencies:**
  ```bash
  pip install langchain langchain-openai langchainhub httpx_sse langgraph
  ```

- **Basic Configuration:**
  ```python
  import os
  from langchain_openai import ChatOpenAI

  llm = ChatOpenAI(
      temperature=0.7,
      model="glm-4.7-flash",
      openai_api_key=os.getenv("ZAI_API_KEY"),
      openai_api_base="https://api.z.ai/api/paas/v4/",
  )
  ```

- **Capabilities (V1):**
  - **Course Recommendations:** LangChain chain (prompt template + `ChatOpenAI`) that takes enrollment history + completion patterns → personalized "next course" suggestions
  - **Content Auto-Tagging:** GLM-4.6-Flash classifies uploaded content into categories/difficulty via structured output parsing
  - **Course Description Generation:** Instructors auto-generate course descriptions and learning objectives from lesson outlines
  - **Streaming Responses:** Real-time token streaming for chat-like interactions:
    ```python
    from langchain.callbacks.streaming_stdout import StreamingStdOutCallbackHandler

    llm = ChatOpenAI(
        model="glm-4.7-flash",
        openai_api_key=os.getenv("ZAI_API_KEY"),
        openai_api_base="https://api.z.ai/api/paas/v4/",
        streaming=True,
        callbacks=[StreamingStdOutCallbackHandler()],
    )
    ```

- **Deployment:** Python FastAPI service, deployed as its own pod, accessed via Kong REST proxy at `/api/v1/ai/*`
- **Key APIs:**
  - `POST /ai/recommendations` — Personalized course suggestions for a user
  - `POST /ai/content/tag` — Auto-tag content with categories and difficulty
  - `POST /ai/content/summarize` — Generate summaries for course descriptions

- **Z.AI Best Practices (from official docs):**
  - Enable LangChain caching mechanism to reduce API calls
  - Use batch processing for bulk content tagging
  - Set reasonable `max_tokens` limits per request
  - Use async processing (`agenerate`) for better concurrency
  - Implement retry with exponential backoff and timeout handling
  - Use `ConversationBufferWindowMemory` to limit history length for tutor conversations
  - Store `ZAI_API_KEY` in environment variables, rotate regularly

### 2.3 Mobile & Frontend Enhancements ⬜
- Full responsive optimization of Svelte web app
- Advanced course player: playback speed control (0.5x–2x), resume from last position, keyboard shortcuts
- Offline download preparation (service worker caching strategy)
- React Native scaffolding for iOS/Android with shared API client

### 2.4 Integrations & Admin Tooling ⬜
- **Enterprise SSO:** SAML 2.0 integration in Auth Service (Okta, Azure AD, OneLogin)
- **Admin Dashboard:** Tenant management (create/suspend/configure), user management, billing overview, system health
- **Webhook System:** Configurable webhooks for key events (`course.completed`, `enrollment.created`); admin UI for webhook CRUD with secret signing and retry logic

### 2.5 AI-Enhanced Learning (AI Service V1.5) ⬜

> **Goal:** Elevate the AI service from content-utility endpoints to interactive, student-facing learning features. Uses existing infrastructure (Redis for session memory, Elasticsearch for vector search).

#### AI Tutor Chatbot ⬜
- **Responsibility:** Per-course conversational tutor that answers student questions in context
- **Architecture:** Stateful chat with session-based memory stored in Redis
- **Key APIs:**
  - `POST /ai/tutor/chat` — Send message, receive AI response (supports streaming via SSE)
  - `DELETE /ai/tutor/chat/{session_id}` — Clear session history
- **Implementation:**
  - Modern LCEL with `RunnableWithMessageHistory` (replaces deprecated `ConversationBufferMemory`)
  - System prompt dynamically injected with course name, lesson context, and instructor guidelines
  - Conversation window limited to last 20 messages to control token usage
  - Socratic questioning mode: AI guides students toward answers rather than giving them directly
- **Integration:** Embedded in the Svelte course player as a slide-out chat panel

#### Q&A over Course Materials (RAG) ⬜
- **Responsibility:** Answer student questions using actual course content (videos transcripts, PDFs, lesson text)
- **Architecture:** RAG (Retrieval-Augmented Generation) pipeline
- **Key APIs:**
  - `POST /ai/search/semantic` — Semantic search over course content with cited sources
- **Implementation:**
  - **Embeddings:** Generate vector embeddings of course content chunks using Z.AI embedding API
  - **Vector Store:** Elasticsearch with dense vector fields (leverages existing ES instance — no new infra)
  - **RAG Pipeline:** Student query → embed → ES similarity search (top-5 chunks) → inject into GLM prompt → generate contextual answer with source citations
  - **Indexing:** Triggered by `content.uploaded` Kafka event — content is chunked, embedded, and indexed automatically
- **Chunking Strategy:** 512-token chunks with 50-token overlap; metadata preserved (course_id, lesson_id, content_type)

#### Auto Quiz Generation ⬜
- **Responsibility:** Generate quiz questions from lesson content for instructors to review and publish
- **Key APIs:**
  - `POST /ai/quiz/generate` — Generate quiz from lesson content
- **Implementation:**
  - One-shot LCEL chain (same pattern as existing endpoints — no memory needed)
  - Input: lesson text/transcript + desired question count + question types (MCQ, true/false, short answer)
  - Output: Structured JSON matching Assessment Service payload format (ready to import)
  - Instructor reviews and edits generated questions before publishing
- **Integration:** "Generate Quiz with AI" button in the instructor course builder UI

#### AI-Assisted Grading Hints ⬜
- **Responsibility:** Provide grading suggestions for essay/file-upload assignments to reduce instructor workload
- **Key APIs:**
  - `POST /ai/grading/suggest` — Analyze student submission against rubric, return suggested score + feedback
- **Implementation:**
  - Input: student submission text + rubric criteria + max points per criterion
  - Output: Per-criterion score suggestion + written feedback + overall confidence score
  - **Final grade is always set by the instructor** — AI provides hints only, never auto-grades essays
  - Flagged submissions (low confidence or plagiarism indicators) highlighted for closer review
- **Integration:** Grading panel in Assessment Service UI shows AI suggestions alongside the submission

#### Infrastructure Requirements
- **Redis:** Already available — used for chat session memory (key: `tutor:{session_id}`, TTL: 24h)
- **Elasticsearch:** Already available — add dense vector field to existing content index for semantic search
- **No new infrastructure** required for this phase

### 2.6 Testing Strategy (Phase 2) ⬜
- **Performance Testing:** K6 load tests targeting <200ms P95 API latency under 1,000 concurrent users
- **Contract Testing:** Protobuf compatibility checks in CI (buf breaking)
- **E2E Expansion:** Playwright flows for assessment submission, certificate download, notification delivery

---

## Phase 3: Scale (Months 13-18) — Advanced Features & True Scalability ⬜

**Goal:** Deliver advanced AI, social learning, live classes, full compliance, and production-grade cloud infrastructure.

### 3.1 AI Service V2 — Agentic Workflows (LangGraph + GLM-4.7-Flash) ⬜

Major upgrade from simple chains to **stateful, multi-step AI agents** using **LangGraph**.

- **Model:** Z.AI GLM-4.6-Flash (same OpenAI-compatible API provider)
- **Framework:** **LangGraph** for stateful agent orchestration on top of LangChain
- **Architecture:**
  ```
  Student/Instructor Request
      → Kong API Gateway
          → AI Service (FastAPI)
              → LangGraph Agent (stateful graph)
                  → ChatOpenAI(model="glm-4.7-flash", openai_api_base="https://api.z.ai/api/paas/v4/")
                  → Tool Nodes (DB queries, Search, Content APIs)
                  → Response
  ```

- **LangGraph Agent Capabilities:**

  | Feature | Agent Graph Design |
  |---------|--------------------|
  | **Adaptive Learning Paths** | Multi-step agent: assess student performance → query course catalog → reorder lessons by difficulty → return personalized path |
  | **At-Risk Prediction** | Scheduled agent: gather engagement data → analyze patterns with GLM → classify risk level → trigger notification if high-risk |
  | **AI Quiz Generation** | Tool-using agent: retrieve lesson content → generate questions with GLM → validate answers → format as Assessment Service payload |
  | **Conversational Tutor** | Persistent-state agent: maintain conversation history → RAG over course content (vector search) → answer questions in context |

- **Intelligent Agent (ReAct pattern from Z.AI docs):**
  ```python
  import os
  from langchain import hub
  from langchain.agents import AgentExecutor, create_react_agent
  from langchain_openai import ChatOpenAI

  llm = ChatOpenAI(
      model="glm-4.7-flash",
      openai_api_key=os.getenv("ZAI_API_KEY"),
      openai_api_base="https://api.z.ai/api/paas/v4/",
  )

  # Define custom tools (DB lookups, content retrieval, etc.)
  tools = [course_search_tool, assessment_query_tool, content_retrieval_tool]

  prompt = hub.pull("hwchase17/react")
  agent = create_react_agent(llm, tools, prompt)
  agent_executor = AgentExecutor(agent=agent, tools=tools, verbose=True)

  result = agent_executor.invoke({"input": "Generate a quiz for lesson 42"})
  ```

- **Conversational Tutor with Memory (from Z.AI docs):**
  ```python
  from langchain_openai import ChatOpenAI
  from langchain.prompts import (
      ChatPromptTemplate, MessagesPlaceholder,
      SystemMessagePromptTemplate, HumanMessagePromptTemplate,
  )
  from langchain.chains import LLMChain
  from langchain.memory import ConversationBufferWindowMemory

  llm = ChatOpenAI(
      temperature=0.7,
      model="glm-4.7-flash",
      openai_api_key=os.getenv("ZAI_API_KEY"),
      openai_api_base="https://api.z.ai/api/paas/v4/",
  )

  prompt = ChatPromptTemplate(messages=[
      SystemMessagePromptTemplate.from_template(
          "You are a helpful learning tutor for the course: {course_name}. "
          "Answer questions based on the provided lesson content. "
          "Use the Socratic method when appropriate."
      ),
      MessagesPlaceholder(variable_name="chat_history"),
      HumanMessagePromptTemplate.from_template("{question}"),
  ])

  memory = ConversationBufferWindowMemory(
      memory_key="chat_history", return_messages=True, k=20
  )

  tutor = LLMChain(llm=llm, prompt=prompt, memory=memory)
  response = tutor.invoke({"course_name": "Intro to ML", "question": "What is gradient descent?"})
  ```

- **Semantic Search & RAG:**
  - Vector database: **Pinecone** for storing embeddings of course content
  - RAG pipeline: User query → embed → Pinecone similarity search → inject top-K results into GLM prompt → generate contextual answer

- **LangGraph Stateful Agent (Adaptive Path):**
  ```python
  from langgraph.graph import StateGraph, END
  from langchain_openai import ChatOpenAI

  llm = ChatOpenAI(
      model="glm-4.7-flash",
      openai_api_key=os.getenv("ZAI_API_KEY"),
      openai_api_base="https://api.z.ai/api/paas/v4/",
  )

  def assess_student(state):
      # Query assessment scores from Assessment Service
      ...

  def generate_path(state):
      # Use GLM to reorder lessons based on performance
      response = llm.invoke(
          f"Reorder these lessons for a student who scored {state['scores']}..."
      )
      ...

  graph = StateGraph()
  graph.add_node("assess", assess_student)
  graph.add_node("generate", generate_path)
  graph.add_edge("assess", "generate")
  graph.add_edge("generate", END)
  agent = graph.compile()
  ```

- **New APIs (V2):**
  - `POST /ai/learning-path` — Generate adaptive learning path for a student
  - `POST /ai/risk-assessment` — Predict at-risk students in a course
  - `POST /ai/quiz/generate` — Auto-generate quiz from lesson content
  - `POST /ai/tutor/chat` — Conversational tutor (stateful, per-session)
  - `POST /ai/search/semantic` — Semantic search over all course content

### 3.2 New Microservices (5 services)

#### Discussion Service ⬜
- **Responsibility:** Threaded forums, lesson comments, Q&A
- **Database:** MongoDB (document-based threading)
- **Key APIs:**
  - `GET/POST /courses/{id}/discussions` — Course-level discussion threads
  - `GET/POST /lessons/{id}/comments` — Lesson-level inline comments
  - `POST /discussions/{id}/replies` — Threaded replies with @mentions
- **Features:** Rich text (Markdown), upvoting, instructor-pinned answers, moderation tools

#### Gamification Service ⬜
- **Responsibility:** Points, badges, leaderboards, learning streaks
- **Database:** PostgreSQL + Redis (leaderboard sorted sets)
- **Key APIs:**
  - `GET /gamification/users/{id}/profile` — Points, badges, streak info
  - `GET /gamification/leaderboard?scope=course|tenant|global` — Ranked leaderboard
- **Point Triggers:** Lesson complete (+10), quiz passed (+25), course complete (+100), streak day (+5)
- **Badges:** "First Course", "Perfect Score", "7-Day Streak", "Top Learner", custom tenant badges

#### Live Class Service ⬜
- **Responsibility:** Live virtual classroom scheduling and integration
- **Database:** PostgreSQL
- **Key APIs:**
  - `POST /live-classes` — Schedule a live session (linked to course/module)
  - `GET /live-classes/{id}/join` — Generate join URL (Zoom/Teams)
- **Integrations:** Zoom API (create/manage meetings), Microsoft Graph API (Teams meetings)
- **Features:** Calendar sync, attendance tracking, recording links saved to Content Service

#### Integration Service ⬜
- **Responsibility:** Webhooks, third-party connectors, data import/export
- **Database:** PostgreSQL
- **Key APIs:**
  - `GET/POST /integrations/webhooks` — CRUD for webhook subscriptions
  - `POST /integrations/import/scorm` — Import SCORM 1.2/2004 packages
  - `GET /integrations/export/xapi` — Export xAPI statements
- **Connectors:** HRIS (BambooHR, Workday), SIS (Banner, PeopleSoft), Slack notifications

#### Audit Service ⬜
- **Responsibility:** Immutable compliance logging for all critical actions
- **Database:** PostgreSQL (append-only audit log table)
- **Architecture:** Pure Kafka consumer — subscribes to all service events
- **Logged Actions:** User login/logout, role changes, course publish/unpublish, grade modifications, data exports
- **Compliance:** SOC 2 Type II, GDPR (right to erasure tracking), FERPA (educational records access log)
- **Retention:** Configurable per-tenant (default: 7 years)

### 3.3 Infrastructure Scaling ⬜
- **Cloud Migration:** Terraform IaC for AWS — EKS (Kubernetes), RDS (PostgreSQL), ElastiCache (Redis), MSK (Kafka), S3 (MinIO replacement in prod)
- **Kubernetes:** HPA (CPU/memory based), resource limits/requests per pod, pod disruption budgets
- **Database:** Read replicas for PostgreSQL, connection pooling (PgBouncer), schema-per-tenant optimization
- **CDN:** CloudFront for static assets and MinIO/S3 media distribution
- **Multi-Region:** Active-passive setup in 2 AWS regions (us-east-1 primary, eu-west-1 DR)

### 3.4 Observability Stack ⬜
- **Metrics:** Prometheus + Grafana dashboards (per-service latency, error rates, saturation)
- **Logs:** Loki for centralized log aggregation with structured JSON logging
- **Traces:** Tempo for distributed tracing across gRPC service calls
- **Alerts:** PagerDuty integration for P1/P2 incidents (latency >500ms, error rate >1%, pod restarts)
- **SLOs:** 99.95% availability, <200ms P95 API latency, <2s page load

### 3.5 Testing Strategy (Phase 3) ⬜
- **Chaos Engineering:** Chaos Mesh for pod failure injection, network partition testing
- **Load Testing:** K6 at 10,000 concurrent users, targeting <200ms P95
- **Security:** OWASP ZAP automated scans, dependency vulnerability scanning (Snyk)
- **Compliance Audit:** SOC 2 readiness assessment, GDPR data flow mapping

---

## Phase 4: Innovation (Months 19-24) — Differentiators ⬜

**Goal:** Introduce cutting-edge features that differentiate LearnHub as a market leader.

### 4.1 Immersive Learning ⬜
- VR/AR content support in Content Service (WebXR-compatible viewer in course player)
- 360° video playback for virtual field trips
- Interactive 3D model viewer for technical training

### 4.2 Advanced Credentials ⬜
- **Blockchain Credentials:** Verifiable digital badges and certificates on Ethereum/Polygon
- **Open Badges 3.0:** Compliance with IMS Global Open Badges standard
- **Digital Wallets:** Students can export credentials to LinkedIn, digital wallet apps

### 4.3 Next-Gen Interfaces ⬜
- **Voice Interface:** Alexa Skill / Google Action for course progress queries, quiz practice
- **Conversational AI Tutor (Production):** Full LangGraph persistent-state agent embedded in the Svelte course player; uses `ChatOpenAI(model="glm-4.7-flash", openai_api_base="https://api.z.ai/api/paas/v4/")` with RAG over course content via Pinecone; supports follow-up questions, Socratic questioning mode, `ConversationBufferWindowMemory` for session continuity, and streaming responses
- **Accessibility:** WCAG 2.1 AA compliance audit and remediation, screen reader optimization

### 4.4 Marketplace & Extensibility ⬜
- **Content Marketplace:** Third-party instructors can publish and sell courses (revenue sharing model)
- **Plugin System:** REST-based plugin API for extending functionality (custom grading, integrations)
- **White-Labeling:** Custom domains, theme engine (CSS variables), logo/branding, email templates
- **API Marketplace:** Public REST API with API key management, rate limiting tiers, developer portal

### 4.5 Testing Strategy (Phase 4) ⬜
- **Accessibility Testing:** axe-core automated checks, manual screen reader testing
- **Penetration Testing:** Third-party security audit
- **Beta Program:** 50-tenant beta rollout with structured feedback collection
