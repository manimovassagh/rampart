# SDK & Integration Guide

Rampart is designed to integrate with any technology stack. The REST API is the universal contract — SDKs and adapters are thin wrappers that make integration easier.

## Architecture

```mermaid
graph TB
    subgraph "Rampart Server"
        API[REST API<br/>OpenAPI 3.0]
    end

    subgraph "Official SDKs (auto-generated from OpenAPI)"
        JS["@rampart/js<br/>TypeScript/JavaScript"]
        GO["rampart-go<br/>Go"]
        PY["rampart-py<br/>Python"]
        JAVA["rampart-java<br/>Java/Kotlin"]
    end

    subgraph "Framework Adapters (thin wrappers around JS SDK)"
        REACT["@rampart/react<br/>React hooks + AuthProvider"]
        VUE["@rampart/vue<br/>Vue composables"]
        ANGULAR["@rampart/angular<br/>Angular module + guards"]
    end

    subgraph "Backend Middleware"
        EXPRESS["@rampart/express<br/>Express.js middleware"]
        GOMW["rampart-go/middleware<br/>Go HTTP middleware"]
        SPRING["rampart-spring<br/>Spring Security"]
        DJANGO["rampart-django<br/>Django middleware"]
        FASTAPI["rampart-fastapi<br/>FastAPI dependency"]
    end

    subgraph "CLI"
        CLI["rampart-cli<br/>Management + Auth"]
    end

    JS --> API
    GO --> API
    PY --> API
    JAVA --> API

    REACT --> JS
    VUE --> JS
    ANGULAR --> JS

    EXPRESS --> JS
    GOMW --> GO
    SPRING --> JAVA
    DJANGO --> PY
    FASTAPI --> PY

    CLI --> API
```

## Integration Tiers

### Tier 0: Direct REST API

Any HTTP client works. No SDK required.

```bash
# Get an access token
curl -X POST https://auth.example.com/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=my-service" \
  -d "client_secret=my-secret" \
  -d "scope=api:read"

# Use the token
curl https://api.example.com/data \
  -H "Authorization: Bearer eyJhbGci..."
```

### Tier 1: Official SDKs

