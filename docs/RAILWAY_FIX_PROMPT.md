# PROMPT: Deploy Backend Go a Railway

## Contexto

Tenemos un backend Go en un monorepo con esta estructura:

```
confirmation-review-service/
├── backend/          ← servicio Go (Gin + pgx)
│   ├── Dockerfile    ← Dockerfile de producción (multi-stage)
│   ├── docker/Dockerfile  ← copia idéntica (ignorar)
│   ├── railway.toml  ← config de Railway
│   ├── go.mod
│   ├── cmd/server/main.go
│   └── internal/...
├── frontend/         ← Next.js (se deploya aparte en Vercel)
└── docs/
```

## El problema

Railway falló con este error:

```
using build driver railpack-v0.23.0
⚠ Script start.sh not found

The app contents that Railpack analyzed contains:
├── backend/
├── docs/
├── frontend/
├── .gitignore
├── README.md
└── docker-compose.yml
```

Railway está analizando la raíz del monorepo en vez de `backend/`.

## La solución

### En el dashboard de Railway:

1. Andá al proyecto donde está el servicio
2. Entrá a **Settings** del servicio
3. Buscá **"Root Directory"** (o "Source Directory")
4. Setealo a: `backend`
5. Guardá y hacé **Re-deploy**

Con eso Railway va a:
- Leer `backend/railway.toml`
- Usar `backend/Dockerfile` (multi-stage Go → Alpine)
- Exponer puerto 8080
- Health check en `/api/health`

### Variables de entorno (también en Settings → Railway):

| Variable | Valor |
|----------|-------|
| `DATABASE_URL` | `postgresql://postgres.jutczosptsbmjqhzprvp:Powing2025!@aws-1-us-east-1.pooler.supabase.com:5432/postgres` |
| `JWT_SECRET` | *(generar uno, ej: cr-prod-jwt-xxxxx)* |
| `BRIDGE_KEY` | `bridge-local-dev-key-2026` |
| `N8N_PENDING_ACTION_WEBHOOK_URL` | *(URL del webhook de n8n, si ya existe)* |
| `PORT` | `8080` |

### El railway.toml ya está listo:

```toml
[build]
builder = "DOCKERFILE"
dockerfilePath = "docker/Dockerfile"

[deploy]
healthcheckPath = "/api/health"
healthcheckTimeout = 30
```

### El Dockerfile (`backend/docker/Dockerfile`):

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=America/Argentina/Buenos_Aires
COPY --from=builder /server /usr/local/bin/server
EXPOSE 8080
CMD ["server"]
```

### Después del deploy:

Crear el primer usuario:

```bash
curl -X POST https://TU_URL_RAILWAY.up.railway.app/api/auth/setup \
  -H "Content-Type: application/json" \
  -d '{"email":"TU_EMAIL","password":"TU_PASSWORD"}'
```

---

**TL;DR:** El Root Directory en Railway apunta a la raíz del repo. Cambialo a `backend`. Re-deploy. Listo.
