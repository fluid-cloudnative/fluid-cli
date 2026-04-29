# fluid CLI

`fluid` is a kubectl plugin for the [Fluid](https://github.com/fluid-cloudnative/fluid) project.
It provides commands to inspect and diagnose Fluid-managed Datasets and their associated Kubernetes resources.

---

## Installation

### From source

```bash
# Clone the repository
git clone https://github.com/fluid-cloudnative/fluid-cli.git
cd fluid-cli

# Build and install into the same directory as kubectl
make install-plugin

# Verify
fluid --help
```

### Manual install

```bash
make build
cp bin/fluid /usr/local/bin/fluid
```

kubectl discovers any binary named `kubectl-<name>` in `PATH` and exposes it as `kubectl <name>`.

---

## Usage

```text
fluid [command]

Inspect:
  inspect     List all Kubernetes resources associated with a Fluid Dataset

Diagnose:
  diagnose    Collect diagnostic data for a Fluid Dataset and its Runtime(s)

Utility:
  version     Print the version of fluid

Global Flags:
  --kubeconfig   Path to kubeconfig file
  --context      Kubeconfig context to use
  -n, --namespace   Namespace scope
```

### inspect

> **Phase 1 — available.**

```bash
fluid inspect <dataset-name> [-n namespace] [-o table|json|yaml] [--wide]
```

List all Kubernetes resources (Pods, StatefulSets, DaemonSets, PVCs, PVs, Services)
owned by a given Fluid Dataset and its Runtime(s).

### diagnose

> **Phase 2 — available.**

```bash
fluid diagnose <dataset-name> [-n namespace] [flags]

Flags:
  --archive                  Package artifacts into a tar.gz archive
  --output-dir string        Directory to write artifacts
  --no-logs                  Skip collecting pod logs
  --include-controller-logs  Also collect Fluid controller logs
  --since duration           Only collect logs/events newer than this (e.g. 1h)
```

Collect diagnostic data (Dataset/Runtime YAML, pod logs, events, PVCs/PVs)
and optionally package them into a tar.gz archive for support.

### version

```bash
fluid version
```

---

## Roadmap

| Phase | Focus | Status |
|-------|-------|--------|
| 0 | Plugin skeleton, build, install | Done |
| 1 | `inspect` subcommand | Done |
| 2 | `diagnose` subcommand | Done |
| 3 | AI/LLM-ready diagnostic format | Planned |
| 4 | Polish, docs, release | Planned |

---

## Development

```bash
# Build
make build

# Run tests
make test

# Cross-compile for all platforms
make build-all

# Format and vet
make fmt vet
```

The module depends on published Fluid releases in `go.mod`.
