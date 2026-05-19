# Contributing to fluid-cli

Thank you for contributing. This project is a Go CLI built with [Cobra](https://github.com/spf13/cobra).

## Prerequisites

- Go version from [go.mod](go.mod)
- Access to a Kubernetes cluster with Fluid installed (for manual testing)
- `kubectl` configured for that cluster

## Build and test

```bash
make build          # produces bin/fluid
make test           # unit tests (race detector in CI)
make fmt vet        # format and static analysis
make install-plugin # install fluid onto PATH
```

Run a single package:

```bash
go test ./pkg/inspect/... -v -count=1
```

## Documentation

### In-terminal help (source of truth)

Command text lives in `cmd/fluid/root/`:

- `Short`, `Long`, and `Example` on each `cobra.Command`
- Flag descriptions on `cmd.Flags()`

Users rely on `fluid <command> --help`; keep examples copy-pasteable.

### Repository docs

| Path | Purpose |
|------|---------|
| [README.md](README.md) | Quickstart and links |
| [docs/](docs/) | Guides (hand-written) |
| [docs/reference/](docs/reference/) | Auto-generated; do not edit by hand |

After changing commands or flags:

```bash
make docs
git add docs/reference/
```

CI runs `make docs` and fails if `docs/reference/` is out of date.

### End-to-end testing

See [test/e2e/test.md](test/e2e/test.md) for manual e2e scenarios (when documented).

## Pull requests

- Use the PR template checklist
- Run `make test` and `make docs` before pushing
- One logical change per PR when possible
- Link related Fluid or fluid-cli issues

## Code style

- Match existing copyright headers and package layout
- `go fmt` on all touched Go files
- Prefer extending existing packages over new abstractions

## License

By contributing, you agree that your contributions are licensed under the Apache License 2.0.
