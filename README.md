# Cloud Cost Dashboard

A lightweight, self-hosted cost dashboard that pulls spend directly from AWS,
Azure, GCP, and OCI, plus manual/CSV import for Contabo and any other
provider — organized into a hierarchy so managers can see cost at any level:
**Organization → Product → Project → Environment → Region → Cloud Account**.

## Stack

- **Backend:** Go (single binary), SQLite (pure-Go driver, no CGO), REST API
- **Frontend:** React + Vite + TypeScript, recharts for charts
- **Scheduler:** built-in cron loop, pulls fresh costs every `SYNC_INTERVAL_MINUTES`

## Running locally with Docker (recommended)

```sh
cp backend/.env.example backend/.env   # edit JWT_SECRET, ENCRYPTION_KEY, ADMIN_EMAIL/PASSWORD
docker compose up --build
```

- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- Log in with the `ADMIN_EMAIL` / `ADMIN_PASSWORD` you set (defaults to
  `admin@example.com` / `change-me` — **change this immediately** if you don't
  set your own).

## Running locally without Docker

**Backend** (requires Go 1.22+):
```sh
cd backend
go mod tidy
go run ./cmd/server
```

**Frontend** (requires Node 18+):
```sh
cd frontend
npm install
npm run dev
```
The dev server proxies to `http://localhost:8080` by default; override with a
`VITE_API_URL` env var if the backend runs elsewhere.

## Setting up the hierarchy

1. Log in as admin, go to **Admin**.
2. Create your Organization, then Products under it, Projects under those,
   Environments (prod/staging/dev) under those, and Regions under those.
3. Under **Cloud accounts**, attach a real AWS account / Azure subscription /
   GCP project / OCI tenancy / Contabo account / other to a Region.
4. For AWS/Azure/GCP/OCI, add a **credential** first (read-only!) and attach
   it to the cloud account — costs sync automatically on the schedule, or
   click **Sync now** to pull immediately.
5. For Contabo or any custom provider, use **Import CSV** on that account
   instead (header row: `date,service_name,amount,currency`).

## Provider credentials — required permissions (read-only)

| Provider | What you need | Required access |
|---|---|---|
| AWS | Access key + secret for an IAM user/role | `ce:GetCostAndUsage` only |
| Azure | Service principal (tenant/client id + secret) | `Cost Management Reader` on the subscription |
| GCP | Service account JSON key | `BigQuery Data Viewer` + `BigQuery Job User` on the project that owns your [Billing Export dataset](https://cloud.google.com/billing/docs/how-to/export-data-bigquery-setup) — GCP has no direct cost API, billing export to BigQuery must be enabled first |
| OCI | User OCID, fingerprint, private key (PEM), region | read access to usage-report resources in the tenancy |
| Contabo | none | no cost API exists — use CSV/manual import |
| Other/Generic | none | no adapter — use CSV/manual import (or add a new adapter, see below) |

Credentials are encrypted at rest (AES-256-GCM, keyed by `ENCRYPTION_KEY`) and
never returned by the API.

## Roles

- **Admin** — manages hierarchy, credentials, cloud accounts, users
- **Manager** — read-only, sees the whole organization
- **Viewer** — read-only, scoped to one Product or Project (assigned by an admin)

## Adding a new cloud provider

Implement `providers.CostProvider` (`backend/internal/providers/provider.go`)
— one method, `FetchCosts`, returning daily per-service cost records — and
register it in `backend/internal/providers/registry.go`. No other code needs
to change; the scheduler, API, and frontend already handle any registered
provider generically. Providers with no live cost API (like Contabo) skip
registration and rely on the CSV import endpoint instead.

## API overview

All routes except `/api/auth/login` require `Authorization: Bearer <token>`.

- `POST /api/auth/login` — get a JWT
- `GET /api/tree?start=&end=` — full hierarchy with rolled-up cost per node
- `GET /api/costs/timeseries?scopeType=&scopeId=&groupBy=day|service&start=&end=`
- `POST /api/organizations` / `products` / `projects` / `environments` / `regions` (admin)
- `POST /api/credentials`, `POST /api/cloud-accounts` (admin)
- `POST /api/cloud-accounts/{id}/sync-now`, `POST /api/cloud-accounts/{id}/import-csv` (admin)
- `POST /api/users` (admin) — create Manager/Viewer accounts
