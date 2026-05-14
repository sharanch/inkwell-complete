# Inkwell

A privacy-first blogging platform built on a Go microservices architecture.
No passwords — login with a one-time code sent to your email.

## Architecture

```
Frontend (React + Vite)
      │
      ▼
API Gateway (Go, port 8080)   ← JWT validation, routing
  ├── auth-service   :8081    ← OTP login, JWT issue/refresh
  ├── blog-service   :8082    ← Post CRUD, likes, visibility
  ├── feed-service   :8083    ← Personalised feed, interests
  └── notify-service :8084    ← SMTP email (internal only)

Data stores
  ├── postgres-auth   (users, OTP sessions)
  ├── postgres-blog   (posts, likes)
  ├── postgres-feed   (interests, feed index)
  └── redis           (OTP TTL, feed cache)
```

Each service owns its own database (database-per-service pattern).
Services communicate over HTTP. JWT is validated at the gateway — downstream
services trust the injected `X-User-ID` header.

## Tech stack

- **Backend**: Go 1.22, Chi router, sqlx, golang-jwt, go-redis
- **Frontend**: React 18, Vite, Tailwind CSS, React Router
- **Infra**: Kubernetes (minikube), Docker Compose, Postgres 16, Redis 7, Nginx
- **CI**: GitHub Actions — lint, test, build on every PR

---

## Option A — Docker Compose (local dev)

### Prerequisites
- Docker Desktop

```bash
# 1. Enter the project
cd inkwell-complete

# 2. Copy env file
cp .env.example .env

# 3. Start everything
docker compose up --build
```

- Frontend: http://localhost:3000
- API: http://localhost:8080

Without SMTP configured, OTP codes print to the notify-service logs:

```bash
docker compose logs notify-service
```

---

## Option B — Minikube (Kubernetes)

