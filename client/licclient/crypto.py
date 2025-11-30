import base64
from nacl.signing import SigningKey


def load_signing_key(path="hw.json"):
    import json
    import os

    if not os.path.exists(path):
        raise FileNotFoundError("hw_lock_not_found")
    with open(path, "r") as f:
        j = json.load(f)
    sk_b = base64.b64decode(j["private_key_base64"])
    return SigningKey(sk_b)


def get_public_key_b64(sk):
    return base64.b64encode(sk.verify_key.encode()).decode()
