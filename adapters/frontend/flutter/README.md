# rampart_flutter

Flutter authentication SDK for [Rampart](https://github.com/manimovassagh/rampart). Implements the OAuth 2.0 Authorization Code flow with PKCE for mobile (iOS/Android) and web platforms.

## Features

- **OAuth 2.0 Authorization Code + PKCE** -- secure auth without exposing client secrets
- **Cross-platform** -- works on iOS, Android, and web
- **Secure token storage** via `flutter_secure_storage` (Keychain on iOS, EncryptedSharedPreferences on Android)
- **Automatic token refresh** with silent retry on 401 responses via `authFetch`
- **JWT expiry checking** via the `isAuthenticated` getter

## Install

Add to your `pubspec.yaml`:

```yaml
dependencies:
  rampart_flutter:
    git:
      url: https://github.com/manimovassagh/rampart.git
      path: adapters/frontend/flutter
```

## Platform Setup

### iOS

Add your custom URL scheme to `ios/Runner/Info.plist`:

```xml
<key>CFBundleURLTypes</key>
<array>
  <dict>
    <key>CFBundleURLSchemes</key>
    <array>
      <string>com.example.app</string>
    </array>
  </dict>
</array>
```

### Android

Add an intent filter to `android/app/src/main/AndroidManifest.xml`:

```xml
<intent-filter>
  <action android:name="android.intent.action.VIEW" />
  <category android:name="android.intent.category.DEFAULT" />
  <category android:name="android.intent.category.BROWSABLE" />
  <data android:scheme="com.example.app" android:host="callback" />
</intent-filter>
```

### Web

No special configuration needed. Use a standard HTTPS redirect URL (e.g. `http://localhost:3000/callback`).

## Quick Start

```dart
import 'package:rampart_flutter/rampart.dart';

// 1. Create the client
final client = RampartClient(RampartClientConfig(
  issuer: 'http://localhost:8080',
  clientId: 'my-flutter-app',
  redirectUri: 'com.example.app://callback',
));

// 2. Restore a previous session on app startup
await client.loadStoredTokens();

// 3. Start login -- opens the system browser
await client.loginWithRedirect();

// 4. Handle the callback URI (from deep link or redirect)
final tokens = await client.handleCallback(callbackUri);

// 5. Fetch the authenticated user
final user = await client.getUser();
print('Hello, ${user.email}');
```

## Handling Deep Links

On mobile, listen for incoming deep links to capture the OAuth callback. A common approach using `app_links` or the platform channel:

```dart
import 'package:app_links/app_links.dart';
import 'package:rampart_flutter/rampart.dart';

final appLinks = AppLinks();
final client = RampartClient(RampartClientConfig(
  issuer: 'http://localhost:8080',
  clientId: 'my-flutter-app',
  redirectUri: 'com.example.app://callback',
));

// Listen for incoming links
appLinks.uriLinkStream.listen((uri) async {
  if (uri.scheme == 'com.example.app' && uri.host == 'callback') {
    final tokens = await client.handleCallback(uri);
    // Navigate to the authenticated area of the app
  }
});
```

## API

### `RampartClient(RampartClientConfig config)`

Creates a client instance. Call `loadStoredTokens()` after construction to restore any persisted session.

#### `RampartClientConfig`

| Property      | Type          | Default    | Description                                                      |
|---------------|---------------|------------|------------------------------------------------------------------|
| `issuer`      | `String`      | --         | Rampart server URL (e.g. `http://localhost:8080`)                |
| `clientId`    | `String`      | --         | OAuth 2.0 client ID registered with the Rampart server           |
| `redirectUri` | `String`      | --         | OAuth 2.0 redirect URI (custom scheme for mobile, HTTPS for web) |
| `scope`       | `String`      | `"openid"` | OAuth 2.0 scopes                                                 |
| `httpClient`  | `http.Client?`| `null`     | Optional HTTP client for testing or custom configuration          |

### Methods

#### `loadStoredTokens() -> Future<void>`

Loads tokens from `flutter_secure_storage`. Call once at app startup.

#### `loginWithRedirect() -> Future<void>`

Generates PKCE code verifier and challenge, stores them in secure storage, and opens the Rampart authorization endpoint in the system browser.

#### `handleCallback(Uri uri) -> Future<RampartTokens>`

Handles the OAuth callback. Validates the state parameter, exchanges the authorization code for tokens, and persists them in secure storage.

#### `getUser() -> Future<RampartClaims>`

Fetches the current user profile from the `/me` endpoint using `authFetch`.

#### `authFetch(Uri url, {String method, Map<String, String>? headers, Object? body}) -> Future<http.Response>`

HTTP request with automatic `Authorization: Bearer` header. On a 401 response, attempts one silent token refresh and retries.

#### `refresh() -> Future<RampartTokens>`

Refreshes the access token using the stored refresh token. Clears tokens on failure.

#### `logout() -> Future<void>`

Invalidates the refresh token on the server and clears local tokens.

### Properties

#### `isAuthenticated -> bool`

Returns `true` if an access token is present and not expired (checks JWT `exp` claim).

#### `accessToken -> String?`

Returns the current access token, or `null`.

#### `tokens -> RampartTokens?`

Returns the current tokens, or `null`.

### `RampartTokens`

| Field          | Type     | Description                    |
|----------------|----------|--------------------------------|
| `accessToken`  | `String` | JWT access token               |
| `refreshToken` | `String` | Refresh token                  |
| `tokenType`    | `String` | Token type (typically `Bearer`) |
| `expiresIn`    | `int`    | Token lifetime in seconds      |

### `RampartClaims`

| Field               | Type           | Description               |
|---------------------|----------------|---------------------------|
| `sub`               | `String`       | User ID (UUID)            |
| `orgId`             | `String`       | Organization ID (UUID)    |
| `preferredUsername` | `String?`      | Preferred username        |
| `email`             | `String`       | Email address             |
| `emailVerified`     | `bool`         | Whether email is verified |
| `roles`             | `List<String>` | Assigned roles            |
| `givenName`         | `String?`      | First name                |
| `familyName`        | `String?`      | Last name                 |
| `enabled`           | `bool?`        | Whether account is active |
| `createdAt`         | `String?`      | ISO 8601 timestamp        |
| `updatedAt`         | `String?`      | ISO 8601 timestamp        |

### `RampartError`

All API errors are thrown as `RampartError` exceptions:

```dart
class RampartError implements Exception {
  final String error;            // e.g. "invalid_callback", "state_mismatch"
  final String errorDescription;
  final int status;              // HTTP status, or 0 for client-side errors
}
```

## License

MIT
