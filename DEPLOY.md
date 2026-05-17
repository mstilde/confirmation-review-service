# Deploy Paso a Paso

## PASO 1 — Subir a GitHub

```bash
# En una terminal:
cd C:\Users\Matias\Proyectos\confirmation-review-service

# Autenticate con GitHub (abre navegador, pega el código)
"C:\Program Files\GitHub CLI\gh.exe" auth login

# Crear repo y pushear
"C:\Program Files\GitHub CLI\gh.exe" repo create confirmation-review-service --public --push --source . --remote origin
```

---

## PASO 2 — Deploy backend Go en Railway

1. Entrá a https://railway.app
2. Click **"+ New Project"** (o usá el proyecto existente)
3. Click **"+ New Service" → "GitHub Repo"**
4. Seleccioná el repo `confirmation-review-service`
5. **IMPORTANTE:** En la pantalla de configuración del servicio, buscá **"Root Directory"** y poné `backend`
6. Railway va a detectar el Dockerfile. Si no, setealo manual: Build → Dockerfile path → `docker/Dockerfile`
7. Andá a la pestaña **Variables** y agregá:

| Nombre | Valor |
|--------|-------|
| `DATABASE_URL` | `postgresql://postgres.jutczosptsbmjqhzprvp:Powing2025!@aws-1-us-east-1.pooler.supabase.com:5432/postgres` |
| `JWT_SECRET` | `cr-prod-jwt-` + algo aleatorio, ej: `cr-prod-jwt-8a7b3c2d1e` |
| `BRIDGE_KEY` | `bridge-local-dev-key-2026` |
| `N8N_PENDING_ACTION_WEBHOOK_URL` | *(pendiente, ver paso 5)* |
| `PORT` | `8080` |

8. Click **Deploy**. Railway te va a dar una URL tipo:
   `https://confirmation-review-service-production.up.railway.app`
   
   **Anotala.** Es `TU_URL_GO`.

9. Verificá que funcione:
   ```bash
   curl https://TU_URL_GO.up.railway.app/api/health
   # Debe devolver {"status":"ok"}
   ```

10. **Creá el primer usuario:**
    ```bash
    curl -X POST https://TU_URL_GO.up.railway.app/api/auth/setup \
      -H "Content-Type: application/json" \
      -d '{"email":"TU_EMAIL","password":"TU_PASSWORD"}'
    ```

---

## PASO 3 — Deploy frontend en Vercel

1. Entrá a https://vercel.com
2. Click **"New Project"**
3. Importá el repo `confirmation-review-service` desde GitHub
4. Configurá:
   - **Root Directory**: `frontend`
   - **Framework**: Next.js (auto-detectado)
5. En **Environment Variables**:

| Nombre | Valor |
|--------|-------|
| `NEXT_PUBLIC_API_URL` | `https://TU_URL_GO.up.railway.app` (la del paso 2.8) |

6. Click **Deploy**. Vercel te da una URL tipo:
   `https://confirmation-review-frontend.vercel.app`
   
   **Anotala.** Es `TU_URL_FRONTEND`.

7. Probá: abrí `https://TU_URL_FRONTEND.vercel.app` → login → deberías ver la lista vacía.

---

## PASO 4 — Obtener la webhook URL del Action Handler en n8n

1. Entrá a https://n8n.srv1515461.hstgr.cloud
2. Abrí el workflow **"Pending Confirmaciones — Action Handler"**
3. Click en el nodo **"Webhook — Pending Action"**
4. Copiá la **Production URL** (algo como `https://n8n.srv1515461.hstgr.cloud/webhook/pending-action`)
5. Volvé a Railway → pestaña Variables → actualizá `N8N_PENDING_ACTION_WEBHOOK_URL` con esa URL
6. Re-deploy de Railway (o simplemente actualizá la variable, Railway reinicia solo)

---

## PASO 5 — Actualizar n8n para que apunte al nuevo backend

En n8n, abrí estos dos workflows y actualizá las URLs:

### Workflow: "Confirmaciones Automáticas — Citas"

Buscá los 3 nodos HTTP que tienen esta URL:
```
https://mensajer-awhatsapp-production.up.railway.app/api/pending-confirmations
```

Reemplazala por:
```
https://TU_URL_GO.up.railway.app/api/cases
```

Nodos a actualizar:
1. `HTTP — Encolar Pending Review`
2. `HTTP — Emit Skip Account Disabled`
3. `HTTP — Emit Skip Pre-Loop`

### Workflow: "Confirmaciones Automáticas — Mañana"

Misma operación. Reemplazar la URL vieja por la nueva en:
1. `HTTP — Encolar Pending Review`
2. `HTTP — Emit Skip Account Disabled`
3. `HTTP — Emit Skip Pre-Loop`
4. `HTTP — Emit Skip No Template`

**Nota:** El header `x-bridge-key: bridge-local-dev-key-2026` ya está configurado. No tocarlo.

---

## PASO 6 — Probar el flujo completo

1. Asegurate de que los 2 workflows de n8n estén **active**
2. Ejecutalos manualmente (o esperá al schedule)
3. Si hay casos ambiguos → deberían aparecer en la app
4. Abrí `https://TU_URL_FRONTEND.vercel.app` → deberías ver los casos
5. Probá approve / skip / cancel

---

## PASO 7 — Agregar link en el dashboard existente

En `public/index.html` del proyecto unipile-whatsapp, agregá:

```html
<a href="https://TU_URL_FRONTEND.vercel.app" target="_blank" class="nav-item">
  📋 Confirmaciones
</a>
```

---

## Resumen de URLs

| Qué | URL |
|-----|-----|
| Backend Go (Railway) | `https://TU_URL_GO.up.railway.app` |
| Frontend (Vercel) | `https://TU_URL_FRONTEND.vercel.app` |
| API health check | `https://TU_URL_GO.up.railway.app/api/health` |
| Crear usuario | `POST https://TU_URL_GO.up.railway.app/api/auth/setup` |
| n8n webhook (action handler) | La del paso 4 |
