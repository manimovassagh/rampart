# Rampart PHP/Laravel Middleware

[![Packagist Version](https://img.shields.io/packagist/v/rampart/laravel.svg)](https://packagist.org/packages/rampart/laravel)
[![PHP Version](https://img.shields.io/packagist/php-v/rampart/laravel.svg)](https://packagist.org/packages/rampart/laravel)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/manimovassagh/rampart/blob/main/adapters/backend/php/LICENSE)

JWT verification middleware for [Rampart](https://github.com/manimovassagh/rampart) IAM server. Works with **Laravel 10+** out of the box and also as standalone **PSR-15** middleware for any framework.

## Features

- **JWT verification** -- validates RS256-signed tokens against Rampart's JWKS endpoint
- **Laravel integration** -- auto-registered route middleware via service provider
- **PSR-15 support** -- implements `MiddlewareInterface` for use with any PSR-15 compatible framework
- **Role-based access control (RBAC)** -- enforce required roles with `rampart.roles:admin,editor`
- **JWKS caching** -- file-based key caching with configurable TTL to minimize network calls
- **Typed claims** -- verified tokens return a `RampartClaims` object with full type hints
- **Consistent error responses** -- `401`/`403` JSON errors matching Rampart's error format

## Requirements

- PHP 8.1 or later
- Laravel 10+ (for Laravel integration) or any PSR-15 compatible framework
- A running [Rampart](https://github.com/manimovassagh/rampart) IAM server

## Installation

```bash
composer require rampart/laravel
```

The service provider is auto-discovered by Laravel. No manual registration needed.

### Publish Configuration (Optional)

```bash
php artisan vendor:publish --tag=rampart-config
```

This creates `config/rampart.php` where you can customize settings.

## Configuration

Set these environment variables in your `.env`:

```env
RAMPART_ISSUER=https://auth.example.com
RAMPART_AUDIENCE=my-api          # optional
RAMPART_JWKS_CACHE_TTL=300       # optional, default: 300 seconds
```

## Quick Start (Laravel)

### Protect Routes with JWT Verification

```php
use Illuminate\Support\Facades\Route;

// All routes in this group require a valid Rampart JWT
Route::middleware('rampart')->group(function () {
    Route::get('/me', function (\Illuminate\Http\Request $request) {
        $claims = $request->attributes->get('rampart');
        return [
            'user_id' => $claims->sub,
            'email'   => $claims->email,
            'roles'   => $claims->roles,
        ];
    });
});
```

### Role-Based Access Control

```php
// Require "admin" role
Route::middleware(['rampart', 'rampart.roles:admin'])->group(function () {
    Route::get('/admin/users', [AdminController::class, 'listUsers']);
});

// Require multiple roles
Route::middleware(['rampart', 'rampart.roles:admin,editor'])->group(function () {
    Route::put('/articles/{id}', [ArticleController::class, 'update']);
});
```

### Access Claims in Controllers

```php
namespace App\Http\Controllers;

use Illuminate\Http\Request;
use Rampart\Laravel\RampartClaims;

class UserController extends Controller
{
    public function profile(Request $request)
    {
        /** @var RampartClaims $claims */
        $claims = $request->attributes->get('rampart');

        return response()->json([
            'user_id'  => $claims->sub,
            'email'    => $claims->email,
            'org_id'   => $claims->orgId,
            'username' => $claims->preferredUsername,
            'roles'    => $claims->roles,
        ]);
    }
}
```

## PSR-15 Usage (Standalone)

For non-Laravel frameworks that support PSR-15 (Slim, Mezzio, etc.):

```php
use Rampart\Laravel\RampartMiddleware;
use Rampart\Laravel\RequireRoles;

// JWT verification middleware
$auth = new RampartMiddleware(
    issuer: 'https://auth.example.com',
    audience: 'my-api',       // optional
    cacheTtl: 300,            // optional
);

// RBAC middleware
$rbac = new RequireRoles('admin', 'editor');

// Add to your PSR-15 middleware pipeline:
// 1. $auth (verifies JWT, sets 'rampart' attribute)
// 2. $rbac (checks roles from 'rampart' attribute)
```

### Direct Token Verification

```php
use Rampart\Laravel\RampartMiddleware;

$auth = new RampartMiddleware(issuer: 'https://auth.example.com');
$claims = $auth->verifyToken($rawJwtString);

echo $claims->sub;       // "user-123"
echo $claims->email;     // "user@example.com"
print_r($claims->roles); // ["admin", "user"]
echo $claims->orgId;     // "org-456"
```

## Claims

Verified tokens return a `RampartClaims` object:

| Property            | Type          | Description                    |
|---------------------|---------------|--------------------------------|
| `sub`               | `string`      | Subject (user ID)              |
| `iss`               | `string`      | Issuer URL                     |
| `iat`               | `int`         | Issued-at timestamp            |
| `exp`               | `int`         | Expiration timestamp           |
| `orgId`             | `?string`     | Organization / tenant ID       |
| `preferredUsername`  | `?string`     | Username                       |
| `email`             | `?string`     | Email address                  |
| `emailVerified`     | `?bool`       | Whether email is verified      |
| `givenName`         | `?string`     | First name                     |
| `familyName`        | `?string`     | Last name                      |
| `roles`             | `string[]`    | Assigned roles                 |

### Helper Methods

```php
$claims->hasRoles('admin', 'editor'); // true if user has ALL listed roles
$claims->missingRoles('admin', 'editor'); // returns array of missing role names
$claims->toArray(); // convert to associative array
```

## Error Responses

On authentication failure, returns `401` JSON:

```json
{
    "detail": "Token has expired"
}
```

On authorization failure (missing roles), returns `403` JSON:

```json
{
    "detail": "Missing required roles: admin, editor"
}
```

**401 error messages:**

- `"Missing or invalid Authorization header"` -- no `Authorization: Bearer` header
- `"Token has expired"` -- the JWT expiration (`exp`) has passed
- `"Invalid token: <reason>"` -- signature, issuer, or other validation failed
- `"Authentication required before role check"` -- RBAC middleware used without auth middleware

**403 error messages:**

- `"Missing required roles: <role1>, <role2>"` -- the token lacks one or more required roles

## Running Tests

```bash
composer install
./vendor/bin/phpunit tests/
```

## License

This project is licensed under the MIT License. See the [LICENSE](https://github.com/manimovassagh/rampart/blob/main/adapters/backend/php/LICENSE) file for details.

## Links

- [Rampart IAM Server](https://github.com/manimovassagh/rampart)
- [Packagist Package](https://packagist.org/packages/rampart/laravel)
- [Issue Tracker](https://github.com/manimovassagh/rampart/issues)
