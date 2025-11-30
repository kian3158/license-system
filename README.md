# License System

## Project Overview
This project implements a licensing system for Docker-based applications. It includes a Go-based License Manager and a Python-based License Client along with a Hardware Emulator (HW Emulator) for development purposes.

The system allows:
- Registration of clients
- Enforcement of usage quotas
- Periodic reporting of usage
- Integration with a hardware lock for license validation

## Current Status - Milestone 1
- HW Emulator:
  - Generates a hardware lock JSON file
  - Serves `/info` and `/sign` endpoints for license verification
- License Manager (Go):
  - Skeleton for `/ping`, `/heartbeat`, `/register`, `/report` endpoints
  - Stores client info in memory
  - Tracks usage and quota enforcement
- Basic registration and reporting flow tested locally
- Ready for next milestone: integrating client authentication, signing, and persistent storage

## Prerequisites
- Python 3.10+ (for HW Emulator and License Client)
- Go 1.20+ (for License Manager)
- Optional: Docker (for running the apps in containers later)

## Run Instructions

### HW Emulator
```bash
cd hw-emulator
python generate_hw_json.py
uvicorn main:app --reload --port 8081
```

### License Manager
```bash
cd manager/cmd/licmgr
go run main.go
```

### Testing Endpoints
Use curl or Postman to test:
- `GET /ping` — checks if the License Manager is running
- `GET /heartbeat` — returns server status
- `POST /register` — registers a new client
- `POST /report` — reports usage for a client
- `GET /info` — checks if HW Emulator is present
- `POST /sign` — signs a payload using the HW Emulator

## Notes
- All current storage is in memory (for development). Persistent storage will be added in future milestones.
- HW Emulator private keys should be kept secure.
- Pre-commit hooks for code formatting and linting are recommended before committing changes.
