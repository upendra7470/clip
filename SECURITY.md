# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 2.0.x   | Yes       |
| 1.1.x   | Security fixes only |
| 1.0.x   | No        |

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
- XML nesting-depth limits
- RTF nesting-depth limits
- OCR image, pixel, and execution-time limits
- SHA-256-verified installers and release provenance attestations

Official release binaries include GitHub build-provenance attestations. They
can be verified with the GitHub CLI:

```bash
gh attestation verify ./clip_linux_amd64 --repo upendra7470/clip
```

Reports about bypassing these limits are especially welcome.
