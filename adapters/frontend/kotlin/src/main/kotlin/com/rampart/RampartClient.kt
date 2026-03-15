package com.rampart

import android.content.Context
import android.content.SharedPreferences
import android.net.Uri
import androidx.browser.customtabs.CustomTabsIntent
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import io.ktor.client.HttpClient
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.request.forms.submitForm
import io.ktor.client.request.get
import io.ktor.client.request.header
import io.ktor.client.request.request
import io.ktor.client.request.setBody
import io.ktor.client.statement.HttpResponse
import io.ktor.client.statement.bodyAsText
import io.ktor.http.ContentType
import io.ktor.http.HttpMethod
import io.ktor.http.contentType
import io.ktor.http.parameters
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import java.security.MessageDigest
import java.security.SecureRandom
import android.util.Base64

/**
 * Configuration for [RampartClient].
 *
 * @param issuer Rampart server URL (e.g. "https://auth.example.com").
 * @param clientId OAuth 2.0 client ID registered with the Rampart server.
 * @param redirectUri OAuth 2.0 redirect URI -- must match a registered URI.
 *   For Android, use a custom scheme (e.g. "com.example.app://callback").
 * @param scope OAuth 2.0 scopes (default: "openid").
 * @param httpClient Optional custom Ktor [HttpClient] for testing.
 */
data class RampartClientConfig(
    val issuer: String,
    val clientId: String,
    val redirectUri: String,
    val scope: String = "openid",
    val httpClient: HttpClient? = null,
)

/**
 * Token set returned by login and refresh operations.
 */
@Serializable
data class RampartTokens(
    @SerialName("access_token") val accessToken: String,
    @SerialName("refresh_token") val refreshToken: String,
    @SerialName("token_type") val tokenType: String = "Bearer",
    @SerialName("expires_in") val expiresIn: Int = 3600,
)

/**
 * Native Android authentication client for Rampart IAM.
 *
 * Implements OAuth 2.0 Authorization Code flow with PKCE using Chrome Custom Tabs
 * for the browser redirect. Tokens are stored in [EncryptedSharedPreferences].
 *
 * Usage:
 * ```kotlin
 * val client = RampartClient(context, RampartClientConfig(
 *     issuer = "https://auth.example.com",
 *     clientId = "my-android-app",
 *     redirectUri = "com.example.app://callback",
 * ))
 *
 * // Restore previous session
 * client.loadStoredTokens()
 *
 * // Start login
 * client.loginWithRedirect(activity)
 *
 * // In your Activity's onNewIntent / deep-link handler:
 * val tokens = client.handleCallback(intent.data!!)
 * val user = client.getUser()
 * ```
 */
