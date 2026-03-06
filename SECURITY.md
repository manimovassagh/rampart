# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest release | Yes |
| Older releases | Best effort |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

If you discover a security vulnerability in Rampart, please report it responsibly:

1. **Email:** Send details to the maintainer via the email listed on the [GitHub profile](https://github.com/manimovassagh).
2. **Include:** A description of the vulnerability, steps to reproduce, affected versions, and potential impact.
3. **Response time:** We aim to acknowledge reports within 48 hours and provide a fix or mitigation plan within 7 days for critical issues.

## What Qualifies

- Authentication or authorization bypasses
- Token leakage, session fixation, or session hijacking
- SQL injection, XSS, CSRF, or SSRF
- Cryptographic weaknesses
- Privilege escalation
- Information disclosure (secrets, PII, internal state)

## What Does Not Qualify

- Denial of service via resource exhaustion (unless trivially exploitable)
- Issues in dependencies without a demonstrated exploit path in Rampart
- Social engineering attacks
- Issues requiring physical access to the server

## Disclosure Policy

- We will coordinate disclosure with the reporter.
- We will credit reporters in the release notes (unless they prefer anonymity).
- We aim to release fixes before or alongside public disclosure.

## Security Practices

Rampart follows security-first development practices:

- All dependencies are pinned and audited (`govulncheck`, `gosec`, Trivy).
- CI runs security scans on every PR.
- Secrets are never stored in plaintext.
- All authentication flows follow OAuth 2.0 / OIDC RFCs.
