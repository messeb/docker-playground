import logging
from datetime import datetime, timezone
from fastapi import FastAPI

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("scheduler")

app = FastAPI(title="scheduler")
state = {"ticks": 0, "last": None}


@app.post("/tick")
def tick() -> dict:
    state["ticks"] += 1
    state["last"] = datetime.now(timezone.utc).isoformat()
    log.info("cron tick #%d at %s", state["ticks"], state["last"])
    return {"received": True}


@app.get("/status")
def status() -> dict:
    return state


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}
