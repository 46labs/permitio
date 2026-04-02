set shell := ["bash", "-uc"]

default:
	@echo "Usage:"
	@echo "  just docker - Build and run container on localhost:7766"
	@echo "  just kind   - Create Kind cluster + start Tilt"
	@echo "  just ci     - Run gofmt check, tests, and lint"
	@echo "  just down   - Stop docker, tilt down, delete kind cluster"

docker:
	@echo "Building..."
	docker build -t permitio:dev .
	@echo "Starting on http://localhost:7766"
	docker run --rm -d \
		--name permitio \
		-p 7766:7766 \
		permitio:dev
	@echo "API: http://localhost:7766/v2/schema/resources"

ci:
	@echo "Checking format..."
	@gofmt -l .
	@echo "Running tests..."
	@go test -v ./pkg/...
	@echo "Running linter..."
	@golangci-lint run

# Context safety check
_context-guard:
	#!/usr/bin/env bash
	set -euo pipefail

	CURRENT_CONTEXT=$(kubectl config current-context 2>/dev/null || echo "none")
	ALLOWED_CONTEXTS=("kind-permitio" "docker-desktop" "none")

	for allowed in "${ALLOWED_CONTEXTS[@]}"; do
		if [[ "$CURRENT_CONTEXT" == "$allowed" ]]; then
			exit 0
		fi
	done

	echo "ERROR: Current kubectl context '$CURRENT_CONTEXT' is not allowed"
	echo "Allowed contexts: ${ALLOWED_CONTEXTS[*]}"
	echo "Switch context or update ALLOWED_CONTEXTS in justfile"
	exit 1

kind: _context-guard
	#!/usr/bin/env bash
	set -euo pipefail

	if ! kind get clusters 2>/dev/null | grep -q "^permitio$"; then
		echo "Creating Kind cluster..."
		kind create cluster --name permitio
		kubectl config use-context kind-permitio

		echo "Installing nginx-ingress..."
		helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx 2>/dev/null || true
		helm repo update
		helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx \
			--namespace ingress-nginx --create-namespace \
			--set controller.service.type=NodePort \
			--set controller.service.nodePorts.http=30080 \
			--set controller.service.nodePorts.https=30443 \
			--set controller.ingressClassResource.default=true \
			--wait --timeout=5m
	else
		echo "Kind cluster exists"
		kubectl config use-context kind-permitio
	fi

	echo "Starting Tilt..."
	tilt up

down:
	@echo "Stopping..."
	docker stop permitio 2>/dev/null || true
	tilt down 2>/dev/null || true
	kind delete cluster --name permitio 2>/dev/null || true
