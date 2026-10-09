---
name: glutton-configure
description: 'Write or review glutton configuration in yaml or environment variables. Use when adding a route, choosing a parser, saver or notifier, enabling token auth or TLS, storing payloads in Postgres, or working out which source a setting actually came from.'
---

# Configuring glutton

## When to use

- add or change a route
- pick components, enable tokens/TLS, store payloads in a database
- work out why a setting did not take effect

## Precedence and defaults

Applied in this order, later wins: **defaults** (the `default` struct tag) → **yaml file** → **environment variables**.

Top level keys: `debug`, `host` (default `0.0.0.0`), `port` (default `4354`), `use_tls`, `cert_file`, `key_file`.

Route settings live under `settings:`. Environment variables only configure a route when the yaml file defines **no** routes, in which case a single route is built from `NAME`, `URI`, `REDIRECT`, `PARSER`, `NOTIFIER`, `SAVER`, `OUTPUT_FOLDER`, `BASE_NAME`, `USE_TOKEN`, `TOKEN_KEY`, `SQL_*`, `SMTP_*`, `MAX_BODY_SIZE`.

## Route settings

`name`, `uri` (the path below `/v1/glutton/`), `redirect` (302 target, optional), `parser`, `notifier`, `saver`, plus the component settings below. Unset string settings fall back to defaults, so `uri`, `parser`, `notifier`, `saver`, `output_folder` and `base_name` can usually be omitted.

| Component | Choices | Notes |
| --- | --- | --- |
| `parser` | `SimpleParser` | reads the body, bounded by `max_body_size` (default 1 MiB, `0` = unlimited, over the limit → `413`) |
| `notifier` | `NilNotifier`, `SMTPNotifier` | `SMTPNotifier` needs `smtp_server`, `smtp_port`, `smtp_from`, `smtp_to`, `smtp_password`; `smtp_use_tls: true` requires STARTTLS support or the notification fails |
| `saver` | `SimpleFileSystemSaver`, `DatabaseSaver` | filesystem uses `output_folder` (created automatically) and `base_name` (`%d` counter) |

## Token auth

```yaml
use_token: true
token_key: 0123456789abcdef   # 16, 24 or 32 bytes, else startup aborts
```

Callers fetch a token from `GET /v1/glutton/<uri>/token` with the key in the `token-key` header, then save with the token in the `token` header. Tokens last five minutes.

## TLS

`use_tls: true` plus `cert_file` and `key_file` (both required) serves a provided PEM pair. With neither file set, a self-signed certificate is generated in memory for `localhost`, `127.0.0.1`, `::1` and the configured `host`. TLS 1.2 is the minimum. Examples: `examples/configs/tls-self-signed.yaml`, `examples/configs/tls-certificate.yaml`.

## Postgres

```yaml
saver: DatabaseSaver
sql_driver: postgres
sql_connection_string: postgres://user:pass@host:5432/db?sslmode=disable
```

`sql_layout` defaults to `INSERT INTO payload(ts, remote, meta, payload) VALUES ($1, $2, $3, $4)` where `meta` is the request metadata serialized as json. The table is **not** created automatically — use `examples/sql/schema.sql`. Startup pings the database and refuses to start when it is unreachable.

## Checklist before starting

- a unique `uri` per route
- token key length 16/24/32 whenever `use_token` is on
- `cert_file` and `key_file` set together, or neither
- output folder writable / database reachable and migrated

Working files to copy: `examples/configs/`. For running and verifying an instance, see the `glutton-run` skill.
