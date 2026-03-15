# rampart-ruby

[![Gem Version](https://img.shields.io/gem/v/rampart-ruby.svg)](https://rubygems.org/gems/rampart-ruby)
[![license](https://img.shields.io/gem/l/rampart-ruby.svg)](https://github.com/manimovassagh/rampart/blob/main/adapters/backend/ruby/LICENSE)

Rack middleware for verifying [Rampart](https://github.com/manimovassagh/rampart) JWTs. Handles JWKS fetching, RS256 verification, and claim extraction with zero configuration beyond the issuer URL.

## Features

- **Zero-config JWT verification** -- just provide the issuer URL
- **Automatic JWKS fetching and caching** from the Rampart discovery endpoint
- **RS256 signature, issuer, and expiry validation** using the [jwt](https://github.com/jwt/ruby-jwt) gem
- **Role-based access control** with the `RequireRoles` middleware
- **Works with any Rack app** -- Rails, Sinatra, Hanami, Roda, plain Rack
- **Standardized error responses** matching the Rampart error format

## Install

Add to your Gemfile:

```ruby
gem "rampart-ruby"
```

Then run:

```bash
bundle install
```

## Quick Start

### Rails

```ruby
# config/application.rb
config.middleware.use Rampart::Middleware, issuer: "http://localhost:8080"
```

```ruby
# app/controllers/api/profile_controller.rb
class Api::ProfileController < ApplicationController
  def show
    claims = request.env["rampart.claims"]
    render json: { user_id: claims.sub, org: claims.org_id }
  end
end
```

### Sinatra

```ruby
require "sinatra"
require "rampart"

use Rampart::Middleware, issuer: "http://localhost:8080"

get "/api/me" do
  claims = env["rampart.claims"]
  json user_id: claims.sub, org: claims.org_id
end
```

### Plain Rack

```ruby
# config.ru
require "rampart"

use Rampart::Middleware, issuer: "http://localhost:8080"

app = lambda do |env|
  claims = env["rampart.claims"]
  body = JSON.generate(user_id: claims.sub)
  [200, { "content-type" => "application/json" }, [body]]
end

run app
```

## API

### `Rampart::Middleware`

Rack middleware that:

1. Extracts the Bearer token from the `Authorization` header
2. Fetches the JWKS from `{issuer}/.well-known/jwks.json` (cached for 10 minutes)
3. Verifies the RS256 signature, issuer, and expiry
4. Sets `env['rampart.claims']` with a `Rampart::Claims` instance

#### Options

| Option   | Type     | Description                                      |
|----------|----------|--------------------------------------------------|
| `issuer` | `String` | Rampart server URL (e.g. `http://localhost:8080`) |

### `Rampart::Claims`

Available via `env['rampart.claims']` after successful verification:

| Method               | Type       | Description                |
|----------------------|------------|----------------------------|
| `iss`                | `String`   | Issuer URL                 |
| `sub`                | `String`   | User ID (UUID)             |
| `iat`                | `Numeric`  | Issued at (Unix timestamp) |
| `exp`                | `Numeric`  | Expires at (Unix timestamp)|
| `org_id`             | `String`   | Organization ID (UUID)     |
| `preferred_username` | `String`   | Username                   |
| `email`              | `String`   | Email address              |
| `email_verified`     | `Boolean`  | Whether email is verified  |
| `given_name`         | `String?`  | First name (if set)        |
| `family_name`        | `String?`  | Last name (if set)         |
| `roles`              | `Array<String>` | Assigned roles        |

### `Rampart::RequireRoles`

Rack middleware that checks the authenticated user has ALL specified roles. Use after `Rampart::Middleware`:

```ruby
# Rails
config.middleware.use Rampart::Middleware, issuer: "http://localhost:8080"
config.middleware.use Rampart::RequireRoles, "admin"
```

```ruby
# Per-route with Rack::Builder
map "/admin" do
  use Rampart::Middleware, issuer: "http://localhost:8080"
  use Rampart::RequireRoles, "admin"
  run AdminApp
end
```

Returns 403 with `{ error: "forbidden", error_description: "Missing required role(s): ...", status: 403 }` if the user lacks the required roles.

### Error Responses

On failure the middleware returns a `401` JSON response matching Rampart's error format:

```json
{
  "error": "unauthorized",
  "error_description": "Missing authorization header.",
  "status": 401
}
```

Error messages:
- `"Missing authorization header."` -- no `Authorization` header
- `"Invalid authorization header format."` -- not a `Bearer` token
- `"Invalid or expired access token."` -- signature, issuer, or expiry check failed

## License

MIT
