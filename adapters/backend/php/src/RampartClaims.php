<?php

declare(strict_types=1);

namespace Rampart\Laravel;

/**
 * Verified JWT claims from a Rampart token.
 *
 * Mirrors the claims structure used across all Rampart adapter SDKs.
 */
final class RampartClaims
{
    /**
     * @param string      $sub               Subject (user ID)
     * @param string      $iss               Issuer URL
     * @param int         $iat               Issued-at timestamp
     * @param int         $exp               Expiration timestamp
     * @param string|null $orgId             Organization / tenant ID
     * @param string|null $preferredUsername  Username
     * @param string|null $email             Email address
     * @param bool|null   $emailVerified     Whether email is verified
     * @param string|null $givenName         First name
     * @param string|null $familyName        Last name
     * @param string[]    $roles             Assigned roles
     */
    public function __construct(
        public readonly string $sub,
        public readonly string $iss,
        public readonly int $iat,
        public readonly int $exp,
        public readonly ?string $orgId = null,
        public readonly ?string $preferredUsername = null,
        public readonly ?string $email = null,
        public readonly ?bool $emailVerified = null,
        public readonly ?string $givenName = null,
        public readonly ?string $familyName = null,
        public readonly array $roles = [],
    ) {}

    /**
     * Build a RampartClaims instance from a decoded JWT payload.
     *
     * @param object|array<string, mixed> $payload Decoded JWT payload
     * @return self
     */
    public static function fromToken(object|array $payload): self
    {
        $data = (array) $payload;

        $roles = $data['roles'] ?? [];
        if (is_string($roles)) {
            $roles = [$roles];
        }

        return new self(
            sub: (string) ($data['sub'] ?? ''),
            iss: (string) ($data['iss'] ?? ''),
            iat: (int) ($data['iat'] ?? 0),
            exp: (int) ($data['exp'] ?? 0),
            orgId: isset($data['org_id']) ? (string) $data['org_id'] : null,
            preferredUsername: isset($data['preferred_username']) ? (string) $data['preferred_username'] : null,
            email: isset($data['email']) ? (string) $data['email'] : null,
            emailVerified: isset($data['email_verified']) ? (bool) $data['email_verified'] : null,
            givenName: isset($data['given_name']) ? (string) $data['given_name'] : null,
            familyName: isset($data['family_name']) ? (string) $data['family_name'] : null,
            roles: array_values(array_map('strval', (array) $roles)),
        );
    }

    /**
     * Check whether this token has ALL of the given roles.
     *
     * @param string ...$requiredRoles
     * @return bool
     */
    public function hasRoles(string ...$requiredRoles): bool
    {
        foreach ($requiredRoles as $role) {
            if (!in_array($role, $this->roles, true)) {
                return false;
            }
        }
        return true;
    }

    /**
     * Return the claims that are missing from the token.
     *
     * @param string ...$requiredRoles
     * @return string[]
     */
    public function missingRoles(string ...$requiredRoles): array
    {
        $missing = [];
        foreach ($requiredRoles as $role) {
            if (!in_array($role, $this->roles, true)) {
                $missing[] = $role;
            }
        }
        sort($missing);
        return $missing;
    }

    /**
     * Convert claims to an associative array.
     *
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'sub' => $this->sub,
            'iss' => $this->iss,
            'iat' => $this->iat,
            'exp' => $this->exp,
            'org_id' => $this->orgId,
            'preferred_username' => $this->preferredUsername,
            'email' => $this->email,
            'email_verified' => $this->emailVerified,
            'given_name' => $this->givenName,
            'family_name' => $this->familyName,
            'roles' => $this->roles,
        ];
    }
}
