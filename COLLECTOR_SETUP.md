# Collector integration guide for Facile projects

This file is a **copy‑paste template** you can drop into any Facile repo (Nuage, Opus, Capsule, etc.) to wire the Journal collector side‑car and Casier‑based configuration.

## 1. Add the collector service
Create `docker-compose.collector.yml` (or paste the block into your existing `docker-compose.yml`).
```yaml
version: "3.9"

services:
  journal-collector:
    image: ghcr.io/facilestudio/journal/collector:latest
    restart: unless-stopped
    profiles: [collector]
    networks:
      - default
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      JOURNAL_URL: http://journal-api:4010/api   # adjust if API service name differs
      JOURNAL_TOKEN: ${INGEST_TOKEN:-}
    labels:
      journal.ignore: "true"
    depends_on:
      journal-api:
        condition: service_healthy
```
*If your API container is named something other than `journal-api`, change the `JOURNAL_URL` accordingly.*

## 2. Enable the profile when you start the stack
```bash
COMPOSE_PROFILES=collector \
  docker compose -f docker-compose.yml \
                 -f docker-compose.dev.yml \
                 -f docker-compose.collector.yml up -d
```
In production add the `profiles: [collector]` line (as above) to the production compose file so the collector runs automatically.

## 3. Store the ingest token in Casier
```bash
# one‑time secret creation per project / environment
casier secrets set -p <project> -e development INGEST_TOKEN "$(openssl rand -hex 32)"
casier secrets set -p <project> -e production INGEST_TOKEN "$(openssl rand -hex 32)"
```
The `${INGEST_TOKEN}` placeholder will be injected at runtime by `casier run` tasks (see step 4).

## 4. Switch the API launch to Casier
If the repo does not already have a `mise.toml`, copy the one from the Journal repo (it contains the `dev`, `dev‑offline`, and `secrets` tasks). Then run the API with:
```bash
mise dev          # pulls secrets from Casier (network‑first)
# or, when Casier is unreachable:
mise dev-offline
```
Remove any manual `source .env` or `cp .env.example .env` steps from your docs/CI.

## 5. Wire the browser SDK (optional but recommended)
1. **Create a public API key** in the Journal UI → Settings → API → Add key → *Public*.
2. Store the key in Casier:
   ```bash
   casier secrets set -p <project> -e production JOURNAL_PUBLIC_KEY <public‑key>
   ```
3. Expose it to the front‑end (Vite env variable):
   ```ts
   const PUBLIC_KEY = import.meta.env.VITE_JOURNAL_PUBLIC_KEY;
   export const journalClient = new JournalClient({
     baseURL: import.meta.env.VITE_JOURNAL_URL,
     publicKey: PUBLIC_KEY,
   });
   ```
4. Add the origin to Journal's CORS allowlist via Casier:
   ```bash
   casier secrets set -p journal -e production CORS_ALLOWED_ORIGINS "https://<your‑app>.facile.studio"
   ```

## 6. Verify
```bash
# produce a test log line from any container
docker exec -t <some_container> bash -c 'echo "collector test from $(hostname)"'
# then check the Journal UI (http://localhost:5173) or call the API directly:
curl http://localhost:4010/api/logs | jq '.entries[] | .message'
```
You should see the line appear under the appropriate `app` name.

---

**That’s it** – with these three files (`docker-compose.collector.yml`, the `mise.toml` copy, and this README) every Facile project gets:
* Casier as the single source of truth for secrets and env vars,
* Automatic container‑log collection via the published collector image,
* Optional browser‑side logs via the public key.

Apply the steps, commit the new files, and push. All services will now feed into the central Journal dashboard.
