import json
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from dapr.clients import DaprClient

app = FastAPI(title="publisher")

PUBSUB_NAME = "pubsub"
TOPIC = "orders"


class Payload(BaseModel):
    message: str


@app.post("/publish")
def publish(payload: Payload) -> dict:
    try:
        with DaprClient() as client:
            client.publish_event(
                pubsub_name=PUBSUB_NAME,
                topic_name=TOPIC,
                data=json.dumps(payload.model_dump()),
                data_content_type="application/json",
            )
        return {"published": payload.message, "topic": TOPIC}
    except Exception as exc:
        raise HTTPException(status_code=502, detail=f"publish failed: {exc}") from exc


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}
