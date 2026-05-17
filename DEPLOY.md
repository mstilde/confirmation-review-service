# Guía de Deploy: Railway + Vercel

## Paso 1: Subir a GitHub

Creá el repo en github.com (ej: `confirmation-review-service`), después:

```bash
cd C:\Users\Matias\Proyectos\confirmation-review-service
git remote add origin https://github.com/TU_USUARIO/confirmation-review-service.git
git commit -m "Initial commit: Go backend + Next.js PWA for AI-assisted confirmation review"
git push -u origin master
```

---

## Paso 2: Deploy Backend Go en Railway

1. Entrá a [railway.app](https://railway.app) → creá un **nuevo proyecto** (o usá el existente de Unipile)
2. Click en **"+ New Service" → "GitHub Repo"** → seleccioná `confirmation-review-service`
3. Railway va a detectar automáticamente el Dockerfile. Configurá:
   - **Root Directory**: `backend`
4. Agregá estas **variables de entorno** en la pestaña "Variables":

| Variable | Valor |
|----------|-------|
| `DATABASE_URL` | `postgresql://postgres.jutczosptsbmjqhzprvp:Powing2025!@aws-1-us-east-1.pooler.supabase.com:5432/postgres` |
| `JWT_SECRET` | *(generá uno largo y único)* |
| `BRIDGE_KEY` | `bridge-local-dev-key-2026` |
| `N8N_PENDING_ACTION_WEBHOOK_URL` | `https://n8n.srv1515461.hstgr.cloud/webhook/pending-action` |
| `PORT` | `8080` |

5. Railway deploya automáticamente. Te va a dar una URL tipo `https://confirmations-backend.up.railway.app`.
6. **Creá el primer usuario**:
```bash
curl -X POST https://confirmations-backend.up.railway.app/api/auth/setup \
  -H "Content-Type: application/json" \
  -d '{"email":"TU_EMAIL","password":"TU_PASSWORD"}'
```

---

## Paso 3: Deploy Frontend Next.js en Vercel

1. Entrá a [vercel.com](https://vercel.com) → **"New Project"**
2. Importá el repo `confirmation-review-service` de GitHub
3. Configurá:
   - **Root Directory**: `frontend`
   - **Framework Preset**: Next.js (auto-detectado)
4. Agregá **Environment Variables**:

| Variable | Valor |
|----------|-------|
| `NEXT_PUBLIC_API_URL` | `https://TU_BACKEND_RAILWAY_URL.up.railway.app` |

5. Actualizá `frontend/vercel.json` con la URL real del backend (reemplazá `tu-backend.up.railway.app` por la URL de Railway real).

6. Deploy. Vercel te da una URL tipo `https://confirmation-review-frontend.vercel.app`.

---

## Paso 4: Probar

1. Abrí `https://TU_FRONTEND.vercel.app`
2. Login con el usuario creado en el paso 2.6
3. Abrí un celular y probá desde ahí

---

## Paso 5: Configurar n8n para que encole casos al nuevo backend

En los workflows de n8n (`Confirmaciones Automáticas — Citas` y `Confirmaciones Automáticas — Mañana`), agregar nodos HTTP Request que POSTeen a:

```
POST https://TU_BACKEND_RAILWAY_URL.up.railway.app/api/cases
Header: x-bridge-key: bridge-local-dev-key-2026
Body: { idempotency_key, cita_id, contact_name, appointment_at, flow_source, ai_reason, chat_context, suggested_message }
```

Cuando el workflow decide **NO auto-confirmar**, en vez de solo loguear, que haga ese POST.

---

## Paso 6: Vincular el app existente

Agregá un link en el dashboard existente (`index.html`) para que apunte a la URL de Vercel:

```html
<a href="https://TU_FRONTEND.vercel.app" class="nav-item" target="_blank">
  📋 Confirmaciones
</a>
```

---

## Notas

- **El `vercel.json`** tiene un proxy `/api/*` → Railway. Eso simplifica las llamadas del frontend (no necesita CORS complejo).
- **Las migraciones** de DB se ejecutan automáticamente al iniciar el backend.
- **Push notifications** requieren VAPID keys. Para producción, generarlas con `npx web-push generate-vapid-keys` y setearlas en Railway.
