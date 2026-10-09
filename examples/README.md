# glutton examples

Runnable examples for the features the README describes. Each script builds glutton, starts it on a predictable port, exercises it, checks the result and shuts the server down again (even when something fails).

```bash
# run a single example
bash examples/scripts/01-basic.sh

# or run them all in order
bash examples/scripts/run-all.sh
```

## Requirements

* Go (see `go.mod` for the version) and `curl` for every example
* `openssl` (1.1.1 or newer) for the provided certificate example
* `psql` and a reachable Postgres for the database example

## What each example shows

| Script | Port | Demonstrates | Extra requirements |
| --- | --- | --- | --- |
| `01-basic.sh` | 4354 | A route with the filesystem saver; the format of a stored payload | - |
| `02-token.sh` | 4355 | Token protected route: `token-key` header to fetch a token, `token` header to save, `412` when missing | - |
| `03-https-self-signed.sh` | 4356 | HTTPS with a certificate generated in memory at startup | - |
| `04-https-certificate.sh` | 4357 | HTTPS with your own certificate, supplied through `CERT_FILE`/`KEY_FILE` | `openssl` |
| `05-postgres.sh` | 4358 | Storing payloads in Postgres instead of a file | `psql`, Postgres |

Examples that miss a requirement print `skip` and exit successfully, so `run-all.sh` works everywhere.

## Where things end up

* Payloads are written to `/tmp/glutton-examples/<example>` (override the base with `GLUTTON_EXAMPLES_DIR`).
* The binary is built once into `/tmp/glutton-examples/glutton` (override with `GLUTTON_BINARY`) and reused by later examples.
* Server output is captured in `/tmp/glutton-examples/glutton.log`; the scripts print it when startup fails.

## Configuration

The `configs/` directory holds the yaml files the scripts use, and `sql/schema.sql` is the table the default `DatabaseSaver` layout expects. Three ways of configuring a route are on display:

* `configs/basic.yaml` and friends - everything from a file.
* `04-https-certificate.sh` - `CERT_FILE`/`KEY_FILE` come from the environment, which takes precedence over the yaml file.
* `05-postgres.sh` - generates a config on the fly, because route settings (`sql_connection_string`) can only come from yaml once a yaml file defines routes.

Point the database example at another server with:

```bash
GLUTTON_POSTGRES_DSN='postgres://user:pass@host:5432/db?sslmode=disable' bash examples/scripts/05-postgres.sh
```

## Adding an example

1. Put the configuration in `configs/`.
2. Source `scripts/lib.sh`, use `build_glutton`, `start_glutton`, `wait_for_glutton`, `expect_status` and `fail`.
3. Name it `NN-name.sh` so `run-all.sh` picks it up.

The helpers are small on purpose - read [lib.sh](./scripts/lib.sh) before adding anything.
