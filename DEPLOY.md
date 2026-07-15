# Deploying to an Ubuntu server (with Azure + GCP connected)

Assumes: SSH access to the Ubuntu server already works, and you have a
domain name (e.g. `cost.example.com`) with an **A record pointing at the
server's public IP**. Run "local machine" steps on your own computer;
everything else is run over SSH on the server.

---

## 1. Push the code to a git remote (local machine)

```sh
cd cloud-cost-dashboard
git add .
git commit -m "Initial commit"
git branch -M main
```

Create an empty repository on GitHub (or GitLab/your own git server) —
**do not** initialize it with a README — then:

```sh
git remote add origin git@github.com:<you>/cloud-cost-dashboard.git
git push -u origin main
```

## 2. Server prep

SSH in, then update the system and install Docker:

```sh
ssh youruser@your-server-ip

sudo apt update && sudo apt upgrade -y

# Docker Engine + Compose plugin (official Docker repo, not the older
# distro-packaged version)
sudo apt install -y ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo $VERSION_CODENAME) stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# run docker without sudo
sudo usermod -aG docker $USER
newgrp docker

# nginx (reverse proxy) + certbot (Let's Encrypt)
sudo apt install -y nginx certbot python3-certbot-nginx
```

### Firewall

```sh
sudo ufw allow OpenSSH
sudo ufw allow 80
sudo ufw allow 443
sudo ufw enable
sudo ufw status
```

**Important:** Docker manipulates iptables directly and can bypass `ufw`
rules for published container ports — so we don't rely on `ufw` to protect
the app itself. Instead, `docker-compose.prod.yml` (below) binds the
backend/frontend containers to `127.0.0.1` only, so they're unreachable from
outside the server no matter what `ufw` says. Only nginx (listening on
80/443, not in Docker) is exposed publicly.

## 3. Clone and configure

```sh
git clone git@github.com:<you>/cloud-cost-dashboard.git
cd cloud-cost-dashboard
cp backend/.env.example backend/.env
```

Generate real secrets and edit `backend/.env`:

```sh
openssl rand -base64 32   # run twice — once for JWT_SECRET, once for ENCRYPTION_KEY
```

```dotenv
# backend/.env
JWT_SECRET=<paste first random value>
ENCRYPTION_KEY=<paste second random value>
ADMIN_EMAIL=you@yourcompany.com
ADMIN_PASSWORD=<a real password — you'll change it via the app after first login>
SYNC_INTERVAL_MINUTES=60
```

## 4. Build and start

```sh
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
docker compose ps
```

Both containers should be `Up`, bound only to `127.0.0.1` (confirm with
`docker compose port backend 8080` → should print `127.0.0.1:8080`).

## 5. nginx + HTTPS

```sh
sudo tee /etc/nginx/sites-available/cloudcost > /dev/null << 'EOF'
server {
    listen 80;
    server_name cost.example.com;

    location /api/ {
        proxy_pass http://127.0.0.1:8080/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF

sudo ln -s /etc/nginx/sites-available/cloudcost /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# issues the cert and rewrites the config above to add the 443 block + redirect
sudo certbot --nginx -d cost.example.com
```

Replace `cost.example.com` with your real domain in both the nginx config
and the certbot command.

Visit `https://cost.example.com` — you should see the login page. Log in
with `ADMIN_EMAIL`/`ADMIN_PASSWORD` from `backend/.env`, then change the
password (Admin → Users — or add yourself a new admin user and stop using
the bootstrap one).

Certbot installs a systemd timer for automatic renewal — verify with
`sudo certbot renew --dry-run`.

## 6. Updating later

```sh
cd cloud-cost-dashboard
git pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

Containers restart automatically on server reboot (`restart:
unless-stopped`) as long as the Docker daemon starts on boot, which it does
by default after the install above.

## 7. Backing up the database

Cost history and hierarchy live in a Docker named volume (SQLite file). Back
it up periodically:

```sh
docker run --rm \
  -v cloud-cost-dashboard_backend_data:/data \
  -v "$PWD":/backup \
  alpine tar czf /backup/cloudcost-backup-$(date +%F).tar.gz -C /data .
```

Consider adding this as a daily cron job, copying the resulting `.tar.gz`
off the server.

---

## 8. Connect Azure

The dashboard reads cost via **Azure Cost Management** using a service
principal with **read-only** access. Easiest via the Azure CLI (run this
wherever you have `az` logged in — your laptop or Cloud Shell, not
necessarily the Ubuntu server):

```sh
az login
az account show --query id -o tsv
# ^ this is your subscription id — save it, it's the CloudAccount "External ID" in the dashboard

