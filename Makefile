all: build-cli

build-cli:
	cd cli && go build -v && mv cli ../oms

build-cli-linux:
	GOOS=linux GOARCH=amd64 go build -C cli -o ../oms

test:
	# -count=1 to disable caching test results
	go test -count=1 -v ./...

test-integration:
	# Run integration tests with build tag
	go test -count=1 -v -tags=integration ./cli/...

format:
	go fmt ./...

lint: install-build-deps
	go tool golangci-lint run

install-build-deps:
ifeq (, $(shell which copywrite))
	go install github.com/hashicorp/copywrite@v0.22.0
endif

generate: install-build-deps
	go tool mockery
	go generate ./...

VERSION ?= "0.0.0"
LIMA_INSTANCE ?= lima-oms
LIMA_HOST_HOME ?= /home/user/host-home
# Lima mounts the host home directory at /home/user/host-home inside the VM.
LIMA_VM_WORKTREE ?= $(patsubst $(HOME)%,$(LIMA_HOST_HOME)%,$(CURDIR))
LIMA_K0S_INSTALL_CONFIG ?= .installer/lima-k0s-install-config.yaml
LIMA_K0S_INSTALL_CONFIG_VM ?= $(LIMA_VM_WORKTREE)/.installer/lima-k0s-install-config.yaml
LIMA_K0SCTL_CONFIG ?= oms-workdir/k0sctl-config-lima-oms.yaml
LIMA_KUBECONFIG ?= $(HOME)/.kube/$(LIMA_INSTANCE).yaml
LIMA_KUBECONFIG_VM ?= $(patsubst $(HOME)%,$(LIMA_HOST_HOME)%,$(LIMA_KUBECONFIG))
LIMA_SSH_KEY_VM ?= $(LIMA_HOST_HOME)/.lima/_config/user
LIMA_K0S_VERSION ?= v1.30.0+k0s.0
OMS_BOOTSTRAP_LOCAL_ARGS ?=

release-local: install-build-deps
	rm -rf dist
	/bin/bash -c "go tool goreleaser --snapshot --skip=validate,announce,publish -f <(sed s/{{.Version}}/$(VERSION)/g < .goreleaser.yaml)"

.PHONY: docs run-lima run-lima-on-macos check-lima-prereqs-on-macos install-k0s-in-lima bootstrap-local-on-macos stop-lima
docs:
	rm -rf docs
	mkdir docs
	go run -ldflags="-X 'github.com/codesphere-cloud/oms/internal/version.binName=oms'" hack/gendocs/main.go
	cp docs/oms.md docs/README.md

generate-license: generate
	go tool go-licenses report --template .NOTICE.template ./... > NOTICE
	copywrite headers apply

generate-notice:
	go tool go-licenses report --template .NOTICE.template ./... > NOTICE

run-lima: run-lima-on-macos

check-lima-prereqs-on-macos:
	@if ! limactl info 2>/dev/null | awk 'BEGIN { in_guestagents = 0; in_arch = 0; found = 0 } /"guestAgents": \{/ { in_guestagents = 1; next } in_guestagents && /"x86_64": \{/ { in_arch = 1; next } in_arch && /"location":/ { found = 1; exit 0 } in_arch && /^[[:space:]]*}/ { exit 1 } END { exit(found ? 0 : 1) }'; then \
		echo "Missing Lima Linux x86_64 guest agent."; \
		echo "Install it with: brew install lima-additional-guestagents"; \
		exit 1; \
	fi

run-lima-on-macos: check-lima-prereqs-on-macos
	@if [ -d "$$HOME/.lima/$(LIMA_INSTANCE)" ]; then \
		limactl start $(LIMA_INSTANCE); \
	else \
		limactl start ./hack/lima-oms.yaml; \
	fi
	@echo "Lima instance '$(LIMA_INSTANCE)' is ready."
	@echo "Next: make install-k0s-in-lima"

