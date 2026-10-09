---
name: glutton-run
description: 'Run, smoke-test and debug a local glutton instance. Use when starting glutton, verifying that a payload was really stored, checking route status codes, or investigating why a request did not land in the output folder or database.'
---

# Running and smoke-testing glutton

## When to use

- bring up glutton locally to try a configuration
- confirm a payload actually reached the filesystem or the database
- reproduce a status code, routing or TLS problem

## Procedure

1. Build: `go build -o /tmp/glutton ./cmd/glutton`
2. Start it, either from a config file or purely from the environment:
   - `/tmp/glutton -f examples/configs/basic.yaml`
   - `PORT=4354 /tmp/glutton -d` (no config file: one route built from `NAME`/`URI`/`SAVER`/…)
3. Wait for `listening on <host>:<port>` in the output (`… over https` when TLS is on).
4. Send a payload:
   `curl -i -X POST http://127.0.0.1:4354/v1/glutton/save -d '{"hello":"world"}'`
5. Verify the `200` **and** the stored payload: a file below `OUTPUT_FOLDER` (default `glutton/`, created automatically) or a row in Postgres.
6. Stop it with `kill -INT <pid>` for a graceful shutdown (SIGKILL/`os.Kill` is not handled).

Scripts that do exactly this, including starting and cleaning up the server: `examples/scripts/01-basic.sh` through `05-postgres.sh`, or all of them with `examples/scripts/run-all.sh`. The helpers they share are in `examples/scripts/lib.sh`.

## Status codes worth knowing

| Situation | Code |
| --- | --- |
| payload stored | `200` |
| `redirect` configured and stored | `302` + `Location` |
| body larger than `max_body_size` | `413` |
| `use_token` route without a valid `token` header | `412` |
| token endpoint without a valid `token-key` header | `412` |
| parse error | `400` |
| notifying or saving failed | `500` |

## Gotchas

- **Configuration precedence** is defaults → yaml → environment, but route settings only come from environment variables when the yaml file defines no routes at all.
- **Tokens**: `use_token: true` requires a `TOKEN_KEY` of 16, 24 or 32 bytes, otherwise startup aborts. Fetch a token with `curl -H 'token-key: <key>' <url>/token`, then save with `-H 'token: <token>'`.
- **TLS**: with `use_tls: true` and no `cert_file`/`key_file`, a self-signed certificate is generated for `localhost`, `127.0.0.1`, `::1` and the configured host, so clients need `curl -k`.
- **A `500` means the payload was not stored** — the reason is in the log, and the response body is empty, so read the log rather than guessing.
