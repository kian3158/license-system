import requests
import time
import uuid
import json
import base64
from nacl.signing import SigningKey
import os

# automatically find hw.json relative to this file
HW_FILE = os.path.join(os.path.dirname(__file__), "../../hw-emulator/hw.json")
HW_FILE = os.path.abspath(HW_FILE)  # get absolute path

BASE = "http://localhost:8080"


class LicenseClient:
    def __init__(self, client_id=None):
        if client_id is None:
            client_id = str(uuid.uuid4())
        self.client_id = client_id
        self.license_id = None
        self.sk = self._load_hw_key()
        self.last_total = 0

    def _load_hw_key(self):
        """Load signing key from hw.json"""
        with open(HW_FILE, "r") as f:
            j = json.load(f)
        sk_b = base64.b64decode(j["private_key_base64"])
        return SigningKey(sk_b)

    def _sign_payload(self, payload):
        """Sign payload with HW private key"""
        canonical = json.dumps(
            payload, sort_keys=True, separators=(",", ":"), ensure_ascii=False
        ).encode("utf-8")
        sig = self.sk.sign(canonical).signature
        return base64.b64encode(sig).decode("utf-8")

    def ping(self):
        r = requests.get(f"{BASE}/ping")
        print("PING:", r.text)

    def heartbeat(self):
        r = requests.get(f"{BASE}/heartbeat")
        print("HEARTBEAT:", r.json())

    def register(self):
        pub_key_b64 = base64.b64encode(self.sk.verify_key.encode()).decode("utf-8")
        payload = {
            "client_id": self.client_id,
            "pub_key": pub_key_b64,  # send actual public key
            "fingerprint": {"machine_id": "dev-machine-1"},
            "app_id": "api-cybernetics",
            "version": "0.1",
        }
        r = requests.post(f"{BASE}/register", json=payload)
        data = r.json()
        print("REGISTER RESPONSE:", r.status_code, data)
        self.license_id = data.get("license", {}).get("license_id", "LIC-DEV")
        return data

    def report(self, total_usage):
        if not self.license_id:
            raise ValueError("Client not registered yet")
        usage_since_last = total_usage - self.last_total
        self.last_total = total_usage

        payload = {
            "client_id": self.client_id,
            "license_id": self.license_id,
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "usage_bytes_since_last": usage_since_last,
            "total_usage_bytes": total_usage,
        }
        payload["signature"] = self._sign_payload(payload)
        r = requests.post(f"{BASE}/report", json=payload)
        resp = r.json()
        status = "ALLOWED" if resp.get("allowed") else "DENIED"
        print(
            f"REPORT: {status}, Remaining: {resp.get('remaining_bytes')} bytes, Action: {resp.get('action')}"
        )


if __name__ == "__main__":
    client = LicenseClient()
    client.ping()
    client.heartbeat()
    client.register()
    client.report(10 * 1024 * 1024)  # 10 MB
    client.report(110 * 1024 * 1024)  # 110 MB -> should exceed dev quota
