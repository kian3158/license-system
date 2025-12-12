# License Manager System

This project implements the early version of a licensing and usage-limiting system for a distributed set of Dockerized applications.
A central **License Manager** (Go) enforces usage limits and authentication, and each app runs a **Python License Client** that registers itself, reports usage, and receives allow/deny responses.

The design goal is to prevent unauthorized copies of the applications and to enforce resource-based limitations.



## Components

### **License Manager (Go)**
- HTTP-based API.
- Validates client registration.
- Verifies Ed25519 signatures on all client payloads.
- Tracks usage totals and enforces limits.
- Uses a **hardware-lock emulator** to simulate a USB dongle (private key + metadata).
- Responds with JSON containing:
  - allowed / denied
  - remaining bytes
  - action (e.g., `"disable"`)

### **Hardware Lock Emulator**
- Located in `hw-emulator/hw.json`.
- Simulates:
  - Dongle private key
  - License configuration (limits, expiry, etc.)
- If deleted or unreadable, components treat it as “hardware missing.”

### **License Client (Python)**
Used by each Docker application. Current capabilities:

- Generates a unique client ID.
- Loads private key from the hardware emulator.
- Signs registration + usage reports (Ed25519).
- Performs registration handshake.
- Sends heartbeats.
- Sends periodic usage reports with delta + total.
- Processes allow/deny responses from manager.
- Includes **binary integrity check**:
  - Computes SHA256 of its own binary/script.
  - Compares against `integrity.json`.
  - Exits if modified.
  - Can be bypassed using `SKIP_INTEGRITY=1` (dev only).

- Prepared for future compilation via Nuitka to produce a standalone binary.


## Current Features

- Client → Manager registration flow works.
- Usage reporting + enforcement works.
- Signature verification is implemented.
- HW-lock emulation layer is fully functional.
- Anti-tamper integrity check added to the client.
- Manager processes signed data and applies limits.
- Test runs show allow → deny transitions when exceeding configured limit.



## How to Run (Test Build)

### Start Manager
```bash
cd manager
go run main.go
```

### Run Client
```bash
cd client
python3 client.py
```

### Expected Behavior
- Client pings/heartbeats.
- Registration completes.
- Two usage reports are sent.
- Manager responds ALLOWED → DENIED once limit is exceeded.


## Structure
```
/client               # Python license client + integrity check
/manager              # Go license manager API
/hw-emulator          # Simulated hardware lock (private key & config)
integrity.json        # Expected SHA-256 for binary/script
README.md
```
