# Go URL Shortener
Project นี้เป็น URL Shortener ใช้ Golang Clean Architecture, PostgreSQL, Redis และ Next.js
ในการพัฒนา

# Flow overview
Create URL:
Frontend -> Backend -> PostgreSQL -> Redis

Redirect:
GET /abc123
-> Check Redis
-> ถ้าเจอ redirect ทันที
-> ถ้าไม่เจอ query PostgreSQL
-> set Redis ใหม่
-> redirect

##  Go URL Shortener Tech Stack

- Backend: Golang, Chi Router, pgx, go-redis
- Frontend: Next.js App Router
- Database: PostgreSQL
- Cache: Redis
- Infra: Docker Compose


## Backend API

### Create short URL

```http
POST /api/v1/shorten
Content-Type: application/json

{
  "url": "https://example.com"
}
```

### Redirect

```http
GET /{shortCode}
```

## Run with Docker Compose

```bash
docker compose up --build
```

Frontend:
```txt
http://localhost:3000
```

Backend:
```txt
http://localhost:8080
```

## Run backend locally

```bash
cd backend
cp .env.example .env
go mod tidy
go run ./cmd/api
```

## Run frontend locally

```bash
cd frontend
cp .env.example .env.local
npm install
npm run dev
```

Set env on backend:

```env
APP_PORT=8080
BASE_URL=https://your-backend.onrender.com
DATABASE_URL=postgresql://...
REDIS_ADDR=...
REDIS_PASSWORD=...
URL_TTL_MINUTES=10
```

Set env on frontend:

```env
NEXT_PUBLIC_API_BASE_URL=https://your-backend.onrender.com
```
