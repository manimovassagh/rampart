# Rampart Swift Adapter

Native Swift/iOS client for [Rampart](https://github.com/manimovassagh/rampart) IAM server. Implements OAuth 2.0 Authorization Code flow with PKCE using `ASWebAuthenticationSession`, with token storage in the system Keychain.

**Zero external dependencies** -- uses only Foundation, CryptoKit, AuthenticationServices, and Security.

## Installation

### Swift Package Manager

Add the package to your `Package.swift`:

```swift
dependencies: [
    .package(url: "https://github.com/manimovassagh/rampart", from: "1.0.0"),
]
```

Or add it via Xcode: File > Add Package Dependencies, then enter the repository URL and select the `Rampart` library.

## Quick Start (SwiftUI)

### 1. Configure the client

```swift
import Rampart

let rampart = RampartClient(config: .init(
    issuer: "https://auth.example.com",
    clientId: "my-ios-app",
    redirectUri: "com.example.myapp://callback"
))
```

### 2. Create an authentication manager

```swift
import SwiftUI
import Rampart

@MainActor
final class AuthManager: ObservableObject {
    let client: RampartClient

    @Published var isAuthenticated = false
    @Published var user: RampartClaims?
    @Published var error: String?

    init() {
        client = RampartClient(config: .init(
            issuer: "https://auth.example.com",
            clientId: "my-ios-app",
            redirectUri: "com.example.myapp://callback",
            scope: "openid"
        ))
    }

    func restoreSession() {
        client.loadStoredTokens()
        isAuthenticated = client.isAuthenticated
        if isAuthenticated {
            Task { await fetchUser() }
        }
    }

    func login() async {
        do {
            try await client.loginWithRedirect()
            isAuthenticated = true
            await fetchUser()
        } catch {
            self.error = error.localizedDescription
        }
    }

    func fetchUser() async {
        do {
            user = try await client.getUser()
        } catch {
            self.error = error.localizedDescription
        }
    }

    func logout() async {
        await client.logout()
        isAuthenticated = false
        user = nil
    }
}
```

### 3. Use in your SwiftUI views

```swift
@main
struct MyApp: App {
    @StateObject private var auth = AuthManager()

    var body: some Scene {
        WindowGroup {
            if auth.isAuthenticated {
                DashboardView()
                    .environmentObject(auth)
            } else {
                LoginView()
                    .environmentObject(auth)
            }
        }
        .onAppear { auth.restoreSession() }
    }
}

struct LoginView: View {
    @EnvironmentObject var auth: AuthManager

    var body: some View {
        VStack(spacing: 20) {
            Text("Welcome")
                .font(.largeTitle)

            Button("Sign In") {
                Task { await auth.login() }
            }
            .buttonStyle(.borderedProminent)

            if let error = auth.error {
                Text(error)
                    .foregroundColor(.red)
                    .font(.caption)
            }
        }
    }
}

struct DashboardView: View {
    @EnvironmentObject var auth: AuthManager

    var body: some View {
        VStack(spacing: 16) {
            if let user = auth.user {
                Text("Hello, \(user.givenName ?? user.email)")
                    .font(.title)
                Text("Email: \(user.email)")
                Text("Roles: \(user.roles.joined(separator: ", "))")
            }

            Button("Sign Out") {
                Task { await auth.logout() }
            }
            .buttonStyle(.bordered)
        }
    }
}
```

### 4. Register your custom URL scheme

In your Xcode project, go to your target's **Info** tab and add a URL Type:

- **URL Schemes**: `com.example.myapp`
- **Role**: Viewer

This allows iOS to route the OAuth callback (`com.example.myapp://callback`) back to your app.

## API Reference

### `RampartClientConfig`

| Property | Type | Default | Description |
|---|---|---|---|
| `issuer` | `String` | required | Rampart server URL |
| `clientId` | `String` | required | OAuth 2.0 client ID |
| `redirectUri` | `String` | required | Custom scheme callback URL |
| `scope` | `String` | `"openid"` | OAuth 2.0 scopes |
| `urlSession` | `URLSession` | `.shared` | Custom URL session |

### `RampartClient`

| Method | Description |
|---|---|
| `loadStoredTokens()` | Restore tokens from Keychain on app launch |
| `loginWithRedirect(contextProvider:)` | Start OAuth PKCE flow via system browser |
| `handleCallback(_:)` | Exchange authorization code for tokens |
| `getUser()` | Fetch user profile from `/me` |
| `authFetch(_:method:headers:body:)` | Authenticated HTTP request with auto-refresh |
| `refresh()` | Manually refresh the access token |
| `logout()` | Invalidate tokens on server and clear Keychain |

### Properties

| Property | Type | Description |
|---|---|---|
| `isAuthenticated` | `Bool` | `true` if a non-expired access token exists |
| `accessToken` | `String?` | Current access token |

### `RampartClaims`

Codable struct with fields: `sub`, `orgId`, `preferredUsername`, `email`, `emailVerified`, `roles`, `givenName`, `familyName`, `enabled`, `createdAt`, `updatedAt`.

### `RampartError`

| Case | Description |
|---|---|
| `.launchFailed(String)` | Authorization URL could not be opened |
| `.invalidCallback(String)` | Callback URL missing required parameters |
| `.stateMismatch` | CSRF protection: state mismatch |
| `.missingVerifier` | PKCE verifier not found |
| `.noRefreshToken` | No refresh token available |
| `.serverError(error:description:status:)` | Server returned an error |
| `.unknown(String)` | Unexpected error |

## Making Authenticated API Calls

```swift
let (data, response) = try await rampart.authFetch(
    URL(string: "https://api.example.com/protected")!,
    method: "POST",
    headers: ["Content-Type": "application/json"],
    body: try JSONEncoder().encode(["key": "value"])
)
```

The `authFetch` method automatically attaches the Bearer token and retries once with a refreshed token on 401 responses.

## Platform Support

- iOS 15.0+
- macOS 12.0+
- Swift 5.9+
