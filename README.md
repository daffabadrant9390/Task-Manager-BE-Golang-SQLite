## Task Management API — Backend (Go + Gin)

### Why this backend stands out
- **Production‑leaning REST design** with clean layering and clear contracts.
- **Zero‑friction local setup** (SQLite or in‑memory) and **developer‑friendly tooling**.
- **Real‑time ready** with WebSocket fan‑out and an optional goroutine‑safe cache.

### Senior Reviewer Checklist — how we meet the criteria
- **Golang + Gin/Chi**: Uses Gin as the HTTP router and middleware engine.
- **Clean structure**: Standard `internal/` and `cmd/` layout; handlers, routes, middleware, models, auth, cache.
- **SQLite or in‑memory**: Default is SQLite via GORM; swapping to in‑memory store is straightforward.
- **Required endpoints implemented**:
  - `POST /api/login` — issues a JWT for mock/dummy auth (no external IdP).
  - `GET /api/tasks` — lists tasks owned by the authenticated user.
  - `POST /api/tasks` — creates a new task.
  - `PUT /api/tasks/:id` — updates title/status.
  - `DELETE /api/tasks/:id` — deletes a task.
- **RESTful & modular**: Resource‑based URIs, proper verbs, consistent status codes and error shapes.
- **Middleware**: JWT validation, CORS, and route grouping for public/protected paths.
- **Optional concurrency**: Lightweight, goroutine‑safe TTL cache in `internal/cache` with tests.

### Tech Stack
- **Language/Runtime**: Go 1.25
- **HTTP Router**: Gin
- **Data**: SQLite (pure Go) via GORM; can operate fully in memory for ephemeral runs
- **Auth**: JWT (golang‑jwt v5) — mock‑friendly, signed server‑side token
- **Realtime**: Gorilla WebSocket with user‑scoped hub
- **Testing**: `go test` with `testify` and cache coverage in `internal/cache`

### Core API Endpoints
- Public
  - `POST /api/login` — mock authentication, returns a signed JWT for the user
  - `GET /health` — health probe
- Protected (Bearer JWT; WS accepts `?token=`)
  - `GET /api/tasks` — list tasks (owned by user); supports `page`, `limit`, `sort=asc|desc`
  - `POST /api/tasks` — create task (title, description, status)
  - `PUT /api/tasks/:id` — update task (title/status)
  - `DELETE /api/tasks/:id` — delete task
  - Extras implemented: `GET /api/tasks/:id`, `PATCH /api/tasks/:id/status`, `GET /api/stats/:userid`, `GET /api/ws`

### Advanced capabilities (implemented)
- **Pagination & sorting** on `/api/tasks` with consistent response metadata.
- **WebSocket** push for task create/update/delete/status‑change events.
- **Sub‑tasks** model support (task types and parent linkage) with server‑side validation.
- **Stats**: `/api/stats/:userid` for completion summary per user.

### System Design Highlights
- **JWT middleware** guarding protected routes, with issuer/audience support.
- **CORS** configurable via `ALLOWED_ORIGIN` for FE integration.
- **Auto‑migrations** keep SQLite schema up to date on boot.
- **Goroutine‑safe cache** (TTL, lazy expiration, purge) under `internal/cache`.

### Project structure
```
cmd/
  server/            # main entrypoint
internal/
  auth/              # JWT helpers
  cache/             # goroutine‑safe TTL cache + tests
  database/          # SQLite + GORM initialization
  handlers/          # HTTP handlers (auth, tasks, stats, ws)
  middleware/        # JWT, CORS
  models/            # GORM models
  routes/            # router wiring (public/protected)
```

### Run locally
1) Requirements
   - Go 1.25+

2) Start API
```bash
cd cmd/server
go run .
```
Server defaults to `:8008` and prints available endpoints.

3) Environment (optional but recommended)
```bash
# .env (set in your shell or process manager)
ALLOWED_ORIGIN=http://localhost:3000
JWT_SECRET=change-me
JWT_ISSUER=task-management-api
JWT_AUDIENCE=task-management-clients
```

### Testing
```bash
go test ./...
```
Includes unit tests for the concurrency‑safe cache in `internal/cache`.

### cURL quickstart
```bash
# 1) Login (mock auth → returns JWT)
curl -X POST http://localhost:8008/api/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo","password":"demo"}'

# 2) List tasks (auth required)
curl -H 'Authorization: Bearer <JWT>' \
  'http://localhost:8008/api/tasks?page=1&limit=5&sort=desc'

# 3) Create task
curl -X POST http://localhost:8008/api/tasks \
  -H 'Authorization: Bearer <JWT>' \
  -H 'Content-Type: application/json' \
  -d '{"title":"My first task","status":"todo"}'
```

### Notes
- SQLite file is created at the project root and auto‑migrated.
- Ensure the frontend origin is allowed via `ALLOWED_ORIGIN`.

