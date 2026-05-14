# Inkwell — Infrastructure

Production-ready Kubernetes infra for the Inkwell microservices platform.

## Architecture Overview

```
Internet
    │  HTTPS (443) / HTTP→HTTPS redirect (80)
    ▼
nginx Ingress Controller
    │  TLS termination (cert-manager + Let's Encrypt)
    ▼
Istio IngressGateway
    │  VirtualService routing
    ├──► /            → frontend (React, Nginx)
    └──► /api/*       → api-gateway (Go, :8080)
                              │  JWT validation + reverse proxy
              ┌───────────────┼──────────────────┐
              ▼               ▼                  ▼
       auth-service    blog-service        feed-service
         (:8081)         (:8082)             (:8083)
              │                                  │
              ▼                                  │
       notify-service                            │
         (:8084)                                 │
    (internal only —                             │
     no external access)                         │

Data stores (StatefulSets with PVCs):
  postgres-auth  ← auth-service
  postgres-blog  ← blog-service
  postgres-feed  ← feed-service
  redis          ← auth-service, feed-service
```

All service-to-service traffic is **mTLS encrypted** via Istio.

---

## Directory Layout

```
infra/
├── k8s/
│   ├── base/
│   │   ├── namespaces/     # inkwell, inkwell-monitoring namespaces
│   │   ├── configmaps/     # Non-secret config for all services
│   │   ├── secrets/        # Secret templates (use sealed-secrets or ESO in prod)
│   │   └── rbac.yaml       # ServiceAccount + Role + RoleBinding
│   ├── databases/
│   │   ├── postgres-auth/  # StatefulSet + headless Service
│   │   ├── postgres-blog/
│   │   ├── postgres-feed/
│   │   └── redis/
│   ├── services/
│   │   ├── api-gateway/    # Deployment, Service, HPA, PDB
│   │   ├── auth-service/
│   │   ├── blog-service/
│   │   ├── feed-service/
│   │   ├── notify-service/
│   │   └── frontend/
│   ├── networking/
│   │   ├── istio/          # Gateway, VirtualService, DestinationRules,
│   │   │                   # PeerAuthentication, AuthorizationPolicies
│   │   └── ingress/        # nginx Ingress, cert-manager Certificate + ClusterIssuer
│   └── kustomization.yaml
├── argocd/
│   ├── projects/           # AppProject with RBAC
│   └── apps/               # App-of-Apps + individual Application manifests
└── github-actions/
    ├── ci.yml              # Lint → Test → Build → Push (per-service change detection)
    ├── cd.yml              # staging auto-deploy → production (manual approval)
    └── security.yml        # Trivy, Gitleaks, govulncheck, kube-score
```

---

## Prerequisites

| Tool | Purpose |
|------|---------|
| `kubectl` ≥ 1.29 | Cluster management |
| `istioctl` ≥ 1.21 | Service mesh |
| `argocd` CLI | GitOps deployments |
| `helm` ≥ 3.14 | Installing cert-manager, nginx-ingress |
| `kubeseal` | Encrypting secrets for git (optional but recommended) |

---

## First-Time Cluster Setup

### 1. Install Istio

```bash
istioctl install --set profile=default -y
# Enable sidecar injection for inkwell namespace
kubectl label namespace inkwell istio-injection=enabled
```

### 2. Install nginx Ingress Controller

```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx --create-namespace
```

### 3. Install cert-manager

```bash
helm repo add jetstack https://charts.jetstack.io
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager --create-namespace \
  --set installCRDs=true
```

### 4. Install ArgoCD

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Get initial admin password
argocd admin initial-password -n argocd

# Access UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
```

### 5. Create Secrets

Fill in the template and apply (or use sealed-secrets):

```bash
cp infra/k8s/base/secrets/secrets.template.yaml /tmp/secrets.yaml
# Edit /tmp/secrets.yaml with real values
kubectl apply -f /tmp/secrets.yaml
rm /tmp/secrets.yaml   # Never commit!
```

**Recommended: Sealed Secrets**
```bash
# Install sealed-secrets controller
helm install sealed-secrets sealed-secrets/sealed-secrets -n kube-system

# Encrypt a secret
kubeseal --format yaml < /tmp/secrets.yaml > infra/k8s/base/secrets/sealed-secrets.yaml
# Now safe to commit sealed-secrets.yaml
```

### 6. Bootstrap ArgoCD

```bash
# Register your git repo
argocd repo add https://github.com/sharanch/inkwell.git \
  --username git --password YOUR_PAT

# Apply the App-of-Apps
kubectl apply -f infra/argocd/projects/inkwell-project.yaml
kubectl apply -f infra/argocd/apps/applications.yaml

# ArgoCD will now sync and deploy everything
argocd app list
```

---

## GitHub Actions Setup

Add these secrets to your GitHub repository:

| Secret | Description |
|--------|-------------|
| `KUBECONFIG_STAGING` | base64-encoded kubeconfig for staging cluster |
| `KUBECONFIG_PRODUCTION` | base64-encoded kubeconfig for production cluster |
| `CODECOV_TOKEN` | Codecov upload token |
| `SLACK_WEBHOOK_URL` | Slack incoming webhook for deploy notifications |

Set up GitHub **Environments**:
- `staging` — no protection rules (auto-deploys)
- `production` — required reviewers (manual approval gate)

---

## GitOps Flow

```
Developer pushes PR
        │
        ▼
    CI runs (ci.yml)
    ├── golangci-lint per service
    ├── go test ./... -race -cover
    ├── frontend lint + build
        │
        ▼ (on main merge)
    Build & push Docker images to GHCR
    Update image tags in infra/k8s/**
    Commit back to main [skip ci]
        │
        ▼
    ArgoCD detects manifest change
    Auto-syncs to staging cluster
        │
        ▼ (cd.yml — manual approval)
    Deploy to production
```

---

## Production Patterns in Use

| Pattern | Implementation |
|---------|---------------|
| **mTLS everywhere** | Istio PeerAuthentication STRICT mode |
| **Zero-trust networking** | Istio AuthorizationPolicies (allowlist only) |
| **Circuit breaking** | Istio DestinationRules with outlierDetection |
| **Retry logic** | Istio VirtualService retries (3x, gateway-error) |
| **Auto-scaling** | HPA on CPU+memory for all services |
| **High availability** | 2+ replicas + topologySpreadConstraints |
| **Graceful shutdown** | terminationGracePeriodSeconds + preStop sleep |
| **Rolling updates** | maxSurge=1, maxUnavailable=0 on all Deployments |
| **Disruption budgets** | PDB minAvailable=1 on all services |
| **Data safety** | StatefulSets with PVCs, ArgoCD prune=false on databases |
| **TLS termination** | cert-manager + Let's Encrypt via nginx Ingress |
| **Secret hygiene** | Secrets never in git — use sealed-secrets or ESO |
| **Vulnerability scanning** | Trivy on every image build |
| **Change detection** | dorny/paths-filter — only rebuild changed services |

---

## Useful Commands

```bash
# Watch all pods
kubectl get pods -n inkwell -w

# Tail logs for a service
kubectl logs -n inkwell -l app=api-gateway -f --tail=100

# Port-forward a service for local debugging
kubectl port-forward -n inkwell svc/api-gateway 8080:8080

# Force ArgoCD sync
argocd app sync inkwell-services

# Roll back a deployment
kubectl rollout undo deployment/blog-service -n inkwell

# Check HPA status
kubectl get hpa -n inkwell

# Inspect Istio config
istioctl analyze -n inkwell
istioctl proxy-status
```