az ad sp create-for-rbac \
  --name "cost-dashboard-reader" \
  --role "Cost Management Reader" \
  --scopes /subscriptions/<SUBSCRIPTION_ID>
```

This prints something like:

```json
{
  "appId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "displayName": "cost-dashboard-reader",
  "password": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "tenant": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

Map those to the dashboard's credential fields: `tenant` → **Tenant ID**,
`appId` → **Client ID**, `password` → **Client Secret**.

In the dashboard (Admin page):
1. **Provider credentials** → Provider: Azure → fill in Tenant ID / Client
   ID / Client Secret → Save credential.
2. **Cloud accounts** → pick the Region you want this subscription under →
   Provider: Azure → Account name (anything, e.g. `prod-subscription`) →
   Account/Subscription/Project ID: the subscription id from `az account
   show` → select the credential you just saved → Add account.
3. Click **Sync now** on that account. If it fails, the error message
   (shown in the account row) usually tells you exactly what's missing —
   most commonly the service principal needs a minute to propagate, or the
   role assignment scope doesn't match the subscription.

Cost data then refreshes automatically every `SYNC_INTERVAL_MINUTES`.

---

## 9. Connect GCP

Unlike AWS/Azure, GCP has **no direct "get my costs" API** — billing detail
must be exported to BigQuery first, then the dashboard queries that table.
**Enable this before wiring up the credential — it takes ~24 hours for data
to start appearing (not retroactive), so do this step early.**

### 9a. Enable Billing Export to BigQuery

In the GCP Console:
1. Go to **Billing** → select your billing account → **Billing export**.
2. Under **BigQuery export**, click **Detailed usage cost** → **Edit
   settings**.
3. Pick (or create) a project + BigQuery dataset to export into — e.g.
   project `billing-exports`, dataset `billing_export`.
4. Save. GCP will start writing a table named like
   `gcp_billing_export_v1_<BILLING_ACCOUNT_ID>` into that dataset — **the
   first rows appear the following day**, not immediately.

### 9b. Create a read-only service account

Run with `gcloud` authenticated against the project that owns the BigQuery
export dataset:

```sh
gcloud config set project billing-exports   # the export project from 9a

gcloud iam service-accounts create cost-dashboard-reader \
  --display-name="Cost Dashboard Reader"

# lets it run BigQuery queries (billed to this project)
gcloud projects add-iam-policy-binding billing-exports \
  --member="serviceAccount:cost-dashboard-reader@billing-exports.iam.gserviceaccount.com" \
  --role="roles/bigquery.jobUser"

# lets it read the specific export dataset
bq add-iam-policy-binding \
  --member="serviceAccount:cost-dashboard-reader@billing-exports.iam.gserviceaccount.com" \
  --role="roles/bigquery.dataViewer" \
  billing-exports:billing_export

# download the key — treat this file like a password
gcloud iam service-accounts keys create cost-dashboard-key.json \
  --iam-account=cost-dashboard-reader@billing-exports.iam.gserviceaccount.com
```

Find the exact exported table name in the BigQuery Console under your
dataset (`billing_export` in this example) — it'll be
`gcp_billing_export_v1_XXXXXX_XXXXXX_XXXXXX`.

### 9c. Add it to the dashboard

1. **Provider credentials** → Provider: GCP → paste the **entire contents**
   of `cost-dashboard-key.json` into Service Account JSON → BigQuery
   Project: `billing-exports` → BigQuery Dataset: `billing_export` →
   BigQuery Table: the `gcp_billing_export_v1_...` name from above → Save
   credential.
2. **Cloud accounts** → pick a Region → Provider: GCP → Account name (e.g.
   `main-gcp-project`) → Account/Subscription/Project ID: **the specific
   GCP project ID you want costs for** (the export table can contain many
   projects — this filters to just this one) → select the credential →
   Add account.
3. Wait for data to exist in the export table (up to 24h after enabling in
   9a), then **Sync now**.

Delete `cost-dashboard-key.json` from your local machine once it's pasted
into the dashboard (or store it in a password manager) — it's a live
credential.

---

## Recap: what's public vs. private

- **Public (via nginx + HTTPS):** the dashboard UI and its `/api/*` routes.
- **Private (127.0.0.1 only):** the raw backend (8080) and frontend (3000)
  containers — never exposed directly to the internet.
- **Encrypted at rest:** Azure/GCP credentials, in the SQLite database,
  keyed by `ENCRYPTION_KEY` — back that key up somewhere safe (e.g. alongside
  your DB backups); losing it makes stored credentials unrecoverable and
  they'd need to be re-entered.
