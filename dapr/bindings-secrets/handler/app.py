from fastapi import FastAPI, HTTPException
from dapr.clients import DaprClient

app = FastAPI(title="handler")

SECRET_STORE = "secretstore"


@app.get("/info")
def info() -> dict:
    try:
        with DaprClient() as client:
            api_key = client.get_secret(SECRET_STORE, "api-key").secret["api-key"]
            db_pw = client.get_secret(SECRET_STORE, "db-password").secret["db-password"]
        return {
            "api_key": api_key,
            "db_password_len": len(db_pw),
            "source": f"dapr secret store '{SECRET_STORE}'",
        }
    except Exception as exc:
        raise HTTPException(status_code=502, detail=f"secret fetch failed: {exc}") from exc


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}
