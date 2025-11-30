# hw-emulator/main.py
import os
import json
import base64
from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import JSONResponse
from nacl.signing import SigningKey

# Use local ./hw.json for dev (we'll mount /data in docker later)
HW_FILE = os.path.join(os.getcwd(), "hw.json")
app = FastAPI(title="HW Emulator (dev)")


def _load_signing_key() -> SigningKey:
    if not os.path.exists(HW_FILE):
        raise FileNotFoundError("hw_lock_not_found")
    with open(HW_FILE, "r") as f:
        j = json.load(f)
    if "private_key_base64" not in j:
        raise ValueError("invalid_hw_file")
    sk_b = base64.b64decode(j["private_key_base64"])
    return SigningKey(sk_b)


@app.get("/info")
async def info():
    present = os.path.exists(HW_FILE)
    if not present:
        return JSONResponse(status_code=200, content={"present": False})
    try:
        with open(HW_FILE, "r") as f:
            j = json.load(f)
        dev_id = j.get("device_id", None)
    except Exception:
        dev_id = None
    return {"present": True, "device_id": dev_id}


@app.post("/sign")
async def sign(request: Request):
    try:
        sk = _load_signing_key()
    except FileNotFoundError:
        raise HTTPException(status_code=400, detail="hw_lock_not_found")
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

    try:
        body = await request.json()
    except Exception:
        raise HTTPException(status_code=400, detail="invalid_json_body")

    canonical = json.dumps(
        body, sort_keys=True, separators=(",", ":"), ensure_ascii=False
    ).encode("utf-8")
    sig = sk.sign(canonical).signature
    sig_b64 = base64.b64encode(sig).decode("utf-8")
    return {"signature": sig_b64}
