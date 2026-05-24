from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from dapr.clients import DaprClient

app = FastAPI(title="writer")

STORE = "statestore"


class Item(BaseModel):
    key: str
    value: str


@app.post("/save")
def save(item: Item) -> dict:
    try:
        with DaprClient() as client:
            client.save_state(STORE, item.key, item.value)
        return {"saved": item.key}
    except Exception as exc:
        raise HTTPException(status_code=502, detail=f"save failed: {exc}") from exc


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}