install-k0s-in-lima: build-cli-linux
	@mkdir -p .installer "$(dir $(LIMA_KUBECONFIG))"
	@VM_IP=$$(limactl shell --workdir "$(LIMA_VM_WORKTREE)" $(LIMA_INSTANCE) -- /bin/bash -lc "hostname -I | cut -d ' ' -f1"); \
	printf '%s\n' \
		'dataCenter:' \
		'  id: 1' \
		'  name: lima-oms' \
		'  city: Lima' \
		'  countryCode: US' \
		'kubernetes:' \
		'  managedByCodesphere: true' \
		"  apiServerHost: $$VM_IP" \
		'  controlPlanes:' \
		"    - ipAddress: $$VM_IP" \
		'  podCidr: 100.96.0.0/11' \
		'  serviceCidr: 100.64.0.0/13' \
		'codesphere:' \
		'  domain: cs.local' \
		"  publicIP: $$VM_IP" \
		'  deployConfig:' \
		'    images: {}' \
		'  plans:' \
		'    hostingPlans: {}' \
		'    workspacePlans: {}' \
		> "$(LIMA_K0S_INSTALL_CONFIG)"
	@limactl shell --workdir "$(LIMA_VM_WORKTREE)" $(LIMA_INSTANCE) -- /bin/bash -lc 'set -euo pipefail; sudo install -d -m 700 /root/.ssh; sudo cp "$$HOME/.ssh/authorized_keys" /root/.ssh/authorized_keys; sudo chmod 600 /root/.ssh/authorized_keys; cd "$(LIMA_VM_WORKTREE)"; sudo ./oms install k0s --install-config "$(LIMA_K0S_INSTALL_CONFIG_VM)" --ssh-key-path "$(LIMA_SSH_KEY_VM)" --version "$(LIMA_K0S_VERSION)" --force'
	@limactl shell --workdir "$(LIMA_VM_WORKTREE)" $(LIMA_INSTANCE) -- /bin/bash -lc 'set -euo pipefail; sudo /root/.cache/oms/k0sctl kubeconfig --config "$(LIMA_VM_WORKTREE)/$(LIMA_K0SCTL_CONFIG)" > "$(LIMA_KUBECONFIG_VM)"'
	@VM_IP=$$(limactl shell --workdir "$(LIMA_VM_WORKTREE)" $(LIMA_INSTANCE) -- /bin/bash -lc "hostname -I | cut -d ' ' -f1"); \
	sed -i.bak "s#https://$$VM_IP:6443#https://127.0.0.1:6443#g" "$(LIMA_KUBECONFIG)"; \
	rm -f "$(LIMA_KUBECONFIG).bak"
	@chmod 600 "$(LIMA_KUBECONFIG)"
	@$(MAKE) build-cli
	@echo "KUBECONFIG written to $(LIMA_KUBECONFIG)"
	@echo "Next: OMS_REGISTRY_USER=<github-user-or-service-account> OMS_INSTALL_LOCAL=/absolute/path/to/installer-lite.tar.gz make bootstrap-local-on-macos"

bootstrap-local-on-macos: build-cli
	@if [ ! -f "$(LIMA_KUBECONFIG)" ]; then echo "Missing $(LIMA_KUBECONFIG). Run 'make install-k0s-in-lima' first."; exit 1; fi
	@if [ -z "$(OMS_REGISTRY_USER)" ]; then echo "OMS_REGISTRY_USER is required."; exit 1; fi
	@if [ -z "$(OMS_INSTALL_LOCAL)" ] && { [ -z "$(OMS_INSTALL_VERSION)" ] || [ -z "$(OMS_INSTALL_HASH)" ]; }; then echo "Set OMS_INSTALL_LOCAL or both OMS_INSTALL_VERSION and OMS_INSTALL_HASH."; exit 1; fi
	@if [ -z "$(OMS_INSTALL_LOCAL)" ] && [ -z "$(OMS_PORTAL_API_KEY)" ]; then echo "OMS_PORTAL_API_KEY is required when OMS_INSTALL_VERSION/OMS_INSTALL_HASH are used."; exit 1; fi
	@if [ -z "$${OMS_REGISTRY_PASSWORD:-}" ]; then echo "OMS_REGISTRY_PASSWORD is not set; oms will prompt for it."; fi
	@if [ -n "$(OMS_INSTALL_LOCAL)" ]; then \
		KUBECONFIG="$(LIMA_KUBECONFIG)" ./oms beta bootstrap-local --yes --k0s --registry-user "$(OMS_REGISTRY_USER)" --install-local "$(OMS_INSTALL_LOCAL)" $(OMS_BOOTSTRAP_LOCAL_ARGS); \
	else \
		KUBECONFIG="$(LIMA_KUBECONFIG)" ./oms beta bootstrap-local --yes --k0s --registry-user "$(OMS_REGISTRY_USER)" --install-version "$(OMS_INSTALL_VERSION)" --install-hash "$(OMS_INSTALL_HASH)" $(OMS_BOOTSTRAP_LOCAL_ARGS); \
	fi

stop-lima:
	limactl stop $(LIMA_INSTANCE)
	limactl delete $(LIMA_INSTANCE)
