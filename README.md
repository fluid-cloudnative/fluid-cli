# fluid-cli

Official CLI for [Fluid](https://github.com/fluid-cloudnative/fluid) — inspect and diagnose Fluid-managed Datasets and their Kubernetes resources.

Install the `fluid` binary on your `PATH`.

## Quick start

```bash
git clone https://github.com/fluid-cloudnative/fluid-cli.git
cd fluid-cli
make install-plugin

fluid --help
fluid inspect my-dataset -n default
fluid diagnose my-dataset -n default --archive
```

## Commands

| Command | Description |
|---------|-------------|
| `fluid inspect` | List Pods, Runtimes, PVCs, and related resources for a Dataset |
| `fluid diagnose` | Collect a support bundle (YAML, logs, events) for a Dataset |
| `fluid version` | Print CLI version |

For flags and examples, use `--help` on any command:

```bash
fluid inspect --help
fluid diagnose --help
```

## Documentation

- [Install](docs/install.md)
- [Inspect guide](docs/guides/inspect.md)
- [Diagnose guide](docs/guides/diagnose.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Command reference](docs/reference/) (auto-generated from Cobra; run `make docs` to refresh)

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md).

```bash
make build    # build bin/fluid
make test     # unit tests
make docs     # regenerate docs/reference/
```

## License

Apache License 2.0 — see [LICENSE](LICENSE).
