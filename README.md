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
| `fluid diagnose config` | Manage AI/LLM settings for diagnose (`~/.fluid/config`) |
| `fluid version` | Print CLI version |

For flags and examples, use `--help` on any command:

```bash
fluid inspect --help
fluid diagnose --help
```

## AI-assisted diagnosis

`fluid diagnose` can call an OpenAI-compatible LLM API to analyze collected cluster context, or export prompt files for manual copy/paste.

```bash
fluid diagnose config set llm-endpoint https://api.openai.com/v1
export FLUID_LLM_API_KEY=sk-...

fluid diagnose my-dataset -n default -o dir
```

Artifact directory includes `context.json`, `prompt.txt`, and `llm-analysis.txt` when LLM analysis runs. Use `--llm-skip` to collect prompts only without calling the API.

See [Diagnose guide](docs/guides/diagnose.md) for details.

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