Auto-generated from the OpenAPI spec using [openapi-generator](https://openapi-generator.tech/). Provides typed API clients for every endpoint.

**JavaScript/TypeScript** (`@rampart/js`)

```typescript
import { RampartClient } from "@rampart/js";

const rampart = new RampartClient({
  issuer: "https://auth.example.com",
  clientId: "my-spa",
});

// Start login (redirects to Rampart)
await rampart.loginWithRedirect({
  redirectUri: "https://app.example.com/callback",
  scope: "openid profile email",
});

// Handle callback
const tokens = await rampart.handleRedirectCallback();

// Get user info
const user = await rampart.getUserInfo();

// Admin operations (with admin token)
const adminClient = new RampartAdminClient({
  issuer: "https://auth.example.com",
  accessToken: adminToken,
});

const users = await adminClient.users.list({ limit: 20 });
```

**Go** (`rampart-go`)

```go
import "github.com/manimovassagh/rampart-go"

client := rampart.NewClient("https://auth.example.com",
    rampart.WithClientCredentials("my-service", "my-secret"),
)

// Get a token
token, err := client.TokenFromClientCredentials(ctx, "api:read")

// Introspect a token
info, err := client.IntrospectToken(ctx, accessToken)

// Admin operations
admin := rampart.NewAdminClient("https://auth.example.com", adminToken)
users, err := admin.Users.List(ctx, &rampart.ListUsersParams{Limit: 20})
```

**Python** (`rampart-py`)

```python
from rampart import RampartClient, RampartAdmin

# Client credentials
client = RampartClient(
    issuer="https://auth.example.com",
    client_id="my-service",
    client_secret="my-secret",
)
token = client.get_token(scope="api:read")

# Admin
admin = RampartAdmin(issuer="https://auth.example.com", access_token=admin_token)
users = admin.users.list(limit=20)
```

**Java/Kotlin** (`rampart-java`)

```java
var client = RampartClient.builder()
    .issuer("https://auth.example.com")
    .clientCredentials("my-service", "my-secret")
    .build();

var token = client.getToken("api:read");

var admin = RampartAdmin.builder()
    .issuer("https://auth.example.com")
    .accessToken(adminToken)
    .build();

var users = admin.users().list(ListUsersParams.builder().limit(20).build());
```

### Tier 2: Framework Adapters

Thin wrappers around the JS SDK that provide framework-specific patterns.

**React** (`@rampart/react`)

```tsx
import { AuthProvider, useAuth, useUser, ProtectedRoute } from "@rampart/react";

// Wrap your app
function App() {
  return (
    <AuthProvider
      issuer="https://auth.example.com"
      clientId="my-spa"
      redirectUri="https://app.example.com/callback"
    >
      <Router />
    </AuthProvider>
  );
}

// Use in components
function Dashboard() {
  const { isAuthenticated, login, logout } = useAuth();
  const { user, isLoading } = useUser();

  if (!isAuthenticated) return <button onClick={login}>Login</button>;
  if (isLoading) return <div>Loading...</div>;

  return (
    <div>
      <p>Welcome, {user.name}</p>
      <button onClick={logout}>Logout</button>
    </div>
  );
}

// Protected routes
<ProtectedRoute path="/dashboard" component={Dashboard} />
```

**Vue** (`@rampart/vue`)

```vue
<script setup>
import { useAuth, useUser } from "@rampart/vue";

const { isAuthenticated, login, logout } = useAuth();
const { user } = useUser();
</script>

<template>
  <div v-if="isAuthenticated">
    <p>Welcome, {{ user.name }}</p>
    <button @click="logout">Logout</button>
  </div>
  <button v-else @click="login">Login</button>
</template>
```

**Angular** (`@rampart/angular`)

```typescript
// app.module.ts
import { RampartModule } from "@rampart/angular";

@NgModule({
  imports: [
    RampartModule.forRoot({
      issuer: "https://auth.example.com",
      clientId: "my-spa",
      redirectUri: "https://app.example.com/callback",
    }),
  ],
})
export class AppModule {}

// Route guard
const routes: Routes = [
  { path: "dashboard", component: DashboardComponent, canActivate: [AuthGuard] },
];

// In components
@Component({ ... })
export class DashboardComponent {
  constructor(private auth: RampartAuthService) {}
  user$ = this.auth.user$;
}
```

### Tier 3: Backend Middleware

Drop-in authentication middleware for backend frameworks.

**Express.js** (`@rampart/express`)

```typescript
import { rampartAuth, requireScope } from "@rampart/express";

const app = express();

// Validate JWT on all routes
app.use(rampartAuth({
  issuer: "https://auth.example.com",
  audience: "my-api",
}));

// Require specific scopes
app.get("/api/data", requireScope("api:read"), (req, res) => {
  // req.auth contains the verified token claims
  res.json({ userId: req.auth.sub });
});
```

**Go middleware** (`rampart-go/middleware`)

```go
import "github.com/manimovassagh/rampart-go/middleware"

mw := middleware.New(middleware.Config{
    Issuer:   "https://auth.example.com",
    Audience: "my-api",
})

router.Use(mw.Authenticate)

router.Get("/api/data", mw.RequireScope("api:read"), func(w http.ResponseWriter, r *http.Request) {
    claims := middleware.ClaimsFromContext(r.Context())
    // Use claims.Subject, claims.Scope, etc.
})
```

**Spring Security** (`rampart-spring`)

```java
@Configuration
public class SecurityConfig {
    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        return http
            .oauth2ResourceServer(oauth2 -> oauth2
                .jwt(jwt -> jwt
                    .issuerUri("https://auth.example.com")
                ))
            .authorizeHttpRequests(auth -> auth
                .requestMatchers("/api/**").authenticated()
                .anyRequest().permitAll()
            )
            .build();
    }
}
```

**Django** (`rampart-django`)

```python
# settings.py
RAMPART = {
    "ISSUER": "https://auth.example.com",
    "AUDIENCE": "my-api",
}

MIDDLEWARE = [
    ...
    "rampart_django.middleware.RampartAuthMiddleware",
]

# views.py
from rampart_django.decorators import require_scope

@require_scope("api:read")
def data_view(request):
    user_id = request.auth.sub
    return JsonResponse({"user": user_id})
```

**FastAPI** (`rampart-fastapi`)

```python
from rampart_fastapi import RampartAuth, require_scope

auth = RampartAuth(issuer="https://auth.example.com", audience="my-api")

@app.get("/api/data")
async def get_data(claims=Depends(auth)):
    return {"user": claims.sub}

@app.get("/api/admin")
async def admin_action(claims=Depends(require_scope("admin"))):
    return {"admin": claims.sub}
```

## CLI Tool (`rampart-cli`)

A command-line tool for managing Rampart instances and developer authentication.

```bash
# Install
go install github.com/manimovassagh/rampart/cmd/rampart-cli@latest
# or
brew install rampart-cli

# Authenticate (uses Device Flow)
rampart-cli login --issuer https://auth.example.com

# Manage resources
rampart-cli users list
rampart-cli users create --email jane@example.com --username jane
rampart-cli clients list
rampart-cli clients create --name "my-app" --type public

# Export/import configuration
rampart-cli export --org acme > acme-config.yaml
rampart-cli import --org staging < acme-config.yaml

# Health check
rampart-cli status --issuer https://auth.example.com
```

The CLI uses the Device Authorization Flow for authentication — it opens a browser for the user to log in, then receives the token back. This is the same flow used by `gh auth login`, `gcloud auth login`, and `az login`.

## SDK Generation Strategy

All SDKs are auto-generated from `docs/api/openapi.yaml` using [openapi-generator](https://openapi-generator.tech/):

```bash
# Generate TypeScript SDK
openapi-generator generate \
  -i docs/api/openapi.yaml \
  -g typescript-fetch \
  -o sdks/js/src/generated

# Generate Go SDK
openapi-generator generate \
  -i docs/api/openapi.yaml \
  -g go \
  -o sdks/go

# Generate Python SDK
openapi-generator generate \
  -i docs/api/openapi.yaml \
  -g python \
  -o sdks/python

# Generate Java SDK
openapi-generator generate \
  -i docs/api/openapi.yaml \
  -g java \
  -o sdks/java
```

The generated code handles HTTP calls, serialization, and error mapping. The SDK packages add:
- Token management (caching, automatic refresh)
- PKCE flow helpers
- Framework-specific patterns (hooks, guards, middleware)

## Token Validation in Backend Middleware

All backend middleware validates tokens the same way:

1. Extract token from `Authorization: Bearer <token>` header
2. Fetch JWKS from `/.well-known/jwks.json` (cached with auto-refresh)
3. Verify JWT signature against the public key
4. Verify standard claims: `iss`, `aud`, `exp`, `iat`
5. Optionally verify `scope` for endpoint-level authorization
6. Attach verified claims to the request context

No call back to Rampart is needed for validation — JWTs are self-contained. This is critical for performance in high-throughput services.

For opaque tokens or when immediate revocation is needed, use the introspection endpoint instead.

## Roadmap

| Phase | Deliverables |
|-------|-------------|
| **Phase 1** | REST API, OpenAPI spec, `rampart-cli` |
| **Phase 2** | `@rampart/js`, `rampart-go`, Go middleware, Express middleware |
| **Phase 3** | React/Vue/Angular adapters, Python/Java SDKs, Spring/Django/FastAPI middleware |
