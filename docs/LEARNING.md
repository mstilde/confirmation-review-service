# 🎓 Guía de aprendizaje: Confirmation Review Service

> Una guía didáctica paso a paso para entender cada concepto del proyecto mientras lo construimos. Pensada para defender en entrevistas técnicas.

---

## Índice

1. [¿Qué estamos construyendo y por qué?](#1-qué-estamos-construyendo-y-por-qué)
2. [Arquitectura: la foto grande](#2-arquitectura-la-foto-grande)
3. [Go: lo esencial en 10 minutos](#3-go-lo-esencial-en-10-minutos)
4. [Estructura de un proyecto Go profesional](#4-estructura-de-un-proyecto-go-profesional)
5. [Separación en capas: handlers → services → repository](#5-separación-en-capas)
6. [HTTP con Gin: handlers, routing, middleware](#6-http-con-gin)
7. [PostgreSQL con pgx: pool, queries, JSONB](#7-postgresql-con-pgx)
8. [Repository pattern: por qué separamos SQL del HTTP](#8-repository-pattern)
9. [Service layer: donde vive la lógica de negocio](#9-service-layer)
10. [Máquina de estados finita](#10-máquina-de-estados-finita)
11. [Idempotencia: cómo evitar duplicados](#11-idempotencia)
12. [JWT: autenticación sin sesiones](#12-jwt-autenticación-sin-sesiones)
13. [Bridge key: autenticación para máquinas](#13-bridge-key)
14. [Webhooks: comunicación entre servicios](#14-webhooks)
15. [Audit log: trazabilidad de cada acción](#15-audit-log)
16. [Expiración de casos](#16-expiración-de-casos)
17. [Chat refresh: mantener datos actualizados](#17-chat-refresh)
18. [Push notifications: cómo notificar al celular](#18-push-notifications)
19. [PWA: manifest y service worker](#19-pwa)
20. [Edge cases: qué puede fallar y cómo lo manejamos](#20-edge-cases)
21. [Cómo defender esto en una entrevista](#21-cómo-defender-esto-en-una-entrevista)

---

## 1. ¿Qué estamos construyendo y por qué?

### El problema real

Un equipo de ~20 asesores comerciales coordina citas por WhatsApp. Hay dos flujos automáticos (n8n) que se ejecutan todos los días:

- **Citas** — confirma las citas de hoy
- **Mañana** — recuerda las citas de mañana

Cada flujo analiza el chat con IA (Groq/Claude) para decidir si es seguro mandar un mensaje de confirmación. ~80% de los casos se auto-confirman. Pero hay un **20% de casos ambiguos** donde la IA no está segura: el prospecto dijo "quizás", o "no sé si puedo", o hubo otro mensaje reciente que podría confundir.

### La solución

Una **capa humana** (esta app) que recibe esos casos ambiguos y le permite a un operador revisar el contexto del chat, el mensaje que la IA sugirió, y decidir:

- ✅ **Aprobar** → manda el mensaje por WhatsApp, marca Notion como confirmado
- ❌ **Cancelar** → cancela la cita en Notion
- ❓ **Skipear** → sin acción externa, el operador lo resuelve manualmente más tarde

### Por qué esto es valioso profesionalmente

No es un simple CRUD. Es un sistema real con:
- **Human-in-the-loop**: IA propone, humano decide
- **Integración entre servicios**: n8n, Go, PostgreSQL, Unipile, Notion
- **Manejo de estados**: cada caso pasa por transiciones validadas
- **Auditoría completa**: cada acción queda registrada
- **Notificaciones push**: el operador recibe alertas en el celular

---

## 2. Arquitectura: la foto grande

```
┌─────────────────────────────────────────────────────────┐
│                     n8n Workflows                        │
│  ┌───────────────────┐    ┌──────────────────┐          │
│  │ Citas (hoy)       │    │ Mañana            │          │
│  │  - Groq analiza   │    │  - Groq analiza   │          │
│  │  - Decide mandar/ │    │  - Decide mandar/ │          │
│  │    no mandar/     │    │    no mandar/     │          │
│  │    encolar        │    │    encolar        │          │
│  └───────┬───────────┘    └───────┬──────────┘          │
│          │ POST /api/cases        │ POST /api/cases     │
│          │ (bridge key)           │ (bridge key)        │
└──────────┼────────────────────────┼─────────────────────┘
           │                        │
           ▼                        ▼
┌──────────────────────────────────────────────────────────┐
│              Go Backend (Gin + pgx)                       │
│                                                           │
│  ┌──────────────┐    ┌──────────────┐    ┌─────────────┐ │
│  │   handlers   │───▶│   services   │───▶│ repository  │ │
│  │   HTTP/REST  │    │ negocio/     │    │  SQL/pgx   │ │
│  │   Gin        │    │ validación   │    │             │ │
│  └──────────────┘    └──────────────┘    └──────┬──────┘ │
│                                                  │        │
│                                          ┌───────▼──────┐ │
│                                          │  PostgreSQL  │ │
│                                          │  (Supabase)  │ │
│                                          └──────────────┘ │
│                                                           │
│  ┌──────────────────────────────────────────────────┐    │
│  │ POST /cases/:id/approve                           │    │
│  │ POST /cases/:id/cancel                            │───▶│ Webhook a n8n
│  └──────────────────────────────────────────────────┘    │
│                                                           │
│  POST /notify ← n8n avisa que terminó                     │
│  POST /cases/:id/refresh-chat ← n8n actualiza chat        │
└──────────────────────────────────────────────────────────┘
           │
           │ GET /api/cases/pending
           │ POST /api/auth/login
           ▼
┌──────────────────────────────────────────────────────────┐
│              Next.js Frontend (PWA)                       │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Login → Lista de pendientes → Detalle del caso   │  │
│  │  [✓ Aprobar]  [? Skipear]  [✗ Cancelar]          │  │
│  └───────────────────────────────────────────────────┘  │
│                                                          │
│  Service Worker → recibe push notifications              │
└──────────────────────────────────────────────────────────┘
```

### Flujo completo

```
1. n8n ejecuta workflow (Citas o Mañana)
2. IA analiza cada chat
3. Si decide NO mandar → POST /api/cases { cita_id, ai_reason, chat_context, ... }
4. Go guarda en confirmation_cases con status='pending'
5. n8n termina → POST /api/notify { flow_source: "citas" }
6. Go cuenta pending → si hay → envía push notification al celular
7. Operador abre la app, ve la lista de casos
8. Ve el detalle: chat + razón IA + mensaje sugerido
9. Decide: approve / skip / cancel
10. Si approve → Go POSTea webhook a n8n → n8n manda WA + update Notion
11. Go actualiza status + audit log
```

---

## 3. Go: lo esencial en 10 minutos

Go es un lenguaje compilado, con tipado estático, creado por Google. Su filosofía: simplicidad, eficiencia, y legibilidad.

### Structs (en vez de clases)

Go no tiene clases. Tiene **structs** que agrupan campos. Los métodos se definen *fuera* del struct.

```go
// Definición de un struct
type ConfirmationCase struct {
    ID          int64   `json:"id"`          // tags: metadatos para JSON/DB
    ContactName *string `json:"contact_name"` // *string = nullable
    Status      CaseStatus
}

// Método (el receptor va ANTES del nombre)
func (c *ConfirmationCase) IsPending() bool {
    return c.Status == StatusPending
}
```

### Punteros para nullable

En Go, `string` **no puede ser nil**. Si necesitás que un campo sea opcional (nullable en DB), usás `*string` (puntero a string). `nil` significa NULL en la DB.

```go
contactName := "Juan"          // string
var contactName *string = nil  // nullable, NULL en DB
```

### Manejo de errores (sin try/catch)

Go no tiene excepciones. Cada función que puede fallar devuelve `(resultado, error)`. **Siempre** chequeás el error.

```go
row, err := repository.GetCaseByID(id)
if err != nil {
    // manejá el error
    return nil, err
}
if row == nil {
    // no se encontró
    return nil, fmt.Errorf("caso no encontrado")
}
```

### Packages y visibilidad

- Los archivos `.go` en un mismo directorio pertenecen al mismo **package**
- Lo que empieza con **MAYÚSCULA** es exportado (público)
- Lo que empieza con **minúscula** es privado al package

```go
package model

const StatusPending CaseStatus = "pending"  // exportado
var validTransitions = map[...]             // privado (minúscula)
func CanTransition(...) bool { ... }        // exportado
```

### Interfaces

Go usa interfaces implícitas: no necesitás declarar "tal struct implementa tal interfaz". Si un struct tiene los métodos que pide la interfaz, **ya la implementa**.

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}
// Cualquier tipo que tenga Read(...) implementa Reader automáticamente
```

### Slice vs Array

```go
arr := [3]int{1, 2, 3}        // array: tamaño fijo, parte del tipo
slice := []int{1, 2, 3}       // slice: tamaño dinámico, referencia a array subyacente
slice = append(slice, 4)      // agrega elementos
```

---

## 4. Estructura de un proyecto Go profesional

```
backend/
├── cmd/server/main.go          ← entry point (solo inicializa y arranca)
├── internal/                    ← código privado del proyecto
│   ├── config/                  ← carga de variables de entorno
│   ├── model/                   ← structs de datos (sin lógica)
│   ├── auth/                    ← JWT + middleware
│   ├── handler/                 ← HTTP handlers (reciben request, devuelven response)
│   ├── service/                 ← lógica de negocio
│   └── repository/              ← acceso a base de datos
├── docker/                      ← Dockerfile del backend
├── go.mod                       ← dependencias
└── go.sum                       ← checksums de dependencias
```

**Regla**: `cmd/` es solo entry point. `internal/` contiene todo el código del proyecto. Go no permite que otros módulos importen `internal/`.

### ¿Por qué `internal/`?

Significa "esto es privado de este módulo". Nadie desde afuera puede importar `confirmation-review-service/internal/handler`. Es una garantía de encapsulamiento.

---

## 5. Separación en capas: handlers → services → repository

Esta es LA decisión de arquitectura más importante del proyecto. Si entendés esto, entendiste backend engineering.

### ¿Qué problema resuelve?

Imaginate que tu handler HTTP hace TODO: recibe el request, valida, escribe SQL, manda webhooks, loguea... Todo en un mismo lugar. Eso es:

- **Imposible de testear** (necesitás una DB para probar cualquier cosa)
- **Imposible de cambiar** (si cambiás de PostgreSQL a otra DB, tenés que tocar todos los handlers)
- **Imposible de razonar** (un archivo de 500 líneas que hace de todo)

### La solución: 3 capas

```
┌──────────────────────────────────────────────────────┐
│ handler  →  "¿Qué me pidieron?"                       │
│            Recibe HTTP, parsea JSON, llama al service │
│            NO tiene lógica de negocio                 │
│            NO toca SQL directamente                   │
├──────────────────────────────────────────────────────┤
│ service  →  "¿Qué hay que hacer?"                     │
│            Orquesta, valida reglas de negocio         │
│            Llama al repository para datos             │
│            Llama a servicios externos (webhooks)      │
│            NO sabe HTTP                               │
├──────────────────────────────────────────────────────┤
│ repository → "¿Dónde están los datos?"                │
│            Ejecuta SQL contra PostgreSQL              │
│            Devuelve structs de Go                     │
│            NO tiene lógica de negocio                 │
│            NO sabe HTTP                               │
└──────────────────────────────────────────────────────┘
```

### Ejemplo concreto en nuestro código

**Handler** (`handler/cases.go`):
```go
func (h *CaseHandler) Approve(c *gin.Context) {
    id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
    email := c.GetString("user_email")
    updated, err := h.svc.Approve(id, email)  // ← llama al service
    if err != nil {
        c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"success": true, "item": updated})
}
```

**Service** (`service/cases.go`):
```go
func (s *CaseService) Approve(caseID int64, userEmail string) (*model.ConfirmationCase, error) {
    c, err := repository.GetCaseByID(caseID)  // ← obtiene del repo
    if !model.CanTransition(c.Status, model.StatusApproved) {  // ← regla de negocio
        return nil, fmt.Errorf("no se puede aprobar en estado '%s'", c.Status)
    }
    if s.N8NWebhookURL != "" {
        s.dispatchToN8N(c, "send")  // ← orquesta servicio externo
    }
    updated, _ := repository.UpdateCaseStatus(caseID, model.StatusApproved, userEmail)
    repository.InsertAuditLog(caseID, "approved", &userEmail, details)
    return updated, nil
}
```

**Repository** (`repository/cases.go`):
```go
func GetCaseByID(id int64) (*model.ConfirmationCase, error) {
    var row model.ConfirmationCase
    err := Pool.QueryRow(ctx, "SELECT ... FROM confirmation_cases WHERE id = $1", id).
        Scan(&row.ID, &row.IdempotencyKey, ...)
    if err == pgx.ErrNoRows { return nil, nil }
    return &row, err
}
```

### ¿Por qué esto suma en una entrevista?

Porque muestra que entendés:
- **Separation of concerns**: cada capa tiene una responsabilidad clara
- **Testability**: podés testear el service con un repo mock, sin DB real
- **Maintainability**: si cambiás de PostgreSQL a MongoDB, solo tocás el repository
- **Lo que se hace en equipos reales**: esto es standard en startups y tech companies

---

## 6. HTTP con Gin: handlers, routing, middleware

Gin es el framework HTTP más popular de Go (equivalente a Express en Node).

### Handler

Un handler es una función que recibe `*gin.Context` (el request + response):

```go
func (h *CaseHandler) ListPending(c *gin.Context) {
    items, err := repository.ListPending("")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"success": true, "items": items})
}
```

`c.JSON()` serializa a JSON y setea el Content-Type. `c.Param("id")` lee parámetros de la URL. `c.DefaultQuery("kind", "actionable")` lee query params con default. `c.ShouldBindJSON(&input)` parsea el body JSON.

### Router y grupos

```go
func SetupRouter(cfg *config.Config) *gin.Engine {
    r := gin.Default()

    // Grupo público (sin auth)
    r.POST("/api/auth/login", authHandler.Login)

    // Grupo con JWT
    api := r.Group("/api")
    api.Use(auth.JWTAuth(cfg.JWTSecret))  // middleware aplicado a TODO este grupo
    {
        api.GET("/cases/pending", caseHandler.ListPending)
        api.POST("/cases/:id/approve", caseHandler.Approve)
    }

    // Grupo con bridge key (para n8n)
    bridge := r.Group("/api")
    bridge.Use(auth.BridgeKeyAuth(cfg.BridgeKey))
    {
        bridge.POST("/cases", caseHandler.Create)
    }

    return r
}
```

### Middleware

Un middleware es una función que se ejecuta ANTES del handler. Puede:
- Bloquear el request (ej: auth inválida → 401)
- Agregar datos al contexto (ej: `c.Set("user_email", email)`)
- Loggear, medir tiempos, etc.

```go
func JWTAuth(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractBearer(c.GetHeader("Authorization"))
        claims, err := ValidateToken(token, secret)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "Token inválido"})
            return  // ← ABORTA, no llama al siguiente handler
        }
        c.Set("user_email", claims.Email)  // inyecta datos
        c.Next()  // ← continúa al handler
    }
}
```

---

## 7. PostgreSQL con pgx: pool, queries, JSONB

pgx es el driver PostgreSQL más rápido y nativo para Go. Usamos `pgxpool` para manejar un pool de conexiones.

### Pool de conexiones

```go
var Pool *pgxpool.Pool

func Connect(databaseURL string) error {
    Pool, _ = pgxpool.New(context.Background(), databaseURL)
    return Pool.Ping(context.Background())
}
```

El pool mantiene conexiones abiertas y las reusa. **Nunca** crees una conexión por request — usá siempre el pool.

### Queries parametrizadas (previene SQL injection)

```go
// ✅ CORRECTO: placeholders $1, $2, ...
Pool.QueryRow(ctx, "SELECT * FROM confirmation_cases WHERE id = $1", id)
    .Scan(&row.ID, &row.IdempotencyKey, &row.CitaID, ...)

// ❌ NUNCA hagas esto:
Pool.QueryRow(ctx, fmt.Sprintf("SELECT * FROM confirmation_cases WHERE id = %d", id))
```

### NULL handling

En PostgreSQL, un campo puede ser NULL. En Go, los tipos básicos no pueden ser nil. pgx maneja esto con punteros:

```go
var contactName *string    // nil → NULL en DB
var resolvedAt *time.Time   // nil → NULL en DB
```

### JSONB

PostgreSQL tiene `JSONB` para almacenar JSON. pgx lo mapea automáticamente a `json.RawMessage` en Go:

```go
type ConfirmationCase struct {
    ChatContext json.RawMessage  `json:"chat_context"`
}
```

`json.RawMessage` es simplemente `[]byte` que representa JSON válido. Se serializa directo sin procesar.

---

## 8. Repository pattern: por qué separamos SQL del HTTP

### El problema

Si tu handler tiene SQL inline:

```go
// ❌ MAL: SQL en el handler
func ListPending(c *gin.Context) {
    rows, _ := db.Query("SELECT * FROM confirmation_cases WHERE status = 'pending'")
    // ... scan rows ...
    c.JSON(200, rows)
}
```

Problemas:
1. No podés testear el handler sin una DB real
2. Si la query cambia (ej: agregar filtro), tocás el handler
3. No podés reusar la query en otro lado
4. Si cambiás de PostgreSQL a MongoDB, reescribís TODO

### La solución: Repository

El repository es la ÚNICA capa que sabe SQL. Expone funciones con nombres de negocio:

```go
// ✅ repository/cases.go
func ListPending(flowSource string) ([]model.ConfirmationCase, error) {
    rows, _ := Pool.Query(ctx, "SELECT ... FROM confirmation_cases WHERE ...", flowSource)
    // scan y devolver structs
}
```

```go
// ✅ handler/cases.go
func (h *CaseHandler) ListPending(c *gin.Context) {
    items, _ := repository.ListPending("")  // ← no sabe que hay SQL atrás
    c.JSON(200, gin.H{"items": items})
}
```

### Beneficio real en entrevista

"Separé el acceso a datos en una capa repository. Si mañana cambiamos de PostgreSQL a otra base, solo toco el repository. El handler y el service no se enteran. Además puedo testear el service con un mock del repository sin necesitar una DB real."

---

## 9. Service layer: donde vive la lógica de negocio

El service es el "cerebro" de la aplicación. No sabe HTTP ni SQL. Solo sabe reglas de negocio.

### Responsabilidades del service

1. **Validar reglas de negocio** (ej: "no se puede aprobar un caso ya cancelado")
2. **Orquestar** (ej: "para aprobar: primero dispatch webhook, después update status, después audit log")
3. **Llamar a servicios externos** (ej: webhook a n8n)
4. **Coordinar múltiples repositorios**

### Ejemplo

```go
func (s *CaseService) Approve(caseID int64, userEmail string) (*model.ConfirmationCase, error) {
    c, err := repository.GetCaseByID(caseID)
    if !model.CanTransition(c.Status, model.StatusApproved) {
        return nil, fmt.Errorf("no se puede aprobar un caso en estado '%s'", c.Status)
    }

    if s.N8NWebhookURL != "" {
        if err := s.dispatchToN8N(c, "send"); err != nil {
            _ = repository.InsertAuditLog(c.ID, "webhook_failed", &userEmail, ...)
            return nil, fmt.Errorf("error al contactar n8n: %w", err)
        }
    }

    updated, _ := repository.UpdateCaseStatus(caseID, model.StatusApproved, userEmail)
    _ = repository.InsertAuditLog(caseID, "approved", &userEmail, ...)
    return updated, nil
}
```

Notá que **el service no depende de Gin** (no recibe `*gin.Context`). Esto significa que podés llamarlo desde un test, desde un CLI, desde otro service — sin HTTP de por medio.

---

## 10. Máquina de estados finita

Cada caso de confirmación tiene un **estado** que solo puede cambiar por ciertas transiciones:

```
                    ┌─────────┐
                    │ pending │ ← estado inicial (n8n lo crea así)
                    └────┬────┘
               ┌─────────┼─────────┐
               ▼         ▼         ▼
          ┌────────┐ ┌──────┐ ┌──────────┐
          │approved│ │skipped│ │cancelled │
          └────────┘ └──────┘ └──────────┘
               │                   │
               ▼                   ▼
          webhook a n8n      webhook a n8n
          (manda WA)         (marca cancel)
```

Y además:
```
pending ──(pasa el tiempo)──▶ expired
```

### ¿Qué pasa si se intenta una transición inválida?

Por ejemplo: aprobar un caso que ya fue cancelado.

```go
func CanTransition(from, to CaseStatus) bool {
    targets := ValidTransitions[from]  // pending → [approved, skipped, cancelled, expired]
    for _, t := range targets {
        if t == to { return true }
    }
    return false  // cancelado → approved? NO. Devuelve false.
}
```

El service recibe un 409 Conflict: `"no se puede aprobar un caso en estado 'cancelled'"`.

### ¿Por qué esto importa en una entrevista?

Porque muestra que pensaste en **consistencia de datos**. La app no puede quedar en un estado inválido aunque el usuario haga doble tap, aunque dos operadores abran el mismo caso, aunque el webhook falle.

---

## 11. Idempotencia: cómo evitar duplicados

### El problema

¿Qué pasa si n8n manda el mismo caso dos veces? (Por un retry, un bug, o una re-ejecución.)

### La solución: idempotency_key

Cada caso tiene una clave única: `cita_id + flow_source`. Guardamos esto en `idempotency_key`.

```sql
idempotency_key TEXT UNIQUE NOT NULL
```

Cuando n8n hace POST para crear un caso, usamos `ON CONFLICT` en vez de `INSERT` puro:

```sql
INSERT INTO confirmation_cases (idempotency_key, ...) VALUES ($1, ...)
ON CONFLICT (idempotency_key) DO UPDATE SET
    ai_reason = EXCLUDED.ai_reason,
    chat_context = EXCLUDED.chat_context,
    suggested_message = EXCLUDED.suggested_message,
    status = 'pending',          -- resetea a pending si ya estaba resuelto
    resolved_at = NULL,
    created_at = NOW()
```

Esto significa:
- Si el caso **no existe** → se inserta nuevo
- Si el caso **ya existe** → se actualiza con los datos nuevos y se resetea a pending

**Resultado**: n8n puede mandar el mismo caso 100 veces. Siempre hay un solo row en la DB, con los datos más recientes.

### En una entrevista

"El sistema es idempotente. Usé un `idempotency_key` único basado en `cita_id + flow_source` con `ON CONFLICT DO UPDATE`. n8n puede retryar el POST sin crear duplicados. Es un patrón estándar en APIs de pago como Stripe."

---

## 12. JWT: autenticación sin sesiones

### ¿Qué es JWT?

JSON Web Token: un token firmado que contiene claims (datos). El servidor lo genera al hacer login y el cliente lo manda en cada request.

```
Cliente                          Servidor
  │                                │
  │ POST /auth/login {email, pass} │
  │ ─────────────────────────────▶ │
  │                                │ valida password
  │                                │ genera JWT firmado
  │ ◀───────────────────────────── │
  │ token: "eyJhbGciOi..."         │
  │                                │
  │ GET /cases/pending             │
  │ Authorization: Bearer eyJ...   │
  │ ─────────────────────────────▶ │
  │                                │ verifica firma
  │                                │ extrae claims
  │ ◀───────────────────────────── │
  │ { items: [...] }              │
```

### Estructura de un JWT

```
eyJhbGciOiJIUzI1NiJ9           ← header: algoritmo de firma
.eyJlbWFpbCI6ImFAYi5jb20ifQ   ← payload: claims (email, expiración, etc.)
.xXxXxXxXxXxXxXxXxXxXxX       ← firma: hash de header+payload con secreto
```

### Cómo lo implementamos

**Generar token** (`auth/jwt.go:14`):
```go
func GenerateToken(email, secret string) (string, error) {
    claims := Claims{
        Email: email,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
            Issuer:    "confirmation-review-service",
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}
```

**Validar token** (`auth/jwt.go:30`):
```go
func ValidateToken(tokenString, secret string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
        return []byte(secret), nil
    })
    claims := token.Claims.(*Claims)
    return claims, nil
}
```

### Por qué JWT y no sesiones

- **Stateless**: el servidor no guarda sesiones. Cada request es autónomo.
- **Escalable**: cualquier instancia del backend puede validar el token.
- **Simple**: no necesitás Redis para sesiones.

---

## 13. Bridge key: autenticación para máquinas

### El problema

n8n también necesita llamar a nuestra API. Pero n8n no es un usuario humano — no tiene email ni contraseña.

### La solución: bridge key

Una clave secreta compartida entre el backend y n8n. Se manda en un header `x-bridge-key`.

```go
func BridgeKeyAuth(expectedKey string) gin.HandlerFunc {
    return func(c *gin.Context) {
        key := c.GetHeader("x-bridge-key")
        if key == "" || key != expectedKey {
            c.AbortWithStatusJSON(401, gin.H{"error": "Bridge key inválida"})
            return
        }
        c.Next()
    }
}
```

En el router:
```go
// Estas rutas requieren bridge key (n8n)
bridge := r.Group("/api")
bridge.Use(auth.BridgeKeyAuth(cfg.BridgeKey))
{
    bridge.POST("/cases", caseHandler.Create)           // n8n encola casos
    bridge.POST("/notify", caseHandler.Notify)          // n8n avisa que terminó
    bridge.POST("/cases/:id/refresh-chat", ...)        // n8n actualiza chat
}
```

### Diferencia JWT vs Bridge Key

| | JWT | Bridge Key |
|---|---|---|
| Para | Humanos (UI) | Máquinas (n8n) |
| Autenticación | Login + token | Key secreta en header |
| Expira | 72 horas | No expira |
| Identidad | Email del usuario | "n8n" |

---

## 14. Webhooks: comunicación entre servicios

### ¿Qué es un webhook?

Una URL que cuando recibe un POST, ejecuta una acción. Es la forma más simple de comunicación entre servicios. Es un "callback HTTP".

### Webhooks en este proyecto

**1. n8n → Go (encolar caso)**
```
n8n POST /api/cases
{ idempotency_key, cita_id, ai_reason, chat_context, suggested_message }
→ Go guarda en DB
```

**2. n8n → Go (notificar fin de workflow)**
```
n8n POST /api/notify
{ flow_source: "citas", pending_count: 3 }
→ Go cuenta casos pending, envía push si hay
```

**3. Go → n8n (ejecutar acción)**
```
Go POST {N8N_PENDING_ACTION_WEBHOOK_URL}
{ action: "send", cita_id, chat_id, account_id, message }
→ n8n manda WA + actualiza Notion
```

### ¿Qué pasa si el webhook falla?

Si Go llama a n8n y n8n está caído, el error se loguea en `case_audit_log` con `action: "webhook_failed"`. El caso **no cambia de estado** — sigue en pending. Esto es importante porque:
- No perdemos la acción
- El operador puede reintentar
- Tenemos registro de qué falló

```go
if err := s.dispatchToN8N(c, "send"); err != nil {
    _ = repository.InsertAuditLog(c.ID, "webhook_failed", &userEmail, ...)
    return nil, fmt.Errorf("error al contactar n8n: %w", err)
}
```

---

## 15. Audit log: trazabilidad de cada acción

Toda acción que modifica un caso queda registrada:

| case_id | action | performed_by | details | created_at |
|---------|--------|-------------|---------|------------|
| 42 | created | NULL | `{"source":"n8n"}` | 09:15 |
| 42 | approved | a@b.com | `{"action":"approved"}` | 09:32 |
| 43 | created | NULL | `{"source":"n8n"}` | 10:00 |
| 43 | webhook_failed | c@d.com | `{"error":"timeout"}` | 10:05 |

### ¿Para qué sirve?

- **Debugging**: "¿quién aprobó este caso y cuándo?"
- **Auditoría real**: podés mostrar quién hizo qué
- **Métricas**: cuántos casos aprueba cada operador por día

---

## 16. Expiración de casos

Un caso no debería estar en pending para siempre. Si pasa más de N horas/días, se marca como expirado.

### ¿Cómo funciona?

Cada caso tiene un campo `expires_at` que se setea al crearse (default: 24 horas). Un endpoint `POST /api/cases/expire` (llamado por un cron en n8n cada N horas) ejecuta:

```sql
UPDATE confirmation_cases
SET status = 'expired', resolved_at = NOW()
WHERE status = 'pending' AND created_at < NOW() - INTERVAL '1 day'
```

### ¿Por qué es importante?

- El operador no ve casos viejos que ya no son relevantes
- La lista de "Por revisar" siempre está fresca
- Evita acumulación infinita de casos

---

## 17. Chat refresh: mantener datos actualizados

### El problema

Cuando el operador abre un caso, ve el contexto del chat que n8n envió al momento de encolarlo. Pero ¿qué pasa si entre que se encoló y que el operador lo revisa, el prospecto mandó un mensaje nuevo? El operador estaría decidiendo con datos viejos.

### La solución: webhook de refresh

n8n (o el sistema de Unipile) puede llamar a `POST /api/cases/:id/refresh-chat` con el chat actualizado:

```
n8n POST /api/cases/42/refresh-chat
{ chat_context: [{...últimos mensajes actualizados...}] }
→ Go actualiza el campo chat_context en la DB
→ Frontend (cuando refreshee) ve los datos nuevos
```

En el futuro se puede agregar polling desde el frontend o Server-Sent Events para que se actualice en tiempo real.

---

## 18. Push notifications: cómo notificar al celular

### El flujo

```
1. n8n termina workflow → POST /api/notify { flow_source: "citas" }
2. Go consulta: hay casos pending? (SELECT COUNT(*)...)
3. Si count > 0 → busca todas las push subscriptions en DB
4. Para cada dispositivo suscripto → envía push via Web Push API
5. El service worker en el celular recibe el push → muestra notificación
6. El usuario toca la notificación → abre la app
```

### ¿Qué es Web Push?

Es un estándar W3C que permite mandar notificaciones a navegadores (Chrome, Firefox, Edge, Safari 16.4+). Funciona incluso con la app cerrada (si el service worker está registrado).

### Componentes necesarios

1. **VAPID keys**: par de claves pública/privada para identificar al servidor de push
2. **Service worker**: archivo JS que corre en background en el navegador
3. **Push subscription**: el navegador se suscribe y nos da un endpoint único
4. **Web Push protocol**: el servidor manda un POST al endpoint del navegador

### Cómo funciona en este proyecto

**1. El usuario acepta notificaciones** (frontend):
```typescript
const registration = await navigator.serviceWorker.register("/sw.js");
const subscription = await registration.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToUint8Array(vapidPublicKey)
});
// Envía subscription al backend
await api("/api/push/subscribe", { method: "POST", body: JSON.stringify(subscription) });
```

**2. El backend guarda la subscription** en `push_subscriptions`.

**3. Cuando n8n notifica fin de workflow**, el backend envía push a todos los dispositivos.

### El service worker

```javascript
// public/sw.js
self.addEventListener("push", (event) => {
    const data = event.data.json();
    event.waitUntil(
        self.registration.showNotification(data.title, {
            body: data.body,
            icon: "/icon-192.png",
            vibrate: [200, 100, 200],
            requireInteraction: true,  // la notificación no desaparece sola
            data: { url: data.url }
        })
    );
});
```

### Limitación actual

El envío real de push desde Go requiere una librería como `github.com/SherClockHolmes/webpush-go`. La implementación base está hecha; para producción hace falta instalar esa dependencia y reemplazar el stub `sendPushToWorker`.

---

## 19. PWA: manifest y service worker

### ¿Por qué PWA?

Una Progressive Web App permite:
- Instalar la app en la home screen del celular
- Abrir en pantalla completa (sin barra de URL)
- Funcionar offline (con cache)
- Recibir push notifications

### manifest.json

```json
{
    "name": "Confirmaciones",
    "start_url": "/review",
    "display": "standalone",       // sin navegador alrededor
    "background_color": "#0b141a",
    "theme_color": "#0b141a",
    "orientation": "portrait",
    "icons": [
        { "src": "/icon-192.png", "sizes": "192x192" },
        { "src": "/icon-512.png", "sizes": "512x512" }
    ]
}
```

`display: standalone` hace que la app se abra como una app nativa, sin la barra de direcciones del navegador.

---

## 20. Edge cases: qué puede fallar y cómo lo manejamos

### 1. n8n manda el mismo caso 2 veces
→ `idempotency_key UNIQUE` con `ON CONFLICT DO UPDATE`. No se duplica.

### 2. El operador hace doble tap en "Aprobar"
→ El frontend deshabilita los botones inmediatamente (`disabled={busy}`). El backend valida `status = 'pending'` antes de cualquier transición. Si ya fue aprobado, devuelve 409.

### 3. El webhook a n8n falla (timeout, 500)
→ Se loguea en `case_audit_log` como `webhook_failed`. El caso sigue en `pending`. El operador puede reintentar.

### 4. Dos operadores abren el mismo caso simultáneamente
→ El primero que actúa gana. El segundo recibe 409 Conflict. El frontend refreshea y el caso ya no aparece.

### 5. JWT expirado mientras el operador revisa un caso
→ El frontend detecta 401, limpia el token, redirige al login.

### 6. El chat cambió después de que se encoló el caso
→ n8n puede llamar a `POST /cases/:id/refresh-chat` para actualizar el contexto. El frontend ve los datos más recientes al refreshear.

### 7. El caso expiró mientras el operador lo estaba viendo
→ El endpoint de approve/cancel/skip valida que el status sea `pending`. Si ya expiró, devuelve 409.

### 8. La notificación push no llega
→ El sistema no depende de las notificaciones para funcionar. El operador puede abrir la app y ver los casos manualmente. Las notificaciones son un "nice to have" de conveniencia.

---

## 21. Cómo defender esto en una entrevista

### El pitch de 60 segundos

> "Construí un **AI-assisted confirmation review system** para un equipo de operaciones. El pipeline automático, corriendo en n8n con Groq, confirma ~80% de las citas por WhatsApp. Para el 20% restante —donde la IA detecta ambigüedad— construí un backend en **Go con Gin y PostgreSQL** que recibe los casos rechazados junto con el contexto del chat y el análisis de la IA. Una **PWA en Next.js** consume esa API y permite al operador revisar y decidir: aprobar (dispara webhook a n8n), cancelar, o skipear. El sistema tiene **idempotencia** (evita duplicados vía idempotency_key), **máquina de estados** (solo transiciones válidas), **audit log** completo, manejo de **edge cases** como webhooks fallidos o doble acción del usuario, y **push notifications** cuando hay casos pendientes."

### Las 5 preguntas que te van a hacer

**1. "¿Por qué Go y no Node?"**
> El sistema existente ya estaba en Node. Separé esta pieza en Go porque necesitaba un servicio con buen manejo de concurrencia para los webhooks, tipado fuerte para evitar errores con los estados (evitás bugs de runtime), y quería crecer hacia backend engineering — Go es el estándar para microservicios en el ecosistema cloud-native.

**2. "¿Cómo manejás que falle el webhook a n8n?"**
> El caso no cambia de estado hasta que el webhook responde 200. Se loguea el intento fallido en `case_audit_log` con timestamp y error. El caso queda como pending para que el operador pueda reintentar. Si quisiera escalar, agregaría un worker con retry exponencial y dead letter queue.

**3. "¿Cómo evitás race conditions?"**
> Por diseño de estados: solo se puede transicionar desde `pending`. Uso `WHERE status = 'pending'` en el UPDATE y valido en capa de service. Si dos operadores actúan sobre el mismo caso, el primero gana y el segundo recibe 409 Conflict. El frontend maneja eso refresheando la lista.

**4. "¿Cómo probaste esto?"**
> El proyecto está estructurado en 3 capas (handler/service/repository) que permiten testing unitario de cada una. El service se puede testear con un mock del repository sin necesitar PostgreSQL. El repository se puede testear con una DB de prueba en Docker. Planeo agregar tests unitarios para la lógica de estados y tests de integración para los endpoints.

**5. "Si tuvieras que escalar esto a 10,000 casos/día, ¿qué cambiarías?"**
> Primero, pondría un message broker (Redis Streams o RabbitMQ) entre n8n y el backend para desacoplar y manejar backpressure. Después, horizontal scaling del backend Go atrás de un load balancer. Y separaría la UI en su propio deployment con CDN. La arquitectura actual ya está pensada para eso: el backend es stateless (JWT sin sesiones) y la DB tiene índices en los campos de búsqueda frecuentes.

---

## 📚 Recursos para profundizar

| Tema | Recurso |
|------|---------|
| Go syntax | [Go by Example](https://gobyexample.com/) |
| Gin framework | [Gin documentation](https://gin-gonic.com/docs/) |
| pgx driver | [pgx GitHub](https://github.com/jackc/pgx) |
| JWT | [jwt.io](https://jwt.io/) |
| Web Push | [Web Push MDN](https://developer.mozilla.org/en-US/docs/Web/API/Push_API) |
| Repository pattern | [Microsoft docs](https://docs.microsoft.com/en-us/dotnet/architecture/microservices/microservice-ddd-cqrs-patterns/infrastructure-persistence-layer-design) |
| Idempotency | [Stripe blog](https://stripe.com/blog/idempotency) |
| Clean Architecture (Go) | [go-clean-arch](https://github.com/bxcodec/go-clean-arch) |
