# fluid CLI documentation

`fluid` is the CLI for the [Fluid](https://github.com/fluid-cloudnative/fluid) project. It helps you inspect and collect diagnostics for Fluid-managed Datasets.

## Guides

- [Installation](install.md)
- [Inspect](guides/inspect.md) — list resources and status for a Dataset
- [Diagnose](guides/diagnose.md) — collect support bundles
- [Troubleshooting](troubleshooting.md)

## Reference

Command and flag reference is generated from the Cobra command tree:

- [Reference index](reference/fluid.md)
- Regenerate after changing commands: `make docs`

For the latest flags at your installed version, prefer `fluid <command> --help`.

## Roadmap

| Phase | Focus | Status |
|-------|-------|--------|
| 0 | CLI skeleton, build, install | Done |
| 1 | `inspect` subcommand | Done |
| 2 | `diagnose` subcommand | Done |
| 3 | AI/LLM-ready diagnostic format | Planned |
