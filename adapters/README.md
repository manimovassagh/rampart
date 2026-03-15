# Rampart Adapters

Official SDK adapters for integrating with a Rampart IAM server.

## Backend

Server-side middleware for verifying Rampart JWTs on protected API routes.

| Adapter | Package | Framework | Status |
|---------|---------|-----------|--------|
| [Node.js](./backend/node/) | `@rampart-auth/node` | Express >=4 | Ready |
| [Go](./backend/go/) | `rampart` (Go module) | net/http, chi, gin | Ready |
| [Python](./backend/python/) | `rampart-python` | FastAPI, Flask | Ready |
| [Spring Boot](./backend/spring/) | `rampart-spring-boot-starter` | Spring Boot 3.x | Ready |
| [.NET](./backend/dotnet/) | `Rampart.AspNetCore` (NuGet) | ASP.NET Core (.NET 8+) | Ready |
| [Ruby](./backend/ruby/) | `rampart-ruby` (RubyGems) | Rack (Rails, Sinatra) | Ready |
| [PHP](./backend/php/) | `rampart/laravel` (Packagist) | Laravel, PSR-15 | Ready |
| [Rust](./backend/rust/) | `rampart-rust` (crates.io) | Actix-web, Axum | Ready |

## Frontend

Client-side libraries for login, token management, and authenticated API calls.

| Adapter | Package | Framework | Status |
|---------|---------|-----------|--------|
| [Web](./frontend/web/) | `@rampart-auth/web` | Any (vanilla JS/TS) | Ready |
| [React](./frontend/react/) | `@rampart-auth/react` | React >=18 | Ready |
| [Next.js](./frontend/nextjs/) | `@rampart-auth/nextjs` | Next.js 13+ | Ready |
| [Flutter](./frontend/flutter/) | `rampart_flutter` (pub.dev) | iOS, Android, Web | Ready |
| [React Native](./frontend/react-native/) | `@rampart-auth/react-native` (npm) | iOS, Android | Ready |
| [Swift](./frontend/swift/) | `Rampart` (SPM) | iOS, macOS | Ready |
| [Kotlin](./frontend/kotlin/) | `com.rampart` (Maven) | Android | Ready |
