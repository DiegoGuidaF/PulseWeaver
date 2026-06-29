# Security Policy

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

If you discover a security issue, use [GitHub's private vulnerability reporting](https://github.com/DiegoGuidaF/PulseWeaver/security/advisories/new) to disclose it responsibly. This keeps the details private until a fix is available.

Please include:
- A description of the vulnerability and its potential impact
- Steps to reproduce the issue
- Any relevant configuration (redact secrets)
- Suggested fix if you have one

You can expect an acknowledgement within a few days and a resolution or update within a reasonable timeframe depending on severity.

## Scope

This policy covers the PulseWeaver server (`app/`). For the heartbeat clients (PulseWeaver Companion, Docker, curl), please report via the [heartbeat client repository](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client).

## Security Model

PulseWeaver is an **IP gate**, not an authentication system. Before reporting, please read the [Security Model documentation](docs/Security-Model.md) — some limitations are known and by design.
