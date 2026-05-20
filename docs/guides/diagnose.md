# Diagnose

`fluid diagnose` collects diagnostic artifacts for a Fluid Dataset: CR YAML, pod descriptions, logs, namespace events, PVC/PV data, and an optional tar.gz archive for support.

## When to use diagnose

- Opening a GitHub issue or vendor support ticket
- Capturing cluster state after an incident
- Sharing a timestamped bundle with your team

Use [`fluid inspect`](inspect.md) for a lightweight resource listing without log collection.

## Basic usage

```bash
# Collect into a timestamped directory (default TUI to browse results)
fluid diagnose my-dataset -n default

# Write artifacts only (no TUI)
fluid diagnose my-dataset -n default -o dir

# Create a tar.gz archive for attachment
fluid diagnose my-dataset -n default --archive -o dir
```

## Output layout

Artifacts are written under a directory such as `fluid-diagnose-<dataset>-<timestamp>/`:

| Path | Contents |
|------|----------|
| `dataset.yaml` | Dataset CR |
| `dataset.describe.txt` | Human-readable Dataset summary |
| `runtime/` | Runtime CRs and descriptions |
| `pods/` | Per-pod YAML, describe, and logs |
| `events/` | Namespace events |
| `storage/` | PVC and PV YAML |
| `controllers/` | Fluid controller logs (if `--include-controller-logs`) |
| `summary.txt` | High-level summary and warnings |
| `manifest.json` | Index of every artifact and collection status |
| `context.json` | Structured diagnostic context for AI-assisted analysis |
| `prompt.txt` | Prompt-ready text derived from `context.json` |
| `llm-analysis.txt` | LLM diagnosis result (when analysis is enabled) |

`manifest.json` records each file with `status` (`collected`, `failed`, `skipped`) and an optional `reason`. Partial failures are normal when pods are missing or logs are unavailable; check `summary.txt` and the manifest.

With `--archive`, a `.tar.gz` is produced alongside the directory (see command output for the path).

## Support workflow

1. Run diagnose with `--archive` and `-o dir` in non-interactive environments.
2. Redact secrets from the bundle if needed before sharing.
3. Attach the archive to your issue along with `fluid version` output.

## AI-assisted diagnosis

`fluid diagnose` builds structured context and can call an **OpenAI-compatible** chat completions API (`POST /v1/chat/completions`) for automated analysis.

### Configure LLM settings

Settings are stored in `~/.fluid/config` (prefer environment variables for secrets):

```bash
fluid diagnose config set llm-endpoint https://api.openai.com/v1
fluid diagnose config set llm-model gpt-4o-mini
export FLUID_LLM_API_KEY=sk-...

fluid diagnose config view
```

| Setting | Flag | Environment | Config file |
|---------|------|-------------|-------------|
| API base URL | `--llm-endpoint` | `FLUID_LLM_ENDPOINT` | `diagnose.llm.endpoint` |
| API key | — | `FLUID_LLM_API_KEY` | `diagnose.llm.apiKey` |
| Model | `--llm-model` | `FLUID_LLM_MODEL` | `diagnose.llm.model` |

When an endpoint is configured, LLM analysis runs by default. Use `--llm-skip` to write `context.json` and `prompt.txt` only.

### Run with LLM analysis

```bash
fluid diagnose my-dataset -n default -o dir
```

Outputs:

- `context.json` — trimmed JSON (dataset, runtimes, pods, warning events, summary)
- `prompt.txt` — diagnosis-focused prompt (no remediation commands in the default template)
- `llm-analysis.txt` — model response when analysis is enabled
- `--prompt-file` — optional extra copy of the prompt

Compatible with OpenAI, Azure OpenAI, and other OpenAI-compatible gateways (local proxies, vLLM, etc.).

## Flags and help

```bash
fluid diagnose --help
```

Notable options include `--output-dir`, `--no-logs`, `--include-controller-logs`, `--since` (limit log/event age), `--prompt-file`, `--llm-endpoint`, `--llm-model`, and `--llm-skip`. Defaults and full descriptions are in `--help`.