### Prerequisites
- Docker Desktop
- [minikube](https://minikube.sigs.k8s.io/docs/start/)
- kubectl

### Step 1 — Start minikube

```bash
make mk-start
```

Starts minikube with 4 CPUs / 6 GB RAM, enables ingress and metrics-server addons,
and adds `inkwell.local` to `/etc/hosts` (requires sudo).

### Step 2 — Build images

```bash
make build
```

Builds all service images directly into minikube's Docker daemon — no registry needed.

### Step 3 — Create secrets

Apply secrets directly (do not use `make k8s-secrets` — it requires an interactive editor):

```bash
cat > /tmp/inkwell-secrets.yaml << 'EOF'
apiVersion: v1
kind: Secret
metadata:
  name: inkwell-jwt-secrets
  namespace: inkwell
type: Opaque
stringData:
  JWT_SECRET: "your-strong-jwt-secret-at-least-32-chars"
  JWT_REFRESH_SECRET: "your-strong-refresh-secret-at-least-32-chars"
---
apiVersion: v1
kind: Secret
metadata:
  name: postgres-auth-secret
  namespace: inkwell
type: Opaque
stringData:
  POSTGRES_PASSWORD: "devpass"
  AUTH_DB_PASS: "devpass"
---
apiVersion: v1
kind: Secret
metadata:
  name: postgres-blog-secret
  namespace: inkwell
type: Opaque
stringData:
  POSTGRES_PASSWORD: "devpass"
  BLOG_DB_PASS: "devpass"
---
apiVersion: v1
kind: Secret
metadata:
  name: postgres-feed-secret
  namespace: inkwell
type: Opaque
stringData:
  POSTGRES_PASSWORD: "devpass"
  FEED_DB_PASS: "devpass"
---
apiVersion: v1
kind: Secret
metadata:
  name: smtp-secret
  namespace: inkwell
type: Opaque
stringData:
  SMTP_HOST: "smtp.gmail.com"
  SMTP_USER: "your-gmail@gmail.com"
  SMTP_PASS: "your-app-password"
  FROM_EMAIL: "noreply@yourdomain.com"
EOF
kubectl apply -f /tmp/inkwell-secrets.yaml
```

### Step 4 — Deploy

```bash
make k8s-apply
make k8s-ingress
```

### Step 5 — Check status

```bash
kubectl get pods -n inkwell
```

Wait until all pods show `1/1 Running`. Databases take ~30s.

### Access the app

- Frontend: http://inkwell.local
- API: http://inkwell.local/api/

---

## SMTP / Email setup

OTP login codes are sent by email. Three options:

### Option 1 — No SMTP (dev only)
Leave SMTP unconfigured. OTP codes print to notify-service logs:
```bash
kubectl logs -n inkwell -l app=notify-service --tail=50
# or
make k8s-logs svc=notify-service
```

### Option 2 — Gmail with a dedicated account
Create a throwaway Gmail account, enable 2FA, generate an App Password at
https://myaccount.google.com/apppasswords, then:

```bash
kubectl create secret generic smtp-secret \
  --from-literal=SMTP_HOST=smtp.gmail.com \
  --from-literal=SMTP_USER=throwaway@gmail.com \
  --from-literal=SMTP_PASS='your app password' \
  --from-literal=FROM_EMAIL=noreply@yourdomain.com \
  -n inkwell \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl rollout restart deployment/notify-service -n inkwell
```

> Use a throwaway account (not your main Gmail) so Gmail doesn't override
> the FROM_EMAIL with your default sending address.

### Option 3 — Resend (recommended for custom domains)
Sign up at resend.com, verify your domain, grab an API key, then:

```bash
kubectl create secret generic smtp-secret \
  --from-literal=SMTP_HOST=smtp.resend.com \
  --from-literal=SMTP_USER=resend \
  --from-literal=SMTP_PASS='re_your_api_key' \
  --from-literal=FROM_EMAIL=noreply@yourdomain.com \
  -n inkwell \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl rollout restart deployment/notify-service -n inkwell
```

---

## Services

| Service        | Port | Responsibility                         |
|----------------|------|----------------------------------------|
| api-gateway    | 8080 | JWT auth, reverse proxy, rate limiting |
| auth-service   | 8081 | OTP generation, JWT issue/refresh      |
| blog-service   | 8082 | Post CRUD, public/private visibility   |
| feed-service   | 8083 | Ranked feed, user interest management  |
| notify-service | 8084 | SMTP email sending (internal only)     |

## Auth flow

1. User enters email → POST `/api/v1/auth/request-otp`
2. `auth-service` generates 6-digit code, stores in Redis (10 min TTL), calls `notify-service`
3. `notify-service` sends email via SMTP
4. User enters code → POST `/api/v1/auth/verify-otp`
5. `auth-service` validates code (one-time use), upserts user in Postgres
6. Returns `access_token` (15 min) + `refresh_token` (7 days)
7. Frontend auto-refreshes on 401

---

## Useful commands

```bash
# Status
make mk-status                          # minikube + cluster overview
make k8s-status                         # pods, services, ingress

# Logs
make k8s-logs svc=auth-service          # tail pod logs
make k8s-logs svc=notify-service        # check OTP codes / SMTP errors

# Debugging
make k8s-exec svc=auth-service          # shell into a pod
make describe svc=blog-service          # describe pods (good for crash diagnosis)
make forward svc=api-gateway port=8080  # port-forward for direct access

# Rebuild a single service after code changes
make reload svc=blog-service

# Database access
make psql db=auth                       # psql into postgres-auth
make redis-cli                          # redis-cli

# Teardown
make k8s-delete                         # delete all inkwell resources (keeps minikube)
make mk-delete                          # delete minikube cluster entirely
```

## Local development (single service)

```bash
# Requires Postgres + Redis running locally
cd auth-service
go run ./cmd/server

# With hot reload (install air first: go install github.com/air-verse/air@latest)
air

# Tests
go test ./... -race -cover
```

---

## Roadmap

- [ ] Rich text editor for writing posts
- [ ] Post detail page with comments
- [ ] User profile page
- [ ] Feed score updates on like/view events (currently manual)
- [ ] WebSocket notifications
- [ ] Full-text search (pg trigrams or Meilisearch)
- [ ] Image upload (S3/R2)
