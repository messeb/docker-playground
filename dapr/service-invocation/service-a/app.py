from fastapi import FastAPI, HTTPException
from dapr.clients import DaprClient

app = FastAPI(title="service-a")

TARGET_APP_ID = "service-b"


@app.get("/call/{name}")
def call(name: str) -> dict:
    try:
        with DaprClient() as client:
            resp = client.invoke_method(
                app_id=TARGET_APP_ID,
                method_name=f"hello/{name}",
                http_verb="GET",
            )
            return {"from": "service-a", "callee": resp.json()}
    except Exception as exc:
        raise HTTPException(status_code=502, detail=f"invoke failed: {exc}") from exc


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}
