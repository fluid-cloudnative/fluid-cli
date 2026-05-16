# Inspect

`fluid inspect` lists Kubernetes resources associated with a Fluid Dataset and its Runtime(s): Pods, StatefulSets, DaemonSets, PVCs, PVs, Services, and related status.

## When to use inspect

- Quick health check of a Dataset without collecting logs
- Scripted or CI output (`-o json` / `-o yaml`)
- Interactive exploration in the terminal TUI

For support bundles (logs, events, archives), use [`fluid diagnose`](diagnose.md) instead.

## Basic usage

```bash
# Inspect a named Dataset (uses current kubeconfig namespace or default)
fluid inspect my-dataset

# Explicit namespace
fluid inspect my-dataset -n production

# Table output (non-interactive)
fluid inspect my-dataset -n default -o table

# JSON for automation
fluid inspect my-dataset -n default -o json
```

## Interactive dataset picker

Omit the dataset name to open a TUI that lists Datasets in the namespace:

```bash
fluid inspect -n default
```

This requires an interactive terminal. In scripts or CI, always pass the dataset name and use `-o table`, `json`, or `yaml`.

## Output modes

| `-o` value | Description |
|------------|-------------|
| `tui` (default) | Full-screen terminal UI with tabs |
| `table` | Columnar text table |
| `json` | JSON document |
| `yaml` | YAML document |

Use `--wide` with `table`, `json`, or `yaml` for extra columns (e.g. node, restart count).

## Flags and help

```bash
fluid inspect --help
```

Global Kubernetes flags (`--kubeconfig`, `--context`, `-n`) apply via the root command.
