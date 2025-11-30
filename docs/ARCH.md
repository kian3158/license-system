# Architecture (high level)

- License Manager (Go): central authority, signs licenses, validates reports.
- License Client (Python): runs inside each dockerized app, registers, reports hourly.
- HW root-of-trust: USB HSM (prod) / SoftHSM or hw-emulator (dev) used for signing.
- Transport: HTTPS (TLS1.3). Optionally mTLS + signed payloads (ed25519).
- Storage: /var/lib/licman for DB + signed daily summaries.

See docs/schemas for register/report JSON schemas.
