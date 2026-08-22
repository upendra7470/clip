# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 1.0.x   | Yes       |

## Reporting a Vulnerability

Clip parses untrusted documents, so security issues are taken seriously.

**Please do not report security vulnerabilities through public GitHub issues.**

Instead:

1. Open a **private security advisory** via GitHub: *Security → Report a vulnerability* on this repository, or
2. Contact the maintainer directly through the repository owner's GitHub profile.

Please include:

- A description of the vulnerability and its impact
- Steps to reproduce (a sample document that triggers the issue is ideal)
- Affected version(s)

You can expect an initial response within 7 days. We will credit reporters in the fix release unless they prefer to remain anonymous.

## Built-in Defenses

Clip enforces several defenses against malicious documents in `internal/security/`:

- Bounded file reads with maximum input size
- ZIP archive total-size checks (zip-bomb protection)
- XML decoder token limits
- RTF nesting-depth limits

Reports about bypassing these limits are especially welcome.
