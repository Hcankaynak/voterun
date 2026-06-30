# Deploying VoteRun

This folder holds the **server-side** setup: a host nginx reverse proxy that maps
your domains to the running containers.

```
                              ┌─────────────────────────────────────┐
  new.voterun.app    ──▶      │  host nginx  ──▶  frontend :3000     │
                              │                     └─(internal)─▶ backend  │
  api.voterun.app    ──▶      │  host nginx  ──▶  backend  :8081     │
  grafana.voterun.app ──▶     │  host nginx  ──▶  grafana  :3001     │
                              └─────────────────────────────────────┘
```

- `new.voterun.app` → the frontend container (which proxies `/api` and `/ws`
  to the backend internally).
- `api.voterun.app` → the backend container directly (for direct API access).
- `grafana.voterun.app` → the Grafana monitoring UI (see [Monitoring](#monitoring)).

Container ports are bound to `127.0.0.1` only, so the apps are reachable
**exclusively** through the host nginx proxy.

## Prerequisites

- A server with **Docker** + the **docker compose** plugin.
- **nginx** and **certbot** installed on the host (`apt install nginx certbot python3-certbot-nginx`).
- DNS **A/AAAA records** for `new.voterun.app`, `api.voterun.app`, and
  `grafana.voterun.app` pointing at the server's IP.

## 1. Start the containers

Copy this `deploy/` folder to the server, create your `.env`, then bring the
stack up from inside it:

```bash
cd deploy
cp .env.example .env        # then edit the values

# Only needed if the GHCR packages are private:
echo "$GHCR_TOKEN" | docker login ghcr.io -u Hcankaynak --password-stdin

docker compose pull
docker compose up -d
```

Verify they're up and listening on localhost:

```bash
curl -s http://127.0.0.1:8081/api/health   # {"status":"ok"}
curl -sI http://127.0.0.1:3000             # 200 from the frontend
```

## 2. Configure the nginx reverse proxy

Copy the vhost configs and reload nginx:

```bash
sudo cp nginx/new.voterun.app.conf /etc/nginx/conf.d/
sudo cp nginx/api.voterun.app.conf /etc/nginx/conf.d/
sudo cp nginx/grafana.voterun.app.conf /etc/nginx/conf.d/
sudo nginx -t && sudo systemctl reload nginx
```

## 3. Enable HTTPS (Let's Encrypt)

certbot will obtain certificates and rewrite the vhosts to add port 443 + an
HTTP→HTTPS redirect automatically:

```bash
sudo certbot --nginx -d new.voterun.app -d api.voterun.app -d grafana.voterun.app
```

Certificates auto-renew via certbot's systemd timer.

## Updating to a new build

```bash
cd deploy
docker compose pull
docker compose up -d
```

Data persists in the `voterun-data` Docker volume across updates. Pin a specific
build by setting `IMAGE_TAG` in `.env` (e.g. `IMAGE_TAG=sha-1a2b3c4`).

## Monitoring

The stack includes **Prometheus** (metrics scraper) and **Grafana**
(dashboards). Both come up with `docker compose up -d` alongside the app.

- The backend exposes Prometheus metrics at `/metrics`. This endpoint is
  **internal only**: Prometheus scrapes it over the Compose network
  (`backend:8080`), and the `api.voterun.app` vhost returns `404` for `/metrics`
  so it is never reachable publicly.
- Prometheus and Grafana bind to `127.0.0.1` (ports `9090` and `3001`). Grafana
  is exposed publicly only through the `grafana.voterun.app` host nginx vhost.

Setup:

1. Set a strong `GRAFANA_ADMIN_PASSWORD` in `.env`.
2. Add the DNS record and vhost for `grafana.voterun.app` (steps 2-3 above).
3. Open `https://grafana.voterun.app` and log in as `admin` with that password.

A **Prometheus** datasource and a **VoteRun Overview** dashboard are
auto-provisioned (request rate by route, p50/p95/p99 latency, error rate,
in-flight requests, active WebSocket connections, broadcast rate, and Go runtime
stats). Provisioning lives in `monitoring/` and Prometheus retains 30 days of
data in the `prometheus-data` volume.

Verify the scrape target is healthy (optionally, via SSH tunnel to `:9090`):

```bash
curl -s http://127.0.0.1:9090/api/v1/targets | grep voterun-backend
```

## Notes

- **CORS:** the frontend calls `/api` and `/ws` on its own origin (proxied
  internally), so cross-origin requests don't normally occur. `CORS_ORIGIN`
  (in `.env`) only matters if the browser calls `api.voterun.app` directly.
- **Single instance:** the backend uses SQLite and an in-memory WebSocket hub,
  so run a single backend instance. To scale horizontally you'd move to Postgres
  and a shared pub/sub (e.g. Redis) for the realtime layer.
- **Containerized proxy alternative:** instead of host nginx you could run the
  reverse proxy (nginx/Caddy/Traefik) as another compose service on the same
  network, proxying to `frontend:80` / `backend:8080` by service name.
