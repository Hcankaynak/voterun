# Deploying VoteRun

This folder holds the **server-side** setup: a host nginx reverse proxy that maps
your domains to the running containers.

```
                          ┌─────────────────────────────────────┐
  new.voterun.app  ──▶    │  host nginx  ──▶  frontend :3000     │
                          │                     └─(internal)─▶ backend  │
  api.voterun.app  ──▶    │  host nginx  ──▶  backend  :8081     │
                          └─────────────────────────────────────┘
```

- `new.voterun.app` → the frontend container (which proxies `/api` and `/ws`
  to the backend internally).
- `api.voterun.app` → the backend container directly (for direct API access).

Container ports are bound to `127.0.0.1` only, so the apps are reachable
**exclusively** through the host nginx proxy.

## Prerequisites

- A server with **Docker** + the **docker compose** plugin.
- **nginx** and **certbot** installed on the host (`apt install nginx certbot python3-certbot-nginx`).
- DNS **A/AAAA records** for `new.voterun.app` and `api.voterun.app` pointing at the server's IP.

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
sudo nginx -t && sudo systemctl reload nginx
```

## 3. Enable HTTPS (Let's Encrypt)

certbot will obtain certificates and rewrite the vhosts to add port 443 + an
HTTP→HTTPS redirect automatically:

```bash
sudo certbot --nginx -d new.voterun.app -d api.voterun.app
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