class RampartClient(
    context: Context,
    config: RampartClientConfig,
) {
    private val issuer: String = config.issuer.trimEnd('/')
    private val clientId: String = config.clientId
    private val redirectUri: String = config.redirectUri
    private val scope: String = config.scope

    private val httpClient: HttpClient = config.httpClient ?: HttpClient(OkHttp)

    private val json = Json {
        ignoreUnknownKeys = true
        encodeDefaults = true
    }

    private val prefs: SharedPreferences

    private var tokens: RampartTokens? = null

    init {
        val masterKey = MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()

        prefs = EncryptedSharedPreferences.create(
            context,
            PREFS_FILE,
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
    }

    // -------------------------------------------------------------------------
    // Public API
    // -------------------------------------------------------------------------

    /** Whether the client holds a non-expired access token. */
    val isAuthenticated: Boolean
        get() {
            val token = tokens?.accessToken ?: return false
            return try {
                val parts = token.split(".")
                if (parts.size != 3) return false
                val payload = decodeBase64Url(parts[1])
                val claims = json.decodeFromString<JwtExpClaims>(payload)
                val exp = claims.exp ?: return false
                (exp * 1000L) > System.currentTimeMillis()
            } catch (_: Exception) {
                false
            }
        }

    /** Returns the current access token, or null. */
    val accessToken: String?
        get() = tokens?.accessToken

    /** Returns the current token set, or null. */
    val currentTokens: RampartTokens?
        get() = tokens

    /**
     * Load tokens from encrypted storage. Call once at app startup to restore
     * a previous session.
     */
    fun loadStoredTokens() {
        val stored = prefs.getString(KEY_TOKENS, null) ?: return
        tokens = try {
            json.decodeFromString<RampartTokens>(stored)
        } catch (_: Exception) {
            prefs.edit().remove(KEY_TOKENS).apply()
            null
        }
    }

    /**
     * Start the OAuth login flow by opening the authorization URL in a
     * Chrome Custom Tab.
     *
     * After the user authenticates, the browser will redirect back to the app
     * via the custom URI scheme in [RampartClientConfig.redirectUri].
     * Handle the redirect in your Activity and call [handleCallback].
     *
     * @param context An Activity context used to launch the Custom Tab.
     */
    fun loginWithRedirect(context: Context) {
        val codeVerifier = generateCodeVerifier()
        val codeChallenge = generateCodeChallenge(codeVerifier)
        val state = generateState()

        // Persist PKCE values so they survive the browser redirect.
        prefs.edit()
            .putString(KEY_CODE_VERIFIER, codeVerifier)
            .putString(KEY_STATE, state)
            .apply()

        val authorizeUrl = Uri.parse("$issuer/oauth/authorize").buildUpon()
            .appendQueryParameter("response_type", "code")
            .appendQueryParameter("client_id", clientId)
            .appendQueryParameter("redirect_uri", redirectUri)
            .appendQueryParameter("scope", scope)
            .appendQueryParameter("state", state)
            .appendQueryParameter("code_challenge", codeChallenge)
            .appendQueryParameter("code_challenge_method", "S256")
            .build()

        val customTabsIntent = CustomTabsIntent.Builder().build()
        customTabsIntent.launchUrl(context, authorizeUrl)
    }

    /**
     * Handle the OAuth callback after the browser redirects back to the app.
     *
     * Pass the full callback [Uri] (including query parameters). This method
     * validates the state parameter, exchanges the authorization code for
     * tokens, and persists them in encrypted storage.
     *
     * @param uri The callback URI from the deep-link intent.
     * @return The token set received from the server.
     */
    suspend fun handleCallback(uri: Uri): RampartTokens {
        val code = uri.getQueryParameter("code")
        val state = uri.getQueryParameter("state")
        val error = uri.getQueryParameter("error")

        if (error != null) {
            val description = uri.getQueryParameter("error_description") ?: error
            throw RampartError.OAuthError(error, description)
        }

        if (code == null || state == null) {
            throw RampartError.InvalidCallback()
        }

        // Validate state.
        val storedState = prefs.getString(KEY_STATE, null)
        if (storedState == null || storedState != state) {
            cleanupPkceStorage()
            throw RampartError.StateMismatch()
        }

        val codeVerifier = prefs.getString(KEY_CODE_VERIFIER, null)
        if (codeVerifier == null) {
            cleanupPkceStorage()
            throw RampartError.MissingVerifier()
        }

        // Exchange code for tokens.
        val response: HttpResponse = httpClient.submitForm(
            url = "$issuer/oauth/token",
            formParameters = parameters {
                append("grant_type", "authorization_code")
                append("code", code)
                append("client_id", clientId)
                append("redirect_uri", redirectUri)
                append("code_verifier", codeVerifier)
            },
        )

        cleanupPkceStorage()

        val body = response.bodyAsText()
        if (response.status.value != 200) {
            throw RampartError.fromResponse(body, response.status.value)
        }

        val newTokens = json.decodeFromString<RampartTokens>(body)
        setTokens(newTokens)
        return newTokens
    }

    /**
     * Fetch the authenticated user profile from the `/me` endpoint.
     */
    suspend fun getUser(): RampartClaims {
        val response = authFetch("$issuer/me")
        val body = response.bodyAsText()
        if (response.status.value != 200) {
            throw RampartError.fromResponse(body, response.status.value)
        }
        return json.decodeFromString<RampartClaims>(body)
    }

    /**
     * Make an authenticated HTTP request with automatic Bearer token.
     *
     * On a 401 response, attempts one silent token refresh and retries.
     *
     * @param url The request URL.
     * @param method HTTP method (default: GET).
     * @param headers Additional headers.
     * @param body Optional request body (String).
     * @param contentType Content type for the body.
     */
    suspend fun authFetch(
        url: String,
        method: HttpMethod = HttpMethod.Get,
        headers: Map<String, String> = emptyMap(),
        body: String? = null,
        contentType: ContentType? = null,
    ): HttpResponse {
        suspend fun doFetch(): HttpResponse {
            return httpClient.request(url) {
                this.method = method
                header("Authorization", "Bearer ${tokens?.accessToken}")
                headers.forEach { (k, v) -> header(k, v) }
                if (body != null) {
                    contentType?.let { contentType(it) }
                    setBody(body)
                }
            }
        }

        var response = doFetch()

        if (response.status.value == 401 && tokens?.refreshToken != null) {
            try {
                refresh()
                response = doFetch()
            } catch (_: Exception) {
                // Refresh failed -- return the original 401.
            }
        }

        return response
    }

    /**
     * Refresh the access token using the stored refresh token.
     */
    suspend fun refresh(): RampartTokens {
        val refreshToken = tokens?.refreshToken
            ?: throw RampartError.NoRefreshToken()

        val response: HttpResponse = httpClient.request("$issuer/token/refresh") {
            method = HttpMethod.Post
            contentType(ContentType.Application.Json)
            setBody(json.encodeToString(RefreshRequest.serializer(), RefreshRequest(refreshToken)))
        }

        val body = response.bodyAsText()
        if (response.status.value != 200) {
            setTokens(null)
            throw RampartError.fromResponse(body, response.status.value)
        }

        val data = json.decodeFromString<RefreshResponse>(body)
        val updated = RampartTokens(
            accessToken = data.accessToken,
            refreshToken = refreshToken,
            tokenType = data.tokenType,
            expiresIn = data.expiresIn,
        )
        setTokens(updated)
        return updated
    }

    /**
     * Logout -- invalidates the refresh token on the server and clears local
     * tokens.
     */
    suspend fun logout() {
        val refreshToken = tokens?.refreshToken
        if (refreshToken != null) {
            try {
                httpClient.request("$issuer/logout") {
                    method = HttpMethod.Post
                    contentType(ContentType.Application.Json)
                    setBody(json.encodeToString(RefreshRequest.serializer(), RefreshRequest(refreshToken)))
                }
            } catch (_: Exception) {
                // Best-effort server logout.
            }
        }
        setTokens(null)
    }

    // -------------------------------------------------------------------------
    // Private helpers
    // -------------------------------------------------------------------------

    private fun setTokens(newTokens: RampartTokens?) {
        tokens = newTokens
        if (newTokens != null) {
            prefs.edit()
                .putString(KEY_TOKENS, json.encodeToString(RampartTokens.serializer(), newTokens))
                .apply()
        } else {
            prefs.edit().remove(KEY_TOKENS).apply()
        }
    }

    private fun cleanupPkceStorage() {
        prefs.edit()
            .remove(KEY_CODE_VERIFIER)
            .remove(KEY_STATE)
            .apply()
    }

    /** Generate a random 64-character code verifier (base64url-encoded). */
    private fun generateCodeVerifier(): String {
        val bytes = ByteArray(48)
        SecureRandom().nextBytes(bytes)
        return base64UrlEncode(bytes)
    }

    /** Compute S256 code challenge: BASE64URL(SHA256(verifier)). */
    private fun generateCodeChallenge(verifier: String): String {
        val digest = MessageDigest.getInstance("SHA-256").digest(verifier.toByteArray(Charsets.UTF_8))
        return base64UrlEncode(digest)
    }

    /** Generate a random state parameter for CSRF protection. */
    private fun generateState(): String {
        val bytes = ByteArray(32)
        SecureRandom().nextBytes(bytes)
        return base64UrlEncode(bytes)
    }

    /** URL-safe base64 encoding without padding. */
    private fun base64UrlEncode(bytes: ByteArray): String {
        return Base64.encodeToString(bytes, Base64.URL_SAFE or Base64.NO_PADDING or Base64.NO_WRAP)
    }

    /** Decode a base64url-encoded string to a UTF-8 string. */
    private fun decodeBase64Url(input: String): String {
        val bytes = Base64.decode(input, Base64.URL_SAFE or Base64.NO_PADDING or Base64.NO_WRAP)
        return String(bytes, Charsets.UTF_8)
    }

    companion object {
        private const val PREFS_FILE = "rampart_secure_prefs"
        private const val KEY_TOKENS = "rampart_tokens"
        private const val KEY_CODE_VERIFIER = "rampart_pkce_code_verifier"
        private const val KEY_STATE = "rampart_pkce_state"
    }
}

// ---------------------------------------------------------------------------
// Internal DTOs
// ---------------------------------------------------------------------------

@Serializable
internal data class JwtExpClaims(
    @SerialName("exp") val exp: Long? = null,
)

@Serializable
internal data class RefreshRequest(
    @SerialName("refresh_token") val refreshToken: String,
)

@Serializable
internal data class RefreshResponse(
    @SerialName("access_token") val accessToken: String,
    @SerialName("token_type") val tokenType: String = "Bearer",
    @SerialName("expires_in") val expiresIn: Int = 3600,
)
