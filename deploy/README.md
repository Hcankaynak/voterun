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

## After a successful build

Every push to `main` runs the **Build and Publish Docker Images** GitHub Action
(`.github/workflows/docker-publish.yml`), which pushes fresh `voterun-backend`
and `voterun-frontend` images to GHCR (tagged `latest` and `sha-<commit>`). Once
that workflow is green, follow one of the two scenarios below.

### A. Normal deploy (new build of the existing services)

Use this whenever you just want the server to run the latest code.

1. **Confirm the build is green.** Check that the "Build and Publish Docker
  Images" GitHub Action finished successfully so the images exist on GHCR.
2. **SSH to the server** and move into the deploy folder:
  ```bash
   cd deploy
  ```
3. **(Optional) pin a specific build.** By default `.env` uses `IMAGE_TAG=latest`
  (newest build from `main`). To deploy an exact commit, set it to the short
   SHA instead:
4. **Pull the new images:**
  ```bash
   docker compose pull
  ```
5. **Recreate the containers** (Compose only restarts the ones whose image
  changed):
  ```bash
   docker compose up -d
  ```
6. **Verify the stack is healthy:**
  ```bash
   docker compose ps
   curl -s  http://127.0.0.1:8081/api/health   # {"status":"ok"}
   curl -sI http://127.0.0.1:3000              # 200 from the frontend
  ```

Data persists across updates in the `postgres-data`, `prometheus-data`, and
`grafana-data` Docker volumes. To **roll back**, set `IMAGE_TAG` to a previous
`sha-<commit>` in `.env` and repeat steps 4-5.

### B. Adding a new subdomain

Use this when you expose a new service (or an existing one) under a new host,
e.g. `foo.voterun.app`.

1. **Add a DNS record.** Point an A/AAAA record for the new subdomain at the
  server's IP (same as the existing subdomains).
2. **Make sure a local service answers for it.** The subdomain must map to a
  container port bound to `127.0.0.1`. Either reuse an existing one, or add a
   new service to `[docker-compose.yml](docker-compose.yml)` with a
   `127.0.0.1:<port>:<container-port>` binding, then `docker compose up -d`.
3. **Create the nginx vhost.** Copy an existing config in `[nginx/](nginx)` as a
  starting point (e.g. `new.voterun.app.conf`) and edit the `server_name` and
   the `upstream` port to match the new subdomain. Keep the `/ws/` block only if
   the service needs WebSockets:
4. **Install and reload nginx:**
  ```bash
   sudo cp nginx/foo.voterun.app.conf /etc/nginx/conf.d/
   sudo nginx -t && sudo systemctl reload nginx
  ```
5. **Enable HTTPS** (adds the 443 server block + HTTP→HTTPS redirect):
  ```bash
   sudo certbot --nginx -d foo.voterun.app
  ```
6. **Verify** the subdomain responds over HTTPS:
  ```bash
   curl -sI https://foo.voterun.app
  ```

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

