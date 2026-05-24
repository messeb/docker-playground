from fastapi import FastAPI

app = FastAPI(title="service-b")


@app.get("/hello/{name}")
def hello(name: str) -> dict[str, str]:
    return {"msg": f"hello, {name}", "from": "service-b"}


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}
