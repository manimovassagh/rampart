import Foundation

/// User profile claims returned by the Rampart `/me` endpoint.
public struct RampartClaims: Codable, Sendable, Equatable {

    /// User ID (UUID), corresponds to the JWT "sub" claim.
    public let sub: String

    /// Organization ID (UUID).
    public let orgId: String

    /// Preferred username.
    public let preferredUsername: String?

    /// Email address.
    public let email: String

    /// Whether the email has been verified.
    public let emailVerified: Bool

    /// Roles assigned to the user.
    public let roles: [String]

    /// First name.
    public let givenName: String?

    /// Last name.
    public let familyName: String?

    /// Whether the account is active.
    public let enabled: Bool?

    /// Account creation timestamp (ISO 8601).
    public let createdAt: String?

    /// Last update timestamp (ISO 8601).
    public let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case sub = "id"
        case orgId = "org_id"
        case preferredUsername = "preferred_username"
        case email
        case emailVerified = "email_verified"
        case roles
        case givenName = "given_name"
        case familyName = "family_name"
        case enabled
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    public init(
        sub: String,
        orgId: String,
        preferredUsername: String? = nil,
        email: String,
        emailVerified: Bool = false,
        roles: [String] = [],
        givenName: String? = nil,
        familyName: String? = nil,
        enabled: Bool? = nil,
        createdAt: String? = nil,
        updatedAt: String? = nil
    ) {
        self.sub = sub
        self.orgId = orgId
        self.preferredUsername = preferredUsername
        self.email = email
        self.emailVerified = emailVerified
        self.roles = roles
        self.givenName = givenName
        self.familyName = familyName
        self.enabled = enabled
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }
}
