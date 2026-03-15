package com.rampart

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/**
 * Errors raised by [RampartClient] or returned by the Rampart server.
 *
 * Uses a sealed class hierarchy so callers can use exhaustive `when` matching.
 */
sealed class RampartError(
    /** Machine-readable error code (e.g. "invalid_callback", "state_mismatch"). */
    val error: String,
    /** Human-readable error description. */
    val errorDescription: String,
    /** HTTP status code, or 0 for client-side errors. */
    val status: Int = 0,
) : Exception("$error: $errorDescription [HTTP $status]") {

    /** Error returned by the Rampart server in an HTTP response. */
    class ServerError(
        error: String,
        errorDescription: String,
        status: Int,
    ) : RampartError(error, errorDescription, status)

    /** OAuth callback contained an error parameter. */
    class OAuthError(
        error: String,
        errorDescription: String,
    ) : RampartError(error, errorDescription)

    /** Missing or invalid parameters in the OAuth callback URL. */
    class InvalidCallback(
        errorDescription: String = "Missing code or state parameter in callback URL.",
    ) : RampartError("invalid_callback", errorDescription)

    /** PKCE state mismatch -- possible CSRF. */
    class StateMismatch(
        errorDescription: String = "State parameter does not match. Possible CSRF attack.",
    ) : RampartError("state_mismatch", errorDescription)

    /** No refresh token available to perform a refresh. */
    class NoRefreshToken(
        errorDescription: String = "No refresh token available.",
    ) : RampartError("no_refresh_token", errorDescription)

    /** Failed to launch the browser for authorization. */
    class LaunchFailed(
        errorDescription: String = "Could not open the authorization URL in a browser.",
    ) : RampartError("launch_failed", errorDescription)

    /** PKCE code verifier was not found in storage. */
    class MissingVerifier(
        errorDescription: String = "Code verifier not found in secure storage.",
    ) : RampartError("missing_verifier", errorDescription)

    /** Catch-all for unexpected errors. */
    class Unknown(
        errorDescription: String,
        status: Int = 0,
    ) : RampartError("unknown_error", errorDescription, status)

    companion object {
        private val json = Json { ignoreUnknownKeys = true }

        /**
         * Parse an error from a server JSON response body.
         * Falls back to a generic message if parsing fails.
         */
        fun fromResponse(body: String, status: Int): RampartError {
            return try {
                val dto = json.decodeFromString<ErrorDto>(body)
                ServerError(
                    error = dto.error ?: "unknown_error",
                    errorDescription = dto.errorDescription ?: "An unknown error occurred.",
                    status = status,
                )
            } catch (_: Exception) {
                Unknown(errorDescription = "HTTP $status", status = status)
            }
        }
    }
}

/** Internal DTO used only for JSON deserialization of server error responses. */
@Serializable
internal data class ErrorDto(
    @SerialName("error") val error: String? = null,
    @SerialName("error_description") val errorDescription: String? = null,
)
