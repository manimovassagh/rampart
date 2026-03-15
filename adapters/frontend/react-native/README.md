# @rampart-auth/react-native

React Native authentication adapter for [Rampart](https://github.com/manimovassagh/rampart) IAM server. Provides a context provider and `useAuth` hook for OAuth 2.0 PKCE authentication in mobile apps.

## Installation

```bash
npm install @rampart-auth/react-native @rampart-auth/web @react-native-async-storage/async-storage
```

For Expo projects:

```bash
npx expo install @rampart-auth/react-native @rampart-auth/web @react-native-async-storage/async-storage
```

### Crypto Polyfill

React Native does not provide the Web Crypto API by default. You need a polyfill for PKCE code challenge generation:

**Expo:**

```bash
npx expo install expo-crypto
```

Then add to your app entry point (before any Rampart imports):

```ts
import "expo-crypto";
```

**Bare React Native:**

```bash
npm install react-native-get-random-values
```

Then add to your app entry point:

```ts
import "react-native-get-random-values";
```

## Usage with Expo

### 1. Configure deep linking

In your `app.json`:

```json
{
  "expo": {
    "scheme": "myapp"
  }
}
```

### 2. Wrap your app with RampartProvider

```tsx
// App.tsx
import { RampartProvider } from "@rampart-auth/react-native";
import * as Linking from "expo-linking";
import { useEffect } from "react";
import { HomeScreen } from "./screens/HomeScreen";

const RAMPART_ISSUER = "https://auth.example.com";
const CLIENT_ID = "my-mobile-app";
const REDIRECT_URI = "myapp://auth/callback";

export default function App() {
  return (
    <RampartProvider
      issuer={RAMPART_ISSUER}
      clientId={CLIENT_ID}
      redirectUri={REDIRECT_URI}
    >
      <AuthCallbackHandler />
      <HomeScreen />
    </RampartProvider>
  );
}

function AuthCallbackHandler() {
  const { handleCallback } = useAuth();

  useEffect(() => {
    // Handle deep link when app is opened from OAuth redirect
    const subscription = Linking.addEventListener("url", async (event) => {
      if (event.url.startsWith(REDIRECT_URI)) {
        await handleCallback(event.url);
      }
    });

    // Handle the case where the app was opened via a deep link (cold start)
    Linking.getInitialURL().then((url) => {
      if (url && url.startsWith(REDIRECT_URI)) {
        handleCallback(url);
      }
    });

    return () => subscription.remove();
  }, [handleCallback]);

  return null;
}
```

### 3. Use the useAuth hook

```tsx
// screens/HomeScreen.tsx
import { View, Text, Button } from "react-native";
import { useAuth } from "@rampart-auth/react-native";

export function HomeScreen() {
  const { user, isAuthenticated, isLoading, loginWithRedirect, logout } =
    useAuth();

  if (isLoading) {
    return <Text>Loading...</Text>;
  }

  if (!isAuthenticated) {
    return (
      <View>
        <Text>Welcome to MyApp</Text>
        <Button title="Sign In" onPress={loginWithRedirect} />
      </View>
    );
  }

  return (
    <View>
      <Text>Hello, {user?.email}</Text>
      <Button title="Sign Out" onPress={logout} />
    </View>
  );
}
```

## Usage with Bare React Native

### 1. Configure deep linking

Follow the [React Native deep linking guide](https://reactnative.dev/docs/linking) to set up your URL scheme.

**iOS** (`ios/MyApp/Info.plist`):

```xml
<key>CFBundleURLTypes</key>
<array>
  <dict>
    <key>CFBundleURLSchemes</key>
    <array>
      <string>myapp</string>
    </array>
  </dict>
</array>
```

**Android** (`android/app/src/main/AndroidManifest.xml`):

```xml
<intent-filter>
  <action android:name="android.intent.action.VIEW" />
  <category android:name="android.intent.category.DEFAULT" />
  <category android:name="android.intent.category.BROWSABLE" />
  <data android:scheme="myapp" android:host="auth" android:pathPrefix="/callback" />
</intent-filter>
```

### 2. Set up the provider and callback handler

```tsx
// App.tsx
import { useEffect } from "react";
import { Linking } from "react-native";
import { RampartProvider, useAuth } from "@rampart-auth/react-native";

const REDIRECT_URI = "myapp://auth/callback";

function CallbackHandler() {
  const { handleCallback } = useAuth();

  useEffect(() => {
    const subscription = Linking.addEventListener("url", async (event) => {
      if (event.url.startsWith(REDIRECT_URI)) {
        await handleCallback(event.url);
      }
    });

    Linking.getInitialURL().then((url) => {
      if (url && url.startsWith(REDIRECT_URI)) {
        handleCallback(url);
      }
    });

    return () => subscription.remove();
  }, [handleCallback]);

  return null;
}

export default function App() {
  return (
    <RampartProvider
      issuer="https://auth.example.com"
      clientId="my-mobile-app"
      redirectUri={REDIRECT_URI}
    >
      <CallbackHandler />
      {/* Your app screens */}
    </RampartProvider>
  );
}
```

## API Reference

### `<RampartProvider>`

| Prop          | Type      | Required | Default    | Description                                |
|---------------|-----------|----------|------------|--------------------------------------------|
| `issuer`      | `string`  | Yes      |            | Rampart server URL                         |
| `clientId`    | `string`  | Yes      |            | OAuth 2.0 client ID                        |
| `redirectUri` | `string`  | Yes      |            | Deep link URI for OAuth callback           |
| `scope`       | `string`  | No       | `"openid"` | OAuth 2.0 scopes                           |
| `persist`     | `boolean` | No       | `true`     | Persist tokens to AsyncStorage             |

### `useAuth()`

Returns an object with:

| Property              | Type                                              | Description                                     |
|-----------------------|---------------------------------------------------|-------------------------------------------------|
| `user`                | `RampartUser \| null`                             | Current user profile                            |
| `isAuthenticated`     | `boolean`                                         | Whether a user is logged in                     |
| `isLoading`           | `boolean`                                         | Whether tokens are being restored               |
| `loginWithRedirect()` | `() => Promise<void>`                             | Opens system browser for OAuth login            |
| `handleCallback(url)` | `(url: string) => Promise<void>`                  | Exchanges authorization code for tokens         |
| `logout()`            | `() => Promise<void>`                             | Clears tokens and logs out                      |
| `getAccessToken()`    | `() => string \| null`                            | Returns the current access token                |
| `authFetch(url, init)`| `(url: string, init?: RequestInit) => Promise<Response>` | Fetch with auto auth header and token refresh |

## Key Differences from @rampart-auth/react

| Feature            | @rampart-auth/react         | @rampart-auth/react-native          |
|--------------------|-----------------------------|--------------------------------------|
| Token storage      | `localStorage`              | `AsyncStorage`                       |
| PKCE storage       | `sessionStorage`            | `AsyncStorage`                       |
| OAuth redirect     | `window.location.href`      | `Linking.openURL` (system browser)   |
| Callback handling  | Auto-detects from `window.location` | Requires explicit URL parameter |
| Deep linking       | Not needed                  | Required (URL scheme or universal links) |

## Registering Your Mobile Client

When registering your OAuth client with Rampart, use a custom URL scheme as the redirect URI:

```
myapp://auth/callback
```

Make sure the redirect URI registered on the server exactly matches the one used in your app.

## License

MIT
