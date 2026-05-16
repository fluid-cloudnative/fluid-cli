# Troubleshooting

## Fluid CRDs not installed

**Symptom:** Errors mentioning `no matches for kind`, `no kind is registered`, or `data.fluid.io/v1alpha1 not found`.

**Fix:** Install Fluid on the cluster before using `fluid`. See the [Fluid install guide](https://github.com/fluid-cloudnative/fluid/blob/master/docs/en/userguide/install.md).

```bash
kubectl get crd datasets.data.fluid.io
```

## Cannot reach the API server

**Symptom:** Timeouts, connection refused, or TLS errors when running `fluid inspect` or `fluid diagnose`.

**Fix:** Verify kubeconfig and cluster health:

```bash
kubectl cluster-info
kubectl get ns
```

Ensure `--context` and `--kubeconfig` point at the intended cluster.

## Interactive mode in a non-TTY environment

**Symptom:** Errors about non-interactive mode when using default `-o tui` or omitting the dataset name on `inspect`.

**Fix:** Pass explicit arguments and a non-TUI output mode:

```bash
fluid inspect my-dataset -n default -o table
fluid diagnose my-dataset -n default -o dir
```

## RBAC permission denied

**Symptom:** `Forbidden` or `cannot get resource` when listing or collecting resources.

**Fix:** Your kubeconfig user or ServiceAccount needs read access to Fluid CRDs, Pods, Events, PVCs, PVs, and related objects in the target namespace (and `fluid-system` if using `--include-controller-logs`).

## Diagnose partial collection

**Symptom:** Message like `completed with N partial collection issues`.

**Fix:** Open `summary.txt` and `manifest.json` in the output directory. Failed entries include a `reason` (e.g. pod not found, log stream error). The bundle is still useful; re-run with `--since` or `--no-logs` on very large clusters if collection is slow.

## Stale documentation in the repo

Generated command reference lives in `docs/reference/`. After changing commands or flags in `cmd/`, run:

```bash
make docs
```

and commit the updated files. CI verifies they match the Cobra tree.

## Getting help

- Command help: `fluid <command> --help`
- [Inspect guide](guides/inspect.md) / [Diagnose guide](guides/diagnose.md)
- [GitHub issues](https://github.com/fluid-cloudnative/fluid-cli/issues) — include `fluid version`, exact command, and redacted output
