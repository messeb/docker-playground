import urllib.request
import json
import sys

BASE = "http://localhost:8080/api"


def fetch(path):
    try:
        with urllib.request.urlopen(BASE + path, timeout=3) as r:
            return json.load(r)
    except Exception:
        print("Traefik is not running or dashboard is unavailable.")
        sys.exit(1)


overview    = fetch("/overview")
routers     = [r for r in fetch("/http/routers")    if r.get("provider") != "internal"]
services    = [s for s in fetch("/http/services")   if s.get("provider") != "internal"]
middlewares = [m for m in fetch("/http/middlewares") if m.get("provider") != "internal"]

hr = overview.get("http", {})

print(f"\n  Dashboard   http://localhost:8080/dashboard/")
print(f"  API         http://localhost:8080/api/http/routers")
print(f"\n  {hr.get('routers',{}).get('total',0)} routers   "
      f"{hr.get('services',{}).get('total',0)} services   "
      f"{hr.get('middlewares',{}).get('total',0)} middlewares")

print("\nRouters")
for r in sorted(routers, key=lambda x: x["name"]):
    ok  = "✓" if r.get("status") == "enabled" else "✗"
    eps = ", ".join(r.get("using", []))
    print(f"  {ok}  {r['name']:<22}  {r.get('rule',''):<38}  [{eps}]")

print("\nServices")
for s in sorted(services, key=lambda x: x["name"]):
    servers = s.get("loadBalancer", {}).get("servers", [])
    print(f"  {s['name']}")
    for srv in servers:
        status = srv.get("status")
        if status == "UP":
            indicator = "✓"
        elif status == "DOWN":
            indicator = "✗"
        else:
            indicator = "·"  # no health check configured
        label = status if status else "no health check"
        print(f"    {indicator}  {srv.get('url','?')}  [{label}]")

print("\nMiddlewares")
for m in sorted(middlewares, key=lambda x: x["name"]):
    mtype = m.get("type", "?")
    print(f"  {m['name']:<30}  {mtype}")
