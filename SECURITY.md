# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in goBili, please report it
**privately** rather than opening a public issue.

- **Email:** my@dengmengmian.com
- **PGP key:** not yet available — please request one if needed.

### What to include

- A clear description of the vulnerability.
- Steps to reproduce it.
- Affected versions.
- Any suggested mitigations.

### What to expect

1. You will receive an acknowledgment within 48 hours.
2. We will investigate and provide an initial assessment within 5 business days.
3. Once a fix is prepared, we will coordinate a disclosure timeline with you.
4. After the fix is released, you will be credited in the release notes
   (unless you prefer to remain anonymous).

## Scope

Security issues in the following areas are in scope:

- Authentication bypass or credential leakage.
- Arbitrary command execution.
- Path traversal leading to file writes outside the output directory.
- Network request forgery targeting Bilibili APIs.

**Out of scope:** issues that require the attacker to already have local
filesystem access or control of the user's machine.
