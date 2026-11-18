# CLAUDE.md — glutton

Go service that receives page-view/event HTTP calls and writes them to storage.
The [README](README.md) explains the idea and how it is wired; `.claude/skills/`
holds two runnable skills (`glutton-configure`, `glutton-run`) — read them before
inventing a config or a local run.

## Layout

| Path | What |
|---|---|
| `cmd/glutton/` | `main`: flags, wires the pieces |
| `http/` | the HTTP surface (routes, handlers' glue) |
| `pkg/handler/`, `pkg/parser/`, `pkg/saver/`, `pkg/auth/`, `pkg/notifier/`, `pkg/common/`, `pkg/iface/` | the pipeline: request → parse → save → notify |
| `definition/` | configuration and the data-shape definitions |
| `docker/`, `examples/`, `sample/`, `test/` | local stacks and fixtures |
| `makefile`, `magefile.go` | the two entry points for building and testing |

The module path is `github.com/defectus/glutton` — the old username. Do not
"fix" imports to `ondrej-li`: the repository is `defectus` upstream and the
import path is part of the public API of the package.

## Build, test, run

CI (`.github/workflows/ci.yml`, `ubuntu-latest`) runs `go build ./...`,
`go vet ./...`, `go test ./...` and `go test -race ./...` on every push and PR.
Those four commands are the definition of "working":

```sh
go build ./... && go vet ./... && go test ./...
```

Locally, `mage` and `make` both work: `make run` starts glutton on port 4354,
`make test` / `make vet` wrap the same commands and write reports. Nothing here
needs a Mac or Linux-specific build tag.

## Deploy

There is no deploy. The binary is built where it runs. GitHub Pages does build
this repository (`legacy master:/`, Jekyll with `_config.yml`), which publishes the
README as a page — it does not ship the service anywhere.

## Conventions

- Everything the process touches is an interface in `pkg/iface` so that the
  pipeline can be tested without a database or a network: keep new dependencies
  behind an interface there.
- `makefile` and `magefile.go` are both supported entry points; add a new task to
  both or to neither.
- The race detector is part of CI for a reason — the saver and notifier are
  concurrent. Do not add shared state without running `go test -race ./...`.
