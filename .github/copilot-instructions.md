# Rampart Auth — AI Integration Instructions

Rampart is a self-hosted OAuth 2.0 / OpenID Connect server with multi-tenant support, PKCE, MFA, and role-based access control.

## JWT Claims Structure

Every Rampart access token contains these claims:

| Claim | Type | Description |
|-------|------|-------------|
| `sub` | UUID | User ID |
| `iss` | string | Issuer URL (your Rampart instance) |
| `aud` | string | Audience (client_id or issuer URL) |
| `iat` | number | Issued at (Unix timestamp) |
| `exp` | number | Expiry (Unix timestamp) |
| `org_id` | UUID | Organization/tenant ID |
| `preferred_username` | string | Username |
| `email` | string | User email |
| `email_verified` | boolean | Whether email is verified |
| `roles` | string[] | Assigned roles (optional) |

## Which Adapter to Use

| Your Framework | Package | Install |
|---------------|---------|---------|
| Express / Node.js | `@rampart-auth/node` | `npm i @rampart-auth/node` |
| Go (net/http) | `github.com/.../adapters/backend/go` | `go get github.com/manimovassagh/rampart/adapters/backend/go` |
| FastAPI / Flask | `rampart-python` | `pip install rampart-python` |
| Spring Boot | `com.rampart:rampart-spring-boot-starter` | Maven/Gradle (see pom.xml) |
| ASP.NET Core | `Rampart.AspNetCore` | `dotnet add package Rampart.AspNetCore` |
| React (SPA) | `@rampart-auth/react` | `npm i @rampart-auth/react` |
| Next.js | `@rampart-auth/nextjs` | `npm i @rampart-auth/nextjs` |
| Vanilla JS / any SPA | `@rampart-auth/web` | `npm i @rampart-auth/web` |

## Minimal Integration Patterns

### Express (Node.js)
```ts
import { rampartAuth } from "@rampart-auth/node";
const auth = rampartAuth({ issuer: "https://auth.example.com" });
app.get("/api/me", auth, (req, res) => res.json(req.user));
```

### Go (net/http)
```go
import rampart "github.com/manimovassagh/rampart/adapters/backend/go"
auth := rampart.New(rampart.Config{Issuer: "https://auth.example.com"})
http.Handle("/api/me", auth.Protect(handler))
```

### FastAPI (Python)
```python
from rampart import RampartAuth
auth = RampartAuth(issuer="https://auth.example.com")
@app.get("/api/me")
async def me(user=Depends(auth)): return user
```

### Flask (Python)
```python
from rampart.flask import require_auth
@app.route("/api/me")
@require_auth(issuer="https://auth.example.com")
def me(): return jsonify(g.rampart_user)
```

### Spring Boot
```yaml
# application.yml
rampart:
  issuer: https://auth.example.com
```
```java
@GetMapping("/api/me")
public Claims me(@AuthenticationPrincipal RampartClaims claims) { return claims; }
```

### ASP.NET Core
```csharp
builder.Services.AddRampartAuth(o => o.Issuer = "https://auth.example.com");
app.MapGet("/api/me", [Authorize] (ClaimsPrincipal u) => u.Claims);
```

### React
```tsx
import { RampartProvider, useAuth } from "@rampart-auth/react";
<RampartProvider issuer="https://auth.example.com" clientId="my-app" redirectUri="/callback">
  <App />
</RampartProvider>
// In components: const { user, login, logout } = useAuth();
```

### Next.js
```ts
// middleware.ts
import { withRampartAuth } from "@rampart-auth/nextjs";
export default withRampartAuth({ issuer: "https://auth.example.com", clientId: "my-app" });
```

### Vanilla JS
```js
import { RampartClient } from "@rampart-auth/web";
const client = new RampartClient({ issuer: "https://auth.example.com", clientId: "my-app", redirectUri: "/callback" });
await client.loginWithRedirect();
```

## Error Format

All adapters use the OAuth 2.0 error format (RFC 6749):

```json
{ "error": "invalid_token", "error_description": "Token has expired", "status": 401 }
```

TypeScript type: `{ error: string; error_description: string; status: number }`

## Required Configuration

**Backend adapters** need only:
- `issuer` — your Rampart server URL (e.g., `https://auth.example.com`)

**Frontend adapters** need:
- `issuer` — your Rampart server URL
- `clientId` — OAuth client ID registered in Rampart
- `redirectUri` — where to redirect after login (e.g., `/callback`)

## Common Mistakes to Avoid

1. **Do not validate JWTs yourself.** Use the adapter — it handles JWKS fetching, key rotation, and signature verification.
2. **Do not store tokens in localStorage.** The frontend adapters use sessionStorage with PKCE. Do not override this.
3. **Do not hardcode the JWKS URL.** The adapter discovers it from `{issuer}/.well-known/openid-configuration`.
4. **Always check `exp` before trusting a token.** The adapters do this automatically; if parsing manually, compare `exp` against `Date.now() / 1000`.
5. **Use `org_id` for tenant isolation.** Every query to your database should filter by the `org_id` claim from the token.
6. **Do not skip PKCE.** Rampart requires PKCE for all public clients. The frontend adapters handle this automatically.
7. **Handle token refresh.** Use `authFetch()` (web) or the adapter's built-in refresh — do not re-authenticate on every 401.
