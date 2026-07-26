# OMS — Repo Agent Instructions

Baseline: `~/.agents/AGENTS.md` applies unless overridden below.

## Project

Go CLI and library for bootstrapping, installing, and managing Codesphere OMS clusters.
OMS provisions full private-cloud stacks: k0s (Kubernetes), Ceph (storage), PostgreSQL,
Codesphere platform services, Argo CD, and extensible managed-service backends.

### Package layout

- `cli/` — CLI commands; actively being refactored into subpackages (`cli/cmd/<domain>/`)
- `internal/` — core libraries: `installer/`, `bootstrap/`, `portal/`, `codesphere/`, `vault/`
- `hack/` — build/tooling scripts, doc generation
- `docs/` — auto-generated CLI docs (do not hand-edit; run `make docs`)

### Org context

- [`cs-go`](https://github.com/codesphere-cloud/cs-go) — shared Go SDK;
  only org dependency in go.mod
- [`managed-services-lib`](https://github.com/codesphere-cloud/managed-services-lib) —
  framework for managed-service provider backends
  ([Keycloak](https://github.com/codesphere-cloud/Keycloak-provider),
  [Valkey](https://github.com/codesphere-cloud/managed-services-valkey),
  [Postfix](https://github.com/codesphere-cloud/postfix-provider), etc.)
- Providers implement a `Provider` interface and are deployed
  as backends inside the cluster OMS provisions

## Build and test

```bash
make build-cli          # build the CLI binary
make test               # unit tests (-count=1, no caching)
make test-integration   # integration tests (build tag: integration)
make lint               # golangci-lint
make format             # go fmt
make generate           # mockery v3 + go generate
make docs               # regenerate CLI docs (hack/gendocs)
make generate-notice    # update NOTICE via go-licenses (required after dep changes)
make generate-license   # copywrite headers + go-licenses
```

## Conventions

### Language and module

- Go 1.26+; module path `github.com/codesphere-cloud/oms`.
- Prefer `url.ParseRequestURI` over `url.Parse` when the input is always an absolute URI.

### Copyright headers (required)

Every `.go` file must start with:

```go
// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0
```

Enforced by `copywrite` (`make generate-license`).

### Commit messages

Conventional Commits format: `<type>(<scope>): <description>`

| Type | Use |
| ------ | ----- |
| `feat` | new feature |
| `fix` | bug fix |
| `update` | dependency or tooling update |
| `refac` | refactoring |
| `chore` | maintenance, docs |

DCO sign-off required: `git commit -s` (adds `Signed-off-by:` trailer).
Squash-merge workflow — PR title becomes the commit message, suffixed `(#<PR>)`.

### Testing

- **Framework**: Ginkgo v2 + Gomega (BDD specs).
- **Suite file**: each package needs `*_suite_test.go` with `RegisterFailHandler`/`RunSpecs`.
- **New/changed exported functions must have tests** — file-parallel `foo.go` → `foo_test.go`.
- **Integration tests**: tagged `//go:build integration`, run via `make test-integration`.
- **Unit tests**: `make test` (uses `-count=1`).

### Mocks

- Mockery v3 (`github.com/vektra/mockery/v3`).
- Annotate interfaces with `//mockery:generate: true`.
- Run `make generate` to regenerate.

### Error handling

- Wrap with context: `fmt.Errorf("doing X: %w", err)`.
- Validate inputs before network/side-effect calls (fail fast).

### Dependencies

- Renovate bot manages updates via `renovate.json` (extends `github>codesphere-cloud/renovate-config:go-config`).
- When deps change: `go.mod` + `go.sum` + `NOTICE` + `internal/tmpl/NOTICE` must all update together.
- Run `make generate-notice` after any dependency change.
- GitHub Actions are SHA-pinned (not tag-pinned).

### Secrets and encryption

- SOPS v3 for vault encryption — no unencrypted vaults.
- Vault package: `internal/installer/vault/`.
- Never commit unencrypted secrets.

### Helm / Kubernetes

- Helm v4 (`helm.sh/helm/v4`).
- Rook/Ceph CRDs via `github.com/rook/rook/pkg/apis`.
- All `k8s.io/*` libs must stay version-synced.

## Quality gates (pre-commit checklist)

Shared rules (see `~/.agents/rules/`):

- [`clean-history.mdc`](~/.agents/rules/clean-history.mdc) — rebase, no merge/fixup/WIP commits, conventional commit format
- [`pre-push-gates.mdc`](~/.agents/rules/pre-push-gates.mdc) — branch safety, scope hygiene, error handling, tests
- [`pr-lifecycle.mdc`](~/.agents/rules/pr-lifecycle.mdc) — validation pipeline, manual approval gates

OMS-specific gates: run `make format`, `lint`, `test`, `generate` (see [Build and test](#build-and-test)),
ensure copyright headers on new `.go` files, and update `NOTICE` if deps changed.

## Review process

- 1 approval minimum from a maintainer before merge.
- CI must pass: build, test, lint, integration-test workflows.
- Squash-merge; no direct pushes to `main`.

## Upstream

- Origin: `github.com/codesphere-cloud/oms`
- When working from a fork, PRs target the fork by default unless stated otherwise.
