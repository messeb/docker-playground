import logging
from fastapi import FastAPI, Request

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("subscriber")

app = FastAPI(title="subscriber")

PUBSUB_NAME = "pubsub"
TOPIC = "orders"
ROUTE = "/orders"


@app.get("/dapr/subscribe")
def subscribe() -> list[dict]:
    return [
        {
            "pubsubname": PUBSUB_NAME,
            "topic": TOPIC,
            "route": ROUTE,
        }
    ]


@app.post(ROUTE)
async def handle(request: Request) -> dict:
    envelope = await request.json()
    data = envelope.get("data")
    log.info("received topic=%s data=%s", envelope.get("topic"), data)
    return {"status": "SUCCESS"}


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}
