# Todo List Service

![Go](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go&logoColor=white)
![Echo](https://img.shields.io/badge/Echo-v4-00ADD8?style=flat&logo=go&logoColor=white)
![gRPC](https://img.shields.io/badge/gRPC-Protobuf-4285F4?style=flat&logo=google&logoColor=white)
![Kafka](https://img.shields.io/badge/Apache_Kafka-231F20?style=flat&logo=apachekafka&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-DC382D?style=flat&logo=redis&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?style=flat&logo=postgresql&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus-E6522C?style=flat&logo=prometheus&logoColor=white)
![Grafana](https://img.shields.io/badge/Grafana-F46800?style=flat&logo=grafana&logoColor=white)
![coverage](https://img.shields.io/badge/coverage-78.3%25-brightgreen?style=flat&logo=go)


## Architecture

```
+-----------------------------------------------------+
|               Client                                |
|   HTTP (Echo :8080)   gRPC (:50051)                 |
+--------+------------------+-------------------------+
         |                  |
         v                  v
+---------------------------------+
|         TodoListService         |
|  - validation                   |
|  - cache-aside (Redis)          |
|  - audit events (Kafka)         |
+----+-----------+----------------+
     |           |
     v           v
PostgreSQL     Redis
(GORM)         (cache, TTL 2min)
                    |
              Kafka Producer
              (IBM/sarama)
                    |
              Kafka Consumer --> audit_events table
              (ConsumerGroup)   (PostgreSQL)
```

**Data flow on write:**
1. Request -> service validates input
2. Write to PostgreSQL
3. Send audit event to Kafka topic `todos`
4. Invalidate Redis cache key `todos:all`

**Data flow on read (GetAll):**
1. Check Redis `todos:all`
2. Cache hit -> return immediately
3. Cache miss -> query PostgreSQL -> cache result for 2 minutes

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.24 |
| HTTP Framework | Echo v4 |
| gRPC | Protobuf |
| Message Broker | Apache Kafka (IBM/sarama) |
| Database | PostgreSQL 15 (GORM) |
| Cache | Redis |
| Observability | Prometheus + Grafana |
| Config | godotenv |

## Project Structure

```
todo_list/
├── cmd/api/                 # Entrypoint + .env
├── config/                  # DB DSN builder
├── internal/
│   ├── app/                 # Wiring: DB, Redis, Kafka, HTTP, gRPC
│   ├── database/            # PostgreSQL connection
│   ├── handlers/            # Echo HTTP handlers
│   ├── service/             # Business logic (interfaces)
│   │   └── mocks/           # Testify mocks for repo, cache, producer
│   ├── repository/          # PostgreSQL via GORM
│   ├── grpc/                # gRPC server + proto
│   │   └── proto/pb/        # Generated protobuf files
│   ├── kafka/
│   │   ├── audit/           # Kafka producer + TodoEventProducer interface
│   │   │   ├── handlers/    # Consumer message handler
│   │   │   └── repository/  # Audit log -> PostgreSQL
│   │   ├── consumer/        # Sarama ConsumerGroup
│   │   └── producer/        # Sarama SyncProducer factory
│   ├── redis/               # Redis client + CacheStore interface
│   ├── middleware/          # RequestID, Logger, ErrorHandler
│   ├── metrics/             # Prometheus metrics
│   ├── models/              # Todo, KafkaEvent, request/response types
│   └── errors/              # Typed AppError (code + HTTP status)
├── docker-compose.yml
└── prometheus.yml
```

## REST API

Base URL: `http://localhost:8080`

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/posts` | Get all todos (cached) |
| `POST` | `/posts` | Create a todo |
| `PATCH` | `/posts/:id` | Update title and description |
| `PATCH` | `/posts/:id/completed` | Mark todo as completed/uncompleted |
| `DELETE` | `/posts/:id` | Delete a todo |
| `DELETE` | `/posts` | Delete all todos |
| `GET` | `/metrics` | Prometheus metrics |

**Create todo:**
```json
POST /posts
{
  "title": "Buy milk",
  "description": "2% from the corner store"
}
```

**Mark completed:**
```json
PATCH /posts/:id/completed
{
  "completed": true
}
```

**Error response format:**
```json
{
  "code": "ERR_NOT_FOUND",
  "message": "todo not found",
  "status": 404
}
```

## gRPC API

Port `:50051` — gRPC reflection is **not enabled** in this project (learning project).

| Method | Description |
|--------|-------------|
| `CreateTodo` | Create a todo |
| `GetTodo` | Get todo by ID |
| `UpdateTodo` | Update title and description |
| `MarkTodoCompleted` | Set completed status |

## Running Locally

**Prerequisites:** Docker, Docker Compose, Go 1.24+

```bash
git clone https://github.com/Finechh/todo-list
cd todo-list/todo_list
```

Create `cmd/api/.env`:
```bash
DB_HOST=localhost
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=todo_db
DB_PORT=5432
DB_SSLMODE=disable
```

```bash
# Start infrastructure
docker-compose up -d

# Run service
cd ./cmd/api/
go run main.go
```

Services:
- REST API — `http://localhost:8080`
- gRPC — `localhost:50051`
- Prometheus — `http://localhost:9090`
- Grafana — `http://localhost:3000`

## Manual Testing

Use [Postman](https://www.postman.com/) to test the API. Base URL: `http://localhost:8080`

### Full end-to-end flow

**Step 1 — Create a todo**
- Method: `POST`
- URL: `http://localhost:8080/posts`
- Body → raw → JSON:
```json
{
  "title": "Buy milk",
  "description": "From the corner store"
}
```
Response:
```json
{
  "id": "41099048-bf90-4e46-9057-ef3b3db3a551",
  "title": "Buy milk",
  "description": "From the corner store",
  "completed": false,
  "created_at": "2026-02-25T22:30:00Z",
  "updated_at": null
}
```

---

**Step 2 — Get all todos** (first call hits DB, second returns from Redis cache)
- Method: `GET`
- URL: `http://localhost:8080/posts`

---

**Step 3 — Mark as completed**
- Method: `PATCH`
- URL: `http://localhost:8080/posts/{id}/completed`
- Body → raw → JSON:
```json
{ "completed": true }
```

---

**Step 4 — Update title and description**
- Method: `PATCH`
- URL: `http://localhost:8080/posts/{id}`
- Body → raw → JSON:
```json
{
  "title": "Buy oat milk",
  "description": "From the health store"
}
```

---

**Step 5 — Delete**
- Method: `DELETE`
- URL: `http://localhost:8080/posts/{id}`

> **Note:** replace `{id}` with the real ID from the Step 1 response.

---

## Unit Tests

```bash
go test ./internal/service/... -v -cover
```

Tests cover:
- `CreateTodoList` — success, empty title, DB error, Kafka error (non-fatal)
- `GetTodoList` — success, not found, DB error
- `GetAllTodoList` — cache hit, cache miss fallback to DB, DB error
- `UpdateTodoList` — success, empty title, not found
- `DeleteTodoList` — success, not found
- `MarkTodoCompleted` — success, not found, idempotent
- `DeleteAllTodolist` — success, DB error
- `ValidateTodoInput` — empty and valid title

## Observability

Prometheus metrics at `http://localhost:8080/metrics`:

| Metric | Description |
|--------|-------------|
| `http_request_total` | HTTP requests by method and path |
| `http_request_duration_second` | HTTP latency histogram |
| `Http_error_total` | HTTP errors by method and path |
| `Backend_success_total` | Successful backend calls (redis, kafka) |
| `Backend_error_total` | Failed backend calls |
| `Backend_duration_second` | Backend latency (redis, kafka) |
| `Cache_miss` | Redis cache misses |
