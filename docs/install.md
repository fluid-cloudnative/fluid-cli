# Installation

## From source (recommended)

```bash
git clone https://github.com/fluid-cloudnative/fluid-cli.git
cd fluid-cli
make install-plugin
fluid --help
```

`make install-plugin` builds `bin/fluid` and copies it to a directory on your `PATH` (typically next to other CLI tools, or `/usr/local/bin`).

## Manual install

```bash
make build
cp bin/fluid /usr/local/bin/fluid   # or any directory on your PATH
```

## Prerequisites

- A Kubernetes cluster with [Fluid](https://github.com/fluid-cloudnative/fluid) installed (CRDs and controllers)
- Valid kubeconfig (`kubectl` should work against the cluster)
- For interactive TUI modes: a terminal with stdin/stdout (not a pipe-only CI job)

## Verify

```bash
fluid version
fluid inspect --help
```
