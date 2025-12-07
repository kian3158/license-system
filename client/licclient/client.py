import requests
import time
import uuid
import json
import base64
from nacl.signing import SigningKey
import os
import sys
import hashlib
from pathlib import Path

HW_FILE = os.path.abspath(
    os.path.join(os.path.dirname(__file__), "../../hw-emulator/hw.json")
)
BASE = "http://localhost:8080"

# ---------------------------
# INTEGRITY CHECK (ADDED)
# ---------------------------


def get_self_path():
    exe = Path(sys.executable)
    if exe.exists():
        return exe
    return Path(__file__).resolve()


def sha256_of_file(p: Path):
    h = hashlib.sha256()
    with p.open("rb") as f:
        for chunk in iter(lambda: f.read(65536), b""):
            h.update(chunk)
    return h.hexdigest()


def load_expected_hash(path_hint: Path):
    base = path_hint.parent
    cand = base / "integrity.json"
    if cand.exists():
        try:
            j = json.loads(cand.read_text(encoding="utf-8"))
            return j.get("sha256")
        except Exception:
            return None

    # dev fallback (repo root-ish)
    cand2 = Path(__file__).resolve().parents[2] / "integrity.json"
    if cand2.exists():
        try:
            j = json.loads(cand2.read_text(encoding="utf-8"))
            return j.get("sha256")
        except Exception:
            return None

    return None


def check_integrity_or_exit():
    if os.getenv("SKIP_INTEGRITY") == "1":
        print("WARNING: Integrity check skipped via SKIP_INTEGRITY")
        return

    self_path = get_self_path()
    expected = load_expected_hash(self_path)

    if not expected:
        print("WARNING: No integrity metadata found, continuing (dev mode)")
        return

    actual = sha256_of_file(self_path)

    if actual != expected:
        print("INTEGRITY CHECK FAILED: binary has been modified")
        sys.exit(10)


# ---------------------------
# ORIGINAL CLIENT CODE
# ---------------------------


class LicenseClient:
    def __init__(self, client_id=None):
        self.client_id = client_id or str(uuid.uuid4())
        self.license_id = None
        self.sk = self._load_hw_key()
        self.last_total = 0

    def _load_hw_key(self):
        with open(HW_FILE, "r") as f:
            j = json.load(f)
        sk_b = base64.b64decode(j["private_key_base64"])
        return SigningKey(sk_b)

    def _sign_payload(self, payload):
        canonical = json.dumps(
            payload, sort_keys=True, separators=(",", ":"), ensure_ascii=False
        ).encode()
        sig = self.sk.sign(canonical).signature
        return base64.b64encode(sig).decode()

    def ping(self):
        print("PING:", requests.get(f"{BASE}/ping").text)

    def heartbeat(self):
        print("HEARTBEAT:", requests.get(f"{BASE}/heartbeat").json())

    def register(self):
        pub_key_b64 = base64.b64encode(self.sk.verify_key.encode()).decode()
        payload = {
            "client_id": self.client_id,
            "pub_key": pub_key_b64,
            "fingerprint": {"machine_id": "dev-machine-1"},
            "app_id": "api-cybernetics",
            "version": "0.1",
        }
        data = requests.post(f"{BASE}/register", json=payload).json()
        print("REGISTER RESPONSE:", data)
        self.license_id = data.get("license", {}).get("license_id", "LIC-DEV")
        return data

    def report(self, total_usage):
        if not self.license_id:
            raise ValueError("Client not registered")
        usage_delta = total_usage - self.last_total
        self.last_total = total_usage

        payload = {
            "client_id": self.client_id,
            "license_id": self.license_id,
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "usage_bytes_since_last": usage_delta,
            "total_usage_bytes": total_usage,
        }
        payload["signature"] = self._sign_payload(payload)
        resp = requests.post(f"{BASE}/report", json=payload).json()
        status = "ALLOWED" if resp.get("allowed") else "DENIED"
        print(
            f"REPORT: {status}, Remaining: {resp.get('remaining_bytes')} bytes, Action: {resp.get('action')}"
        )
        if resp.get("action") == "disable":
            print("License disabled by server!")


if __name__ == "__main__":
    # 🔒 Integrity check happens BEFORE anything else
    check_integrity_or_exit()

    client = LicenseClient()
    client.ping()
    client.heartbeat()
    client.register()
    client.report(10 * 1024 * 1024)
    client.report(110 * 1024 * 1024)
