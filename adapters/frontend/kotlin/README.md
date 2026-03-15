# Rampart Android SDK (Kotlin)

Native Android/Kotlin adapter for [Rampart](https://github.com/manimovassagh/rampart) IAM server. Implements OAuth 2.0 Authorization Code flow with PKCE, encrypted token storage via `EncryptedSharedPreferences`, and browser-based login via Chrome Custom Tabs.

## Installation

Add the dependency to your app-level `build.gradle.kts`:

```kotlin
dependencies {
    implementation("com.rampart:rampart-android:0.1.0")
}
```

## Quick Start

### 1. Configure the client

```kotlin
import com.rampart.RampartClient
import com.rampart.RampartClientConfig

val config = RampartClientConfig(
    issuer = "https://auth.example.com",
    clientId = "my-android-app",
    redirectUri = "com.example.myapp://callback",
    scope = "openid",
)

val rampart = RampartClient(applicationContext, config)
```

### 2. Restore a previous session

Call `loadStoredTokens()` at app startup (e.g. in `onCreate`) to restore tokens from encrypted storage:

```kotlin
rampart.loadStoredTokens()

if (rampart.isAuthenticated) {
    // User is already logged in
}
```

### 3. Login

```kotlin
// Opens Chrome Custom Tab with the Rampart authorize page
rampart.loginWithRedirect(activity)
```

### 4. Handle the callback

Register a deep-link intent filter in your `AndroidManifest.xml`:

```xml
<activity
    android:name=".CallbackActivity"
    android:exported="true"
    android:launchMode="singleTop">
    <intent-filter>
        <action android:name="android.intent.action.VIEW" />
        <category android:name="android.intent.category.DEFAULT" />
        <category android:name="android.intent.category.BROWSABLE" />
        <data android:scheme="com.example.myapp" android:host="callback" />
    </intent-filter>
</activity>
```

Then handle the redirect:

```kotlin
override fun onNewIntent(intent: Intent) {
    super.onNewIntent(intent)
    val uri = intent.data ?: return

    lifecycleScope.launch {
        try {
            val tokens = rampart.handleCallback(uri)
            // Login successful -- tokens are stored automatically
        } catch (e: RampartError) {
            // Handle error
        }
    }
}
```

### 5. Get user profile

```kotlin
lifecycleScope.launch {
    val user = rampart.getUser()
    println("Logged in as ${user.email}")
}
```

### 6. Make authenticated requests

```kotlin
lifecycleScope.launch {
    val response = rampart.authFetch("https://api.example.com/data")
    val body = response.bodyAsText()
    // 401 responses automatically trigger a silent token refresh and retry
}
```

### 7. Refresh tokens

```kotlin
lifecycleScope.launch {
    val newTokens = rampart.refresh()
}
```

### 8. Logout

```kotlin
lifecycleScope.launch {
    rampart.logout()
    // Tokens cleared from memory and encrypted storage
}
```

## Jetpack Compose Integration

Below is a full example of using Rampart with Jetpack Compose and a `ViewModel`:

### ViewModel

```kotlin
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.rampart.RampartClient
import com.rampart.RampartClaims
import com.rampart.RampartError
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch

data class AuthUiState(
    val isAuthenticated: Boolean = false,
    val user: RampartClaims? = null,
    val error: String? = null,
    val isLoading: Boolean = false,
)

class AuthViewModel(private val rampart: RampartClient) : ViewModel() {

    private val _state = MutableStateFlow(AuthUiState())
    val state: StateFlow<AuthUiState> = _state

    init {
        rampart.loadStoredTokens()
        if (rampart.isAuthenticated) {
            fetchUser()
        }
    }

    fun fetchUser() {
        viewModelScope.launch {
            _state.value = _state.value.copy(isLoading = true, error = null)
            try {
                val user = rampart.getUser()
                _state.value = _state.value.copy(
                    isAuthenticated = true,
                    user = user,
                    isLoading = false,
                )
            } catch (e: RampartError) {
                _state.value = _state.value.copy(
                    error = e.errorDescription,
                    isLoading = false,
                )
            }
        }
    }

    fun logout() {
        viewModelScope.launch {
            rampart.logout()
            _state.value = AuthUiState()
        }
    }
}
```

### Composable screens

```kotlin
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp

@Composable
fun AuthScreen(viewModel: AuthViewModel, rampart: RampartClient) {
    val state by viewModel.state.collectAsState()
    val context = LocalContext.current

    if (state.isAuthenticated && state.user != null) {
        ProfileScreen(state.user!!, onLogout = { viewModel.logout() })
    } else {
        LoginScreen(
            isLoading = state.isLoading,
            error = state.error,
            onLogin = { rampart.loginWithRedirect(context as android.app.Activity) },
        )
    }
}

@Composable
fun LoginScreen(isLoading: Boolean, error: String?, onLogin: () -> Unit) {
    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text("Welcome to My App", style = MaterialTheme.typography.headlineMedium)
        Spacer(modifier = Modifier.height(24.dp))
        Button(onClick = onLogin, enabled = !isLoading) {
            Text(if (isLoading) "Loading..." else "Sign In")
        }
        error?.let {
            Spacer(modifier = Modifier.height(16.dp))
            Text(it, color = MaterialTheme.colorScheme.error)
        }
    }
}

@Composable
fun ProfileScreen(user: RampartClaims, onLogout: () -> Unit) {
    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text("Hello, ${user.givenName ?: user.email}!", style = MaterialTheme.typography.headlineMedium)
        Spacer(modifier = Modifier.height(8.dp))
        Text(user.email, style = MaterialTheme.typography.bodyLarge)
        Spacer(modifier = Modifier.height(4.dp))
        Text("Roles: ${user.roles.joinToString()}", style = MaterialTheme.typography.bodyMedium)
        Spacer(modifier = Modifier.height(24.dp))
        OutlinedButton(onClick = onLogout) {
            Text("Sign Out")
        }
    }
}
```

## Error Handling

All errors are subclasses of `RampartError` (a sealed class), so you can use exhaustive `when` matching:

```kotlin
try {
    val user = rampart.getUser()
} catch (e: RampartError) {
    when (e) {
        is RampartError.ServerError -> println("Server: ${e.status} ${e.errorDescription}")
        is RampartError.NoRefreshToken -> println("Session expired, please log in again")
        is RampartError.StateMismatch -> println("Security error: ${e.errorDescription}")
        is RampartError.OAuthError -> println("OAuth error: ${e.error}")
        is RampartError.InvalidCallback -> println("Bad callback URL")
        is RampartError.LaunchFailed -> println("Could not open browser")
        is RampartError.MissingVerifier -> println("PKCE state lost")
        is RampartError.Unknown -> println("Unexpected: ${e.errorDescription}")
    }
}
```

## Security

- Tokens are stored in Android's `EncryptedSharedPreferences` backed by AES-256-GCM encryption with keys managed by the Android Keystore.
- PKCE (S256) is used for all authorization flows to prevent code interception attacks.
- State parameter is validated to prevent CSRF attacks.
- The `authFetch()` method automatically handles 401 responses with a silent token refresh.

## Requirements

- Android API 23+ (Android 6.0 Marshmallow)
- Kotlin 1.9+
- AndroidX
