## Task Management API — Backend (Go + Gin)

### Goal
- **Provide a secure, reliable REST API and realtime channel** for a task management app.
- Demonstrate clean layering (handlers, middleware, models), robust auth, and pragmatic database design.

### Tech Stack
- **Language/Runtime**: Go 1.25
- **HTTP**: Gin
- **Database/ORM**: SQLite (pure Go driver) + GORM
- **Auth**: JWT (golang-jwt v5) with issuer and audience validation
- **Realtime**: Gorilla WebSocket with in‑memory hub per user
- **Testing**: Testify

### API Endpoints
- Public
  - `POST /api/login` — Login or first‑time signup with username + SHA‑256(password) from FE
  - `GET /health` — Health check
- Protected (Bearer JWT or `?token=` for WS)
  - `GET /api/ws` — WebSocket; user‑scoped events (created/updated/deleted/status changed)
  - `GET /api/tasks` — Paginated list; `page`, `limit`, `sort=asc|desc`, optional `userId`
  - `GET /api/tasks/:id` — Task by id (owned by auth user)
  - `POST /api/tasks` — Create task
  - `PUT /api/tasks/:id` — Update task
  - `PATCH /api/tasks/:id/status` — Update status only
  - `DELETE /api/tasks/:id` — Delete task
  - `GET /api/stats/:userid` — Counts by status for a given assignee
  - `GET /api/users` — List users (id, username)

### Database Design
- Tables
  - `users`
    - `id` (PK, string UUID)
    - `username` (unique)
    - `password` (bcrypt of the SHA‑256 hash sent by FE)
    - timestamps
  - `tasks`
    - `id` (PK, string; `task-{timestamp}`)
    - `title` (required), `description`
    - `status` (`todo` | `inProgress` | `done`)
    - `task_type` (`story` | `defect` | `subtask`)
    - `project_id` (string; parent story id for `defect`/`subtask`, empty for `story`)
    - `assignee_id` (string; references `users.id` logically)
    - `start_date`, `end_date`, `effort` (auto‑computed from dates)
    - `priority` (`high` | `medium` | `low`)
    - `user_id` (owner/auth user)
    - timestamps

### System Design Overview
- **Gin** router with CORS middleware (`ALLOWED_ORIGIN`) and JWT middleware.
- **JWT**
  - Claims include `user_id`, `username`, `iss`, `aud`, and expiry.
  - Issuer and audience validated on every request.
- **Auth flow**
  - FE sends SHA‑256(password); API locates user by username.
  - If user exists: bcrypt compare vs stored bcrypt(SHA‑256(password)).
  - If not: create user with bcrypt(SHA‑256(password)), then issue JWT.
- **Business rules**
  - Effort computed from `start_date` to `end_date` server‑side.
  - `story`: `project_id` must be empty.
  - `defect`/`subtask`: `project_id` required and must point to an existing `story`.
- **Realtime**
  - In‑memory hub maps userId → active WS clients.
  - Handlers broadcast events on task create/update/delete/status change.
  - Browser authenticates WS with `?token=<JWT>` query param.

### Non‑Functional Highlights
- **Security**: JWT with issuer/audience, bcrypt at rest, CORS controlled, header/query token support for WS.
- **Reliability**: Clear error handling, pagination with `total`, `count`, and consistent responses.
- **DX**: Auto‑migrations, structured handlers, tests, health endpoint, startup endpoint logs.

### Run Locally
1) Requirements
   - Go 1.25+

2) Start API
```bash
cd cmd/server
go run .
```
Server defaults to `:8008` and logs available endpoints.

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

### cURL Examples
```bash
# Login (supply SHA-256 hashed password from FE in real usage)
curl -X POST http://localhost:8008/api/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo","password":"<sha256-hex>"}'

# Get tasks
curl -H 'Authorization: Bearer <JWT>' \
  'http://localhost:8008/api/tasks?page=1&limit=5&sort=desc'

# Open WebSocket (browser uses query param)
# ws://localhost:8008/api/ws?token=<JWT>
```

### Notes
- SQLite file is `tasks-management.db` at the project root and is auto‑migrated.
- Make sure the frontend origin is allowed by `ALLOWED_ORIGIN`.

