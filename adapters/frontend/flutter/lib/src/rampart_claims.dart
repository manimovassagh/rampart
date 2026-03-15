/// User profile claims returned by the Rampart /me endpoint.
class RampartClaims {
  /// User ID (UUID), corresponds to the JWT "sub" claim.
  final String sub;

  /// Organization ID (UUID).
  final String orgId;

  /// Preferred username.
  final String? preferredUsername;

  /// Email address.
  final String email;

  /// Whether the email has been verified.
  final bool emailVerified;

  /// Roles assigned to the user.
  final List<String> roles;

  /// First name.
  final String? givenName;

  /// Last name.
  final String? familyName;

  /// Whether the account is active.
  final bool? enabled;

  /// Account creation timestamp (ISO 8601).
  final String? createdAt;

  /// Last update timestamp (ISO 8601).
  final String? updatedAt;

  const RampartClaims({
    required this.sub,
    required this.orgId,
    this.preferredUsername,
    required this.email,
    required this.emailVerified,
    this.roles = const [],
    this.givenName,
    this.familyName,
    this.enabled,
    this.createdAt,
    this.updatedAt,
  });

  /// Parse claims from the JSON response of the /me endpoint.
  factory RampartClaims.fromJson(Map<String, dynamic> json) {
    return RampartClaims(
      sub: json['id'] as String,
      orgId: json['org_id'] as String,
      preferredUsername: json['preferred_username'] as String?,
      email: json['email'] as String,
      emailVerified: json['email_verified'] as bool? ?? false,
      roles: (json['roles'] as List<dynamic>?)
              ?.map((e) => e as String)
              .toList() ??
          const [],
      givenName: json['given_name'] as String?,
      familyName: json['family_name'] as String?,
      enabled: json['enabled'] as bool?,
      createdAt: json['created_at'] as String?,
      updatedAt: json['updated_at'] as String?,
    );
  }

  /// Convert claims back to a JSON-compatible map.
  Map<String, dynamic> toJson() {
    return {
      'id': sub,
      'org_id': orgId,
      if (preferredUsername != null) 'preferred_username': preferredUsername,
      'email': email,
      'email_verified': emailVerified,
      if (roles.isNotEmpty) 'roles': roles,
      if (givenName != null) 'given_name': givenName,
      if (familyName != null) 'family_name': familyName,
      if (enabled != null) 'enabled': enabled,
      if (createdAt != null) 'created_at': createdAt,
      if (updatedAt != null) 'updated_at': updatedAt,
    };
  }

  @override
  String toString() => 'RampartClaims(sub: $sub, email: $email)';
}
