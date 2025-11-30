import requests
import time
import uuid

BASE = "http://localhost:8080"


class LicenseClient:
    def __init__(self, client_id=None):
        if client_id is None:
            client_id = str(uuid.uuid4())
        self.client_id = client_id
        self.license_id = None

    def ping(self):
        r = requests.get(f"{BASE}/ping")
        print("PING:", r.text)

    def heartbeat(self):
        r = requests.get(f"{BASE}/heartbeat")
        print("HEARTBEAT:", r.json())

    def register(self):
        payload = {
            "client_id": self.client_id,
            "pub_key": "dev-pubkey-placeholder",
            "fingerprint": {"machine_id": "dev-machine-1"},
            "app_id": "api-cybernetics",
            "version": "0.1",
        }
        r = requests.post(f"{BASE}/register", json=payload)
        print("REGISTER RESPONSE:", r.status_code, r.json())
        self.license_id = r.json().get("license", {}).get("license_id", "LIC-DEV")
        return r.json()

    def report(self, total_usage):
        if not self.license_id:
            raise ValueError("Client not registered yet")
        payload = {
            "client_id": self.client_id,
            "license_id": self.license_id,
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "usage_bytes_since_last": 1024 * 1024,
            "total_usage_bytes": total_usage,
            "signature": "dev-client-sig",
        }
        r = requests.post(f"{BASE}/report", json=payload)
        print("REPORT RESPONSE:", r.status_code, r.json())
