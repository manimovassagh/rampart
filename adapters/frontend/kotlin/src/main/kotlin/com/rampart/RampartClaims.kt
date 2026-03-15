package com.rampart

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * User profile claims returned by the Rampart `/me` endpoint.
 */
@Serializable
data class RampartClaims(
    /** User ID (UUID), corresponds to the JWT "sub" claim. */
    @SerialName("id") val sub: String,

    /** Organization ID (UUID). */
    @SerialName("org_id") val orgId: String,

    /** Preferred username. */
    @SerialName("preferred_username") val preferredUsername: String? = null,

    /** Email address. */
    @SerialName("email") val email: String,

    /** Whether the email has been verified. */
    @SerialName("email_verified") val emailVerified: Boolean = false,

    /** Roles assigned to the user. */
    @SerialName("roles") val roles: List<String> = emptyList(),

    /** First name. */
    @SerialName("given_name") val givenName: String? = null,

    /** Last name. */
    @SerialName("family_name") val familyName: String? = null,

    /** Whether the account is active. */
    @SerialName("enabled") val enabled: Boolean? = null,

    /** Account creation timestamp (ISO 8601). */
    @SerialName("created_at") val createdAt: String? = null,

    /** Last update timestamp (ISO 8601). */
    @SerialName("updated_at") val updatedAt: String? = null,
)
