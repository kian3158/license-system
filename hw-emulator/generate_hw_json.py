# hw-emulator/generate_hw_json.py
import base64, json, os
from nacl.signing import SigningKey


def generate(path="hw.json", device_id="HW-DEV-001"):
    sk = SigningKey.generate()
    private_key_b64 = base64.b64encode(sk.encode()).decode()
    payload = {"device_id": device_id, "private_key_base64": private_key_b64}
    with open(path, "w") as f:
        json.dump(payload, f, indent=2)
    try:
        os.chmod(path, 0o600)
    except Exception:
        pass
    print(f"Generated {path} (keep this file private).")


if __name__ == "__main__":
    generate()
