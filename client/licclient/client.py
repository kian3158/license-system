# client/licclient/client.py
import requests
import json
import time
import uuid

BASE = "http://localhost:8080"


def ping():
    r = requests.get(f"{BASE}/ping")
    print("PING:", r.text)


def heartbeat():
    r = requests.get(f"{BASE}/heartbeat")
    print("HEARTBEAT:", r.json())


def register(client_id=None):
    if client_id is None:
        client_id = str(uuid.uuid4())
    payload = {
        "client_id": client_id,
        "pub_key": "dev-pubkey-placeholder",
        "fingerprint": {"machine_id": "dev-machine-1"},
        "app_id": "api-cybernetics",
        "version": "0.1",
    }
    r = requests.post(f"{BASE}/register", json=payload)
    print("REGISTER RESPONSE:", r.status_code, r.json())
    return client_id, r.json()


def report(client_id, license_id, total_usage):
    payload = {
        "client_id": client_id,
        "license_id": license_id,
        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "usage_bytes_since_last": 1024 * 1024,
        "total_usage_bytes": total_usage,
        "signature": "dev-client-sig",
    }
    r = requests.post(f"{BASE}/report", json=payload)
    print("REPORT RESPONSE:", r.status_code, r.json())


if __name__ == "__main__":
    ping()
    heartbeat()
    client_id, reg = register()
    license = reg.get("license", {})
    licid = license.get("license_id", "LIC-DEV")
    report(client_id, licid, 10 * 1024 * 1024)  # 10 MB
    report(
        client_id, licid, 110 * 1024 * 1024
    )  # 110 MB -> should exceed 100MB dev quota
