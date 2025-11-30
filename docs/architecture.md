# Architecture (short)

- License Manager (Go): central authority, issues signed license blobs, validates usage reports, enforces quotas.
- License Client (Python): runs in each app container, registers, posts hourly usage reports, and enforces disable when server responds.
- HW Root-of-Trust: USB HSM (YubiKey/Nitrokey) in production; SoftHSM or hw-emulator in dev for signing.
- Transport: HTTPS (TLS1.3). Optionally mTLS + signed payloads (ed25519).
- Persistence: /var/lib/licman for DB and signed daily summaries.
