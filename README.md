# Confirmation Review Service

Sistema de revisión de confirmaciones de citas asistido por IA. Backend en **Go (Gin + pgx)**, frontend **Next.js (PWA)**, PostgreSQL.

## Arquitectura

```
n8n + Groq (decision engine)
        │
        │ POST /api/cases (casos ambiguos)
        ▼
┌──────────────────────┐
│   Go Backend (Gin)   │
│   handlers → svc → repo │
│   PostgreSQL           │
│   Web Push             │
└──────┬───────────────┘
       │ REST API + JWT
       ▼
┌──────────────────────┐
│  Next.js Frontend    │
│  PWA + Service Worker│
└──────────────────────┘
```

## Setup rápido

### Requisitos
- Go 1.23+
- Node.js 20+
- PostgreSQL (misma DB que la app principal)

### Backend

```bash
cd backend
cp .env.example .env   # editar DATABASE_URL y JWT_SECRET
go mod tidy
go run ./cmd/server    # arranca en :8080
```

Crear primer usuario:

```bash
curl -X POST http://localhost:8080/api/auth/setup \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}'
```

### Frontend

```bash
cd frontend
npm install
npm run dev            # arranca en :3004
```

Abrir `http://localhost:3004` en el celular o navegador.

## API Endpoints

### Público
| Método | Path | Descripción |
|--------|------|-------------|
| POST | `/api/auth/login` | Login (devuelve JWT) |
| POST | `/api/auth/setup` | Crear usuario |
| GET | `/api/health` | Health check |

### Autenticado (JWT)
| Método | Path | Descripción |
|--------|------|-------------|
| GET | `/api/auth/me` | Usuario actual |
| GET | `/api/cases/pending?kind=actionable&flow_source=` | Listar pendientes |
| GET | `/api/cases/:id` | Detalle de caso |
| POST | `/api/cases/:id/approve` | Aprobar (→ webhook n8n) |
| POST | `/api/cases/:id/skip` | Skipear |
| POST | `/api/cases/:id/cancel` | Cancelar (→ webhook n8n) |
| GET | `/api/push/vapid-public-key` | Clave VAPID pública |
| POST | `/api/push/subscribe` | Suscribir a push |

### Bridge key (n8n)
| Método | Path | Descripción |
|--------|------|-------------|
| POST | `/api/cases` | Encolar caso |
| GET | `/api/cases/count?flow_source=` | Contar pendientes |
| POST | `/api/cases/:id/refresh-chat` | Actualizar chat context |
| POST | `/api/notify` | Notificar fin de workflow |
| POST | `/api/cases/expire` | Expirar casos viejos |

## Variables de entorno

| Variable | Default | Descripción |
|----------|---------|-------------|
| `DATABASE_URL` | — | PostgreSQL connection string |
| `JWT_SECRET` | — | Secreto para firmar JWTs |
| `BRIDGE_KEY` | `bridge-local-dev-key-2026` | Key compartida con n8n |
| `N8N_PENDING_ACTION_WEBHOOK_URL` | — | Webhook de n8n para approve/cancel |
| `PORT` | `8080` | Puerto del backend |
| `CASE_EXPIRY_DAYS` | `1` | Días hasta expirar casos |
| `VAPID_PUBLIC_KEY` | — | Clave pública VAPID para push |
| `VAPID_PRIVATE_KEY` | — | Clave privada VAPID para push |

## Docker

```bash
docker compose up -d
```

## Estructura del proyecto

```
├── backend/
│   ├── cmd/server/          # Entry point
│   ├── internal/
│   │   ├── auth/            # JWT + middleware
│   │   ├── config/          # Env vars
│   │   ├── handler/         # HTTP handlers (Gin)
│   │   ├── model/           # Structs
│   │   ├── repository/      # PostgreSQL (pgx)
│   │   └── service/         # Lógica de negocio
│   └── docker/              # Dockerfile
├── frontend/                # Next.js 14 (App Router)
│   ├── src/app/             # Páginas
│   ├── src/lib/             # API client + auth
│   └── public/              # PWA manifest + SW
├── docs/
│   └── LEARNING.md          # Guía de aprendizaje
└── docker-compose.yml
```

## Licencia

Privada — uso interno.
