.PHONY: up down run run-worker

up:
	@./scripts/start.sh

down:
	@./scripts/stop.sh

test:
	go test ./...

k8s-platform:
	@echo "Installing ESO..."
	helm repo add external-secrets https://charts.external-secrets.io
	helm repo update
	helm install external-secrets external-secrets/external-secrets -n external-secrets --create-namespace
	@echo "Deploying Vault..."
	kubectl apply -f k8s/platform/vault.yaml

k8s-secrets:
	@echo "Configuring Secrets..."
	kubectl apply -f k8s/secrets/secret-store.yaml
	kubectl apply -f k8s/secrets/external-secret.yaml
	@echo "Waiting for Vault..."
	kubectl wait --for=condition=ready pod -l app=vault --timeout=60s
	@echo "Writing secret to Vault (Dev Mode)..."
	kubectl exec deploy/vault -- vault kv put secret/project/config POSTGRES_PASSWORD=secret_password_from_vault

