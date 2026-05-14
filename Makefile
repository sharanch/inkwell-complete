# ─────────────────────────────────────────────────────────────────────────────
#  Inkwell — Makefile
#  Usage: make <target> [svc=<service-name>]
# ─────────────────────────────────────────────────────────────────────────────

NAMESPACE     := inkwell
SERVICES      := api-gateway auth-service blog-service feed-service notify-service frontend
MINIKUBE_IP   := $(shell minikube ip 2>/dev/null)
HOST_ENTRY    := $(MINIKUBE_IP) inkwell.local

.DEFAULT_GOAL := help

# ─── Colours ─────────────────────────────────────────────────────────────────
BOLD  := \033[1m
RESET := \033[0m
GREEN := \033[32m
CYAN  := \033[36m
YELLOW:= \033[33m
RED   := \033[31m

# ─────────────────────────────────────────────────────────────────────────────
#  HELP
# ─────────────────────────────────────────────────────────────────────────────
.PHONY: help
help:
	@echo ""
	@echo "$(BOLD)Inkwell — available targets$(RESET)"
	@echo ""
	@echo "$(CYAN)── Local dev (Docker Compose) ──────────────────────────────────$(RESET)"
	@echo "  $(BOLD)make up$(RESET)               Start everything with docker compose"
	@echo "  $(BOLD)make down$(RESET)             Stop and remove containers"
	@echo "  $(BOLD)make logs$(RESET)             Tail all service logs"
	@echo "  $(BOLD)make logs svc=auth-service$(RESET)  Tail a specific service"
	@echo "  $(BOLD)make restart svc=blog-service$(RESET)  Restart one service"
	@echo ""
	@echo "$(CYAN)── Minikube ─────────────────────────────────────────────────────$(RESET)"
	@echo "  $(BOLD)make mk-start$(RESET)         Start minikube with correct resources"
	@echo "  $(BOLD)make mk-stop$(RESET)          Stop minikube"
	@echo "  $(BOLD)make mk-delete$(RESET)        Delete minikube cluster entirely"
	@echo "  $(BOLD)make mk-status$(RESET)        Show minikube + cluster status"
	@echo "  $(BOLD)make mk-dashboard$(RESET)     Open Kubernetes dashboard"
	@echo "  $(BOLD)make mk-hosts$(RESET)         Add inkwell.local to /etc/hosts"
	@echo ""
	@echo "$(CYAN)── Build (minikube Docker daemon) ───────────────────────────────$(RESET)"
	@echo "  $(BOLD)make build$(RESET)            Build all service images into minikube"
	@echo "  $(BOLD)make build svc=blog-service$(RESET)  Build one service"
	@echo ""
	@echo "$(CYAN)── Kubernetes deploy ────────────────────────────────────────────$(RESET)"
	@echo "  $(BOLD)make k8s-setup$(RESET)        Full first-time cluster setup"
	@echo "  $(BOLD)make k8s-apply$(RESET)        Apply all manifests"
	@echo "  $(BOLD)make k8s-secrets$(RESET)      Create secrets from template (prompts for values)"
	@echo "  $(BOLD)make k8s-status$(RESET)       Show all pods, services, ingress"
	@echo "  $(BOLD)make k8s-delete$(RESET)       Delete all inkwell resources (keeps cluster)"
	@echo ""
	@echo "$(CYAN)── Reload (rebuild + redeploy one service) ──────────────────────$(RESET)"
	@echo "  $(BOLD)make reload svc=blog-service$(RESET)"
	@echo ""
	@echo "$(CYAN)── Logs & debug ─────────────────────────────────────────────────$(RESET)"
	@echo "  $(BOLD)make k8s-logs svc=api-gateway$(RESET)  Tail k8s pod logs"
	@echo "  $(BOLD)make k8s-exec svc=auth-service$(RESET) Shell into a pod"
	@echo "  $(BOLD)make forward svc=api-gateway port=8080$(RESET)  Port-forward a service"
	@echo "  $(BOLD)make describe svc=blog-service$(RESET) Describe pods for a service"
	@echo ""
	@echo "$(CYAN)── Database ─────────────────────────────────────────────────────$(RESET)"
	@echo "  $(BOLD)make psql db=auth$(RESET)     Connect to postgres-auth (port-forward)"
	@echo "  $(BOLD)make redis-cli$(RESET)        Connect to Redis"
	@echo ""
	@echo "$(CYAN)── Production / CI ──────────────────────────────────────────────$(RESET)"
	@echo "  $(BOLD)make lint$(RESET)             golangci-lint all Go services"
	@echo "  $(BOLD)make test$(RESET)             go test ./... -race on all services"
	@echo "  $(BOLD)make test svc=auth-service$(RESET)  Test one service"
	@echo ""

# ─────────────────────────────────────────────────────────────────────────────
#  DOCKER COMPOSE (local dev)
# ─────────────────────────────────────────────────────────────────────────────
.PHONY: up down logs restart

up:
	@echo "$(GREEN)Starting Inkwell with Docker Compose...$(RESET)"
	docker compose up --build -d
	@echo "$(GREEN)✓ Frontend: http://localhost:3000$(RESET)"
	@echo "$(GREEN)✓ API:      http://localhost:8080$(RESET)"

down:
	@echo "$(YELLOW)Stopping Inkwell...$(RESET)"
	docker compose down

logs:
ifdef svc
	docker compose logs -f $(svc)
else
	docker compose logs -f
endif

restart:
ifndef svc
	$(error svc is required: make restart svc=<service-name>)
endif
	docker compose restart $(svc)

# ─────────────────────────────────────────────────────────────────────────────
#  MINIKUBE
# ─────────────────────────────────────────────────────────────────────────────
.PHONY: mk-start mk-stop mk-delete mk-status mk-dashboard mk-hosts

mk-start:
	@echo "$(GREEN)Starting minikube...$(RESET)"
	minikube start \
		--cpus=4 \
		--memory=6g \
		--disk-size=20g \
		--driver=docker \
		--addons=ingress,metrics-server
	@echo "$(GREEN)✓ minikube is up$(RESET)"
	@$(MAKE) mk-hosts

mk-stop:
	minikube stop

mk-delete:
	@echo "$(RED)Deleting minikube cluster...$(RESET)"
	minikube delete

mk-status:
	@echo "$(CYAN)── minikube ──$(RESET)"
	minikube status
	@echo ""
	@echo "$(CYAN)── nodes ──$(RESET)"
	kubectl get nodes
	@echo ""
	@echo "$(CYAN)── inkwell pods ──$(RESET)"
	kubectl get pods -n $(NAMESPACE) 2>/dev/null || echo "(namespace not created yet)"

mk-dashboard:
	minikube dashboard

mk-hosts:
	@echo "$(CYAN)Checking /etc/hosts for inkwell.local...$(RESET)"
	@if grep -q "inkwell.local" /etc/hosts; then \
		echo "$(YELLOW)⚠ inkwell.local already in /etc/hosts — skipping$(RESET)"; \
	else \
		echo "$(GREEN)Adding $(HOST_ENTRY) to /etc/hosts (requires sudo)...$(RESET)"; \
		echo "$(HOST_ENTRY)" | sudo tee -a /etc/hosts; \
		echo "$(GREEN)✓ Added$(RESET)"; \
	fi

# ─────────────────────────────────────────────────────────────────────────────
#  BUILD (into minikube's Docker daemon)
# ─────────────────────────────────────────────────────────────────────────────
.PHONY: build _build-one

build:
ifdef svc
	@$(MAKE) _build-one SERVICE=$(svc)
else
	@echo "$(GREEN)Building all services into minikube...$(RESET)"
	@eval $$(minikube docker-env) && \
	for svc in $(SERVICES); do \
		echo "$(CYAN)  → building $$svc$(RESET)"; \
		docker build -t inkwell/$$svc:dev ./$$svc || exit 1; \
	done
	@echo "$(GREEN)✓ All images built$(RESET)"
endif

_build-one:
	@echo "$(GREEN)Building $(SERVICE) into minikube...$(RESET)"
	@eval $$(minikube docker-env) && \
		docker build -t inkwell/$(SERVICE):dev ./$(SERVICE)
	@echo "$(GREEN)✓ inkwell/$(SERVICE):dev ready$(RESET)"

# ─────────────────────────────────────────────────────────────────────────────
#  KUBERNETES DEPLOY
# ─────────────────────────────────────────────────────────────────────────────
.PHONY: k8s-setup k8s-patch k8s-apply k8s-secrets k8s-status k8s-delete k8s-ingress

# Full first-time setup
k8s-setup: mk-start build k8s-secrets k8s-patch k8s-apply k8s-ingress
	@echo ""
	@echo "$(GREEN)$(BOLD)✓ Inkwell is running on Kubernetes!$(RESET)"
	@echo "$(GREEN)  Frontend: http://inkwell.local$(RESET)"
	@echo "$(GREEN)  API:      http://inkwell.local/api/$(RESET)"
	@echo ""
	@$(MAKE) k8s-status

# Patch manifests for local dev (imagePullPolicy + local image tags)
k8s-patch:
	@echo "$(CYAN)Patching manifests for local dev...$(RESET)"
	@for f in infra/k8s/services/*/deployment.yaml; do \
		sed -i.bak \
			-e 's|ghcr.io/sharanch/inkwell/\(.*\):latest|inkwell/\1:dev|g' \
			-e 's|imagePullPolicy: Always|imagePullPolicy: Never|g' \
			"$$f"; \
		rm -f "$$f.bak"; \
	done
	@echo "$(GREEN)✓ Manifests patched$(RESET)"

k8s-apply: k8s-patch
	@echo "$(CYAN)Applying manifests...$(RESET)"
	kubectl apply -f infra/k8s/base/namespaces/namespaces.yaml
	kubectl apply -f infra/k8s/base/configmaps/configmaps.yaml
	kubectl apply -f infra/k8s/base/rbac.yaml
	kubectl apply -f infra/k8s/databases/ --recursive
	kubectl apply -f infra/k8s/services/ --recursive
	@echo "$(GREEN)✓ Manifests applied$(RESET)"

k8s-ingress:
	@echo "$(CYAN)Applying dev ingress (no TLS)...$(RESET)"
	kubectl apply -f infra/k8s/networking/ingress/ingress-dev.yaml
	@echo "$(GREEN)✓ Ingress applied — http://inkwell.local$(RESET)"

k8s-secrets:
	@echo "$(CYAN)Creating Kubernetes secrets...$(RESET)"
	@if kubectl get secret inkwell-jwt-secrets -n $(NAMESPACE) &>/dev/null; then \
		echo "$(YELLOW)⚠ Secrets already exist — skipping. Run 'make k8s-delete-secrets' to recreate.$(RESET)"; \
	else \
		cp infra/k8s/base/secrets/secrets.template.yaml /tmp/inkwell-secrets.yaml; \
		echo "$(YELLOW)Edit /tmp/inkwell-secrets.yaml with dev values, then press Enter...$(RESET)"; \
		read; \
		kubectl apply -f /tmp/inkwell-secrets.yaml; \
		rm /tmp/inkwell-secrets.yaml; \
		echo "$(GREEN)✓ Secrets created$(RESET)"; \
	fi

k8s-delete-secrets:
	kubectl delete secret inkwell-jwt-secrets postgres-auth-secret postgres-blog-secret postgres-feed-secret smtp-secret -n $(NAMESPACE) --ignore-not-found

k8s-status:
	@echo "$(CYAN)── pods ──$(RESET)"
	kubectl get pods -n $(NAMESPACE)
	@echo ""
	@echo "$(CYAN)── services ──$(RESET)"
	kubectl get svc -n $(NAMESPACE)
	@echo ""
	@echo "$(CYAN)── ingress ──$(RESET)"
	kubectl get ingress -n $(NAMESPACE)
	@echo ""
	@echo "$(CYAN)── HPAs ──$(RESET)"
	kubectl get hpa -n $(NAMESPACE) 2>/dev/null || true

k8s-delete:
	@echo "$(RED)Deleting all inkwell resources...$(RESET)"
	kubectl delete namespace $(NAMESPACE) --ignore-not-found
	@echo "$(GREEN)✓ Done (cluster still running)$(RESET)"

# ─────────────────────────────────────────────────────────────────────────────
#  RELOAD — rebuild + redeploy a single service
# ─────────────────────────────────────────────────────────────────────────────
.PHONY: reload

reload:
ifndef svc
	$(error svc is required: make reload svc=<service-name>)
endif
	@echo "$(CYAN)Reloading $(svc)...$(RESET)"
	@eval $$(minikube docker-env) && \
		docker build -t inkwell/$(svc):dev ./$(svc)
	kubectl rollout restart deployment/$(svc) -n $(NAMESPACE)
	kubectl rollout status deployment/$(svc) -n $(NAMESPACE)
	@echo "$(GREEN)✓ $(svc) reloaded$(RESET)"

# ─────────────────────────────────────────────────────────────────────────────
#  LOGS & DEBUG
# ─────────────────────────────────────────────────────────────────────────────
.PHONY: k8s-logs k8s-exec forward describe

k8s-logs:
ifndef svc
	$(error svc is required: make k8s-logs svc=<service-name>)
endif
	kubectl logs -n $(NAMESPACE) -l app=$(svc) -f --tail=100

k8s-exec:
ifndef svc
	$(error svc is required: make k8s-exec svc=<service-name>)
endif
	kubectl exec -it -n $(NAMESPACE) \
		$$(kubectl get pod -n $(NAMESPACE) -l app=$(svc) -o jsonpath='{.items[0].metadata.name}') \
		-- /bin/sh

forward:
ifndef svc
	$(error usage: make forward svc=<service-name> port=<local-port>)
endif
ifndef port
	$(error usage: make forward svc=<service-name> port=<local-port>)
endif
	kubectl port-forward -n $(NAMESPACE) svc/$(svc) $(port):$(port)

describe:
ifndef svc
	$(error svc is required: make describe svc=<service-name>)
endif
	kubectl describe pods -n $(NAMESPACE) -l app=$(svc)

# ─────────────────────────────────────────────────────────────────────────────
#  DATABASE
# ─────────────────────────────────────────────────────────────────────────────
.PHONY: psql redis-cli

psql:
ifndef db
	$(error usage: make psql db=auth|blog|feed)
endif
	@echo "$(CYAN)Port-forwarding postgres-$(db):5432 → localhost:5432...$(RESET)"
	@kubectl port-forward -n $(NAMESPACE) svc/postgres-$(db) 5432:5432 &
	@sleep 1
	@PGPASSWORD=$$(kubectl get secret postgres-$(db)-secret -n $(NAMESPACE) \
		-o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d) \
		psql -h localhost -U $(db)_user $(db)_db

redis-cli:
	@echo "$(CYAN)Port-forwarding redis:6379 → localhost:6379...$(RESET)"
	@kubectl port-forward -n $(NAMESPACE) svc/redis 6379:6379 &
	@sleep 1
	redis-cli -h localhost

# ─────────────────────────────────────────────────────────────────────────────
#  LINT & TEST
# ─────────────────────────────────────────────────────────────────────────────
.PHONY: lint test

lint:
ifdef svc
	@echo "$(CYAN)Linting $(svc)...$(RESET)"
	cd $(svc) && golangci-lint run ./...
else
	@echo "$(CYAN)Linting all Go services...$(RESET)"
	@for svc in api-gateway auth-service blog-service feed-service notify-service; do \
		echo "$(CYAN)  → $$svc$(RESET)"; \
		cd $$svc && golangci-lint run ./... && cd ..; \
	done
endif

test:
ifdef svc
	@echo "$(CYAN)Testing $(svc)...$(RESET)"
	cd $(svc) && go test ./... -race -cover
else
	@echo "$(CYAN)Testing all Go services...$(RESET)"
	@for svc in api-gateway auth-service blog-service feed-service notify-service; do \
		echo "$(CYAN)  → $$svc$(RESET)"; \
		(cd $$svc && go test ./... -race -cover) || exit 1; \
	done
endif

# ─────────────────────────────────────────────────────────────────────────────
#  WAIT HELPER (used internally)
# ─────────────────────────────────────────────────────────────────────────────
.PHONY: wait-ready
wait-ready:
	@echo "$(CYAN)Waiting for all pods to be ready...$(RESET)"
	@for svc in $(SERVICES); do \
		echo "  waiting for $$svc..."; \
		kubectl rollout status deployment/$$svc -n $(NAMESPACE) --timeout=180s || true; \
	done
	@echo "$(GREEN)✓ All pods ready$(RESET)"
