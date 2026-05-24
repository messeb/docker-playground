from fastapi import FastAPI, HTTPException
from dapr.clients import DaprClient

app = FastAPI(title="reader")

STORE = "statestore"


@app.get("/load/{key}")
def load(key: str) -> dict:
    try:
        with DaprClient() as client:
            resp = client.get_state(STORE, key)
        return {"key": key, "value": resp.text() or None}
    except Exception as exc:
        raise HTTPException(status_code=502, detail=f"load failed: {exc}") from exc


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}
