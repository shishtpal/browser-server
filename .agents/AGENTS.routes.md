# How to Add a New Route

Adding a new API route involves touching **6 files** (plus `internal/db/db.go` for entirely new domains):

### 1. Define the model (`internal/models/models.go`)

Add your request/response structs with JSON tags. For import endpoints, create a dedicated result struct:

```go
type MyDomain struct {
  ID        int       `json:"id"`
  UserID    int       `json:"user_id"`
  Name      string    `json:"name"`
  CreatedAt time.Time `json:"created_at"`
}
```

### 2. Create the handler (`internal/handlers/<domain>.go`)

Each handler file groups all CRUD functions for a domain. Handlers follow these conventions:
- Function signature: `func HandlerName(w http.ResponseWriter, r *http.Request)`
- Use `helpers.GetUserIDFromQuery(r)` for `?user_id=` filtering
- Use `helpers.GetIDFromPath(r)` for `{id}` path params
- Query the global DB var from `internal/db` (e.g., `db.HistoryDB`)
- Return JSON with `json.NewEncoder(w).Encode(...)`
- Set `w.Header().Set("Content-Type", "application/json")` before writing
- For file uploads, use `r.ParseMultipartForm()` + `r.FormFile("file")`
- Use `http.Error(w, "message", httpStatusCode)` for errors

### 3. Register the route (`cmd/server/main.go`)

API routes are registered on the auth-protected `api` subrouter (`api := r.PathPrefix("/api").Subrouter()`), so use **relative** paths (no `/api` prefix) — the subrouter adds it and `middleware.Auth` covers them automatically:

```go
api.HandleFunc("/mydomain", handlers.GetMyDomain).Methods("GET")
api.HandleFunc("/mydomain", handlers.CreateMyDomain).Methods("POST")
api.HandleFunc("/mydomain/{id}", handlers.GetMyDomainByID).Methods("GET")
api.HandleFunc("/mydomain/{id}", handlers.UpdateMyDomain).Methods("PUT")
api.HandleFunc("/mydomain/{id}", handlers.DeleteMyDomain).Methods("DELETE")
```

Only register on `r` directly for public, unauthenticated endpoints (like `/health`).

### 4. Add route description (`internal/handlers/routes.go`)

Add a `models.Route` entry so the `/api/routes` endpoint reflects the new route:

```go
{Method: "GET", Path: "/api/mydomain", Description: "Get all mydomain entries (filter: user_id)"},
```

### 5. Add the client method (`shared/browser-client/src/client.ts`)

Prefer adding the method to the **shared client** so both the web app and the extension can use it. The shared `apiFetch`/raw `fetch` calls must pass `getToken` so the Bearer header is attached:

```typescript
getMyDomain(userId?: number): Promise<MyDomain[]> {
  return apiFetch<MyDomain[]>(normalizedBaseUrl, 'GET', `/api/mydomain${buildQuery({ user_id: userId })}`, undefined, getToken)
}
```

Add any new types to `shared/browser-types/src/index.ts` (re-exported by `frontend/src/types.ts`). Then expose a thin wrapper in [`frontend/src/lib/api/index.ts`](../frontend/src/lib/api/index.ts); any remaining raw `fetch` calls there must include `...authHeaders()` (from `frontend/src/lib/auth.ts`).

### Checklist

- [ ] Model struct in `internal/models/models.go`
- [ ] Handler functions in `internal/handlers/<domain>.go`
- [ ] Route registered on the `api` subrouter in `cmd/server/main.go`
- [ ] Route description in `internal/handlers/routes.go`
- [ ] Client method in `shared/browser-client/src/client.ts` (passes `getToken`)
- [ ] Types in `shared/browser-types/src/index.ts`
- [ ] Thin wrapper in `frontend/src/lib/api/index.ts` (raw fetches include `authHeaders()`)
- [ ] For new domains: SQLite DB init in `internal/db/db.go` (global var + `Init*DB()` + wire into `InitAll`/`CloseAll`)
- [ ] Go builds without errors (`go build ./cmd/server`)
- [ ] Web/extension components use the new client method as needed

For cross-domain search endpoints like `/api/search/omnibox`, keep the response type normalized and source-tagged rather than leaking raw domain models. Add the shared client method first and have the extension/frontend call that method instead of duplicating fetch logic.
