[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
![Build Status](https://github.com/codesphere-cloud/oms/actions/workflows/cli-build_test.yml/badge.svg)
![Integration Test Status](https://github.com/codesphere-cloud/oms/actions/workflows/integration-test.yml/badge.svg)

# Operations Management System - OMS

This repository contains the source for the operations management system. It
contains the sources for both the CLI and the Service. 

## OMS CLI

The OMS CLI tool is used to bootstrap Codesphere cluster on customer sites and
replaces the formerly used private cloud installer.

### Installation

You can install the OMS CLI in a few ways:

#### Using GitHub CLI (`gh`)

If you have the [GitHub CLI](https://cli.github.com/) installed, you can install the OMS CLI with a command like the following.
Note that some commands may require you to elevate to the root user with `sudo`.

##### ARM Mac

```
gh release download -R codesphere-cloud/oms -O /usr/local/bin/oms -p "oms*darwin_arm64"
chmod +x /usr/local/bin/oms
```

##### Linux Amd64

```
gh release download -R codesphere-cloud/oms -O /usr/local/bin/oms -p "oms*linux_amd64"
chmod +x /usr/local/bin/oms
```

#### Using `wget`

This option requires to have the `wget` and `jq` utils installed. Download the OMS CLI and add permissions to run it with the following commands:
Note that some commands may require you to elevate to the root user with `sudo`.

##### ARM Mac

```
wget -qO- 'https://api.github.com/repos/codesphere-cloud/oms/releases/latest' | jq -r '.assets[] | select(.name | match("oms.*darwin_arm64")) | .browser_download_url' | xargs wget -O oms
mv oms /usr/local/bin/oms
chmod +x /usr/local/bin/oms
```

##### Linux Amd64

```
wget -qO- 'https://api.github.com/repos/codesphere-cloud/oms/releases/latest' | jq -r '.assets[] | select(.name | match("oms.*linux_amd64")) | .browser_download_url' | xargs wget -O oms
mv oms /usr/local/bin/oms
chmod +x /usr/local/bin/oms
```

#### Manual Download

You can also download the pre-compiled binaries from the [OMS Releases page](https://github.com/codesphere-cloud/oms/releases).
Note that some commands may require you to elevate to the root user with `sudo`.

1. Go to the [latest release](https://github.com/codesphere-cloud/oms/releases/latest).

2. Download the appropriate release for your operating system and architecture (e.g., `oms_darwin_amd64` for macOS, `oms_linux_amd64` for Linux, or `oms_windows_amd64` for Windows).

3. Move the `oms` binary to a directory in your system's `PATH` (e.g., `/usr/local/bin` on Linux/Mac, or a directory added to `Path` environment variable on Windows).

4. Make the binary executable (e.g. by running `chmod +x /usr/local/bin/oms` on Mac or Linux)

#### Available Commands

The OMS CLI organizes its functionality into several top-level commands, each with specific subcommands and flags.

See our [Usage Documentation](docs) for usage information about the specific subcommands.

### How to Build?

```shell
make build-cli
```

### macOS arm64 local bootstrap

`oms beta bootstrap-local` is viable on macOS arm64 when the CLI runs on your
Mac and the Kubernetes cluster runs inside a Linux VM-backed environment. The
cluster still needs Linux worker nodes and storage suitable for Rook/Ceph; do
not use a host-native macOS cluster for this flow.

Install the Homebrew prerequisites listed in
`hack/bootstrap-local-macos-requirements.txt`, for example:

```shell
brew install age helm kubectl lima lima-additional-guestagents node sops
```

`lima-additional-guestagents` is required on Homebrew-based macOS setups so
Lima can boot a Linux `x86_64` guest. Without it, `make run-lima-on-macos`
fails with `guest agent binary could not be found for Linux-x86_64`.

Recommended first-run sequence:

```shell
make run-lima-on-macos
make install-k0s-in-lima

OMS_REGISTRY_USER='<github-user-or-service-account>' \
OMS_INSTALL_LOCAL='/absolute/path/to/installer-lite.tar.gz' \
make bootstrap-local-on-macos
```

The Makefile flow does the following:

- `make run-lima-on-macos` starts the Linux VM.
- If the first Lima start created the instance and then failed, rerunning
  `make run-lima-on-macos` now resumes the existing `lima-oms` instance instead
  of failing with `instance "lima-oms" already exists`.
- `make install-k0s-in-lima` installs a single-node k0s cluster in that VM and
  writes host kubeconfig to `~/.kube/lima-oms.yaml`, rewritten to use
  `https://127.0.0.1:6443` via Lima port forwarding.
- `make bootstrap-local-on-macos` runs `oms beta bootstrap-local --k0s` from
  macOS against that kubeconfig.

If you already created a `lima-oms` instance before the `6443` port forward was
added, recreate it once so the host kubeconfig can reach the API server:

```shell
make stop-lima
make run-lima-on-macos
```

Registry password handling for first-time contributors:

- OMS does not generate or derive the registry password locally.
- The local flow expects credentials for `ghcr.io` with access to the private
  `codesphere-cloud` packages used by the installer.
- Pass the username via `OMS_REGISTRY_USER`.
- Provide the password/token via `OMS_REGISTRY_PASSWORD`, or leave that unset
  and enter it when `oms` prompts interactively.

Installer bundle handling:

- If you already have an installer bundle, set
  `OMS_INSTALL_LOCAL=/absolute/path/to/installer-lite.tar.gz`.
- If you want OMS to download it from the portal instead, set
  `OMS_INSTALL_VERSION`, `OMS_INSTALL_HASH`, and `OMS_PORTAL_API_KEY` before
  running `make bootstrap-local-on-macos`.

If CIDR auto-discovery does not work in your cluster, pass the flags through
`OMS_BOOTSTRAP_LOCAL_ARGS`:

```shell
OMS_BOOTSTRAP_LOCAL_ARGS='--service-cidr 10.96.0.0/12 --pod-cidr 10.244.0.0/16' \
OMS_REGISTRY_USER='<github-user-or-service-account>' \
OMS_INSTALL_LOCAL='/absolute/path/to/installer-lite.tar.gz' \
make bootstrap-local-on-macos
```

Practical notes:

- If your local cluster does not provide `LoadBalancer` services, install a
  local load balancer such as MetalLB inside the Linux VM-backed cluster.
- The default `hack/lima-oms.yaml` VM does not attach a dedicated extra block
  device for Rook/Ceph yet. On a clean Lima VM, `make bootstrap-local-on-macos`
  currently gets through k0s and into the Rook bootstrap, but CephFS stays
  blocked until you provide a Ceph-suitable disk to the guest.
- `make install-k0s-in-lima` assumes the repo checkout lives under your macOS
  home directory so Lima can access it through the default host-home mount.

See also [CONTRIBUTION.md]

## Service

The service implementation is currently WIP

### How to Build?

```shell
make build-service
```

## Community & Contributions

Please review our [Code of Conduct](CODE_OF_CONDUCT.md) to understand our community expectations.
We welcome contributions! All contributions to this project must be made in accordance with the Developer Certificate of Origin (DCO). See our full [Contributing Guidelines](CONTRIBUTING.md) for details.
