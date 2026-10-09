# glutton

[![Build Status](https://travis-ci.org/defectus/glutton.svg?branch=master)](https://travis-ci.org/defectus/glutton)
[![GoDoc](https://godoc.org/github.com/defectus/glutton/pkg?status.svg)](https://godoc.org/github.com/defectus/glutton/pkg)
[![Coverage status](https://codecov.io/github/defectus/glutton/coverage.svg?branch=master)](https://codecov.io/github/defectus/glutton?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/defectus/glutton)](https://goreportcard.com/report/github.com/defectus/glutton)



Glutton is a small HTTP server that can be called with *ANY* data and the data is stored.

## How to run glutton

* from source
   * run `make run` - this will spin up glutton on local port 4354
* the bundled examples
   * run `bash examples/scripts/run-all.sh` - builds glutton and walks through the filesystem saver, token auth, HTTPS and Postgres setups, starting and stopping the server for you
* docker
   * run `docker --rm -it -p 4354:4354 -v glutton:glutton defectus/glutton` - this will spin up glutton on local port 4354
   * more meaningful command would look like `run -d --name glutton --restart=always --log-driver=syslog --log-opt tag=glutton --env-file /etc/glutton/glutton.env -v /var/glutton/:/out/ -p 8888:8080 defectus/glutton:latest`. 

## Getting started quickly

* [examples/](examples/README.md) holds runnable scripts and the configs they use - one per feature, each building the binary, starting the server on a fixed port, exercising it and shutting it down again.
* [.claude/skills/](.claude/skills/) holds Claude Skills (`glutton-run`, `glutton-configure`) so an agent can start, verify and configure an instance without reading the source first.

## Configuration

First, command line arguments. At the moment two:

* -f *file.yaml* : use yaml file to configure application
* -d : enable debug messages

Configuration lives either in OS environment, or is provided as a yaml file. Yaml offers greater variablity and more importantly allows you to define more than one route.

Basic structure of the yaml file:
```yaml
debug: true
port: 8080
host: 0.0.0.0
use_tls: false # serve over https; without cert_file/key_file a self signed certificate is generated
cert_file: # optional path to a certificate, requires key_file
key_file: # optional path to the matching private key, requires cert_file
settings:
  - name: default glutton route
    redirect: some_url
    uri: save
    parser: SimpleParser # choice of `SimpleParser`
    notifier: NilNotifier # choice of `NilNotifier`, `SMTPNotifier`
    saver: SimpleFileSystemSaver # choice of `SimpleFileSystemSaver`, `DatabaseSaver`
    # SimpleFileSystemSaver settings
    output_folder: glutton # location to which request are saved
    base_name: glutton_%d # name of request files (supports single numeric counter variable)
    # SMTPNotifier settings
    smtp_server: smtp.gmail.com
    smtp_port: 25 # for gmail use 587
    smtp_use_tls: true # gmail requires TLS
    smtp_from: your@email.address
    smtp_to: target@email.address
    smtp_password:  # for gmail, configure your account to allow unsecured connection
    token_key: 0123456789abcdef # a key to use to encrypt access tokens, if enabled; must be 16, 24 or 32 bytes
    use_token: false 
    sql_driver: postgres # if configured to use the `DatabaseSaver`
    sql_layout: "INSERT INTO payload(ts, remote, meta, payload) VALUES ($1, $2, $3, $4)" # $1 is the timestamp, $2 is the remote host, $3 is the meta data serialized as json and $4 is the payload
    sql_connection_string: "postgres://user:password@localhost:5432/glutton?sslmode=disable"
```

As you can see, the settings is fairly straight forward. When using the environment keys are:
* `DEBUG`
* `HOST`
* `PORT`
* `USE_TLS`
* `CERT_FILE`
* `KEY_FILE`
* `NAME`
* `URI`
* `REDIRECT`
* `PARSER`
* `NOTIFIER`
* `SAVER`

SimpleFileSystemSaver settings

* `OUTPUT_FOLDER`
* `BASE_NAME`

DatabaseSaver settings

* `SQL_DRIVER` - the database driver to use; `postgres` is bundled
* `SQL_LAYOUT` - the insert statement; `$1` is the timestamp, `$2` the remote host, `$3` the meta data as json and `$4` the payload
* `SQL_CONNECTION_STRING` - the driver specific connection string, e.g. `postgres://user:password@localhost:5432/glutton?sslmode=disable`

The table referenced by `SQL_LAYOUT` is not created automatically. For the default layout create it with:

```sql
CREATE TABLE payload (
    ts      timestamptz NOT NULL,
    remote  text        NOT NULL,
    meta    jsonb       NOT NULL,
    payload text        NOT NULL
);
```

SMTPNotifier settings

* `SMTP_SERVER`
* `SMTP_PORT`
* `SMTP_USE_TLS`
* `SMTP_FROM`
* `SMTP_TO`
* `SMTP_PASSWORD`

When `SMTP_USE_TLS` is enabled the SMTP server must support STARTTLS, otherwise the notification is not sent — the connection is never silently downgraded to plain text.

Token settings

* `USE_TOKEN`
* `TOKEN_KEY` - the key used to sign access tokens. It must be 16, 24 or 32 bytes long; the application refuses to start otherwise.

Please note that only one route can be defined with environment variables.

## Endpoint

A sample request can be found in the http/save-basic.http file. Effectively you have to do HTTP `POST` on `/v1/glutton/save`. As the payload is in no paricular format any payload will do.

## Access tokens

If `use_token` is enabled for a route, saving requires a valid `token` header. A token is obtained from `GET /v1/glutton/<uri>/token` and must be presented to that endpoint in the `token-key` header, set to the configured `TOKEN_KEY` (so only callers that know the key can mint a token):

```
curl -H 'token-key: 0123456789abcdef' http://localhost:4354/v1/glutton/save/token
```

Tokens are valid for five minutes.

## TLS

Set `use_tls` (or `USE_TLS=true`) to serve every route over https. Two modes are supported:

* **Provided certificate** - set `cert_file` and `key_file` to a PEM encoded certificate and its private key. Both are required; the application refuses to start when only one is given or the files cannot be read.
* **Self signed certificate** - leave `cert_file`/`key_file` unset and glutton generates a certificate in memory at startup (valid for `localhost`, `127.0.0.1`, `::1` and the configured `host`). Handy for local testing, but clients have to skip verification.

```yaml
use_tls: true
cert_file: /etc/glutton/cert.pem
key_file: /etc/glutton/key.pem
```

## Output

Requests are stored on a path defined by the `OUTPUT_FOLDER` variable. If ommited it defaults to `glutton`.

## Future

In the future releases you hopefully find the following features

✔️️️ saving to database

✔️ redirect on save 

️️✔️ auth tokens (allow saving with a valid token only)

✔️ serving over https (self signed or provided certificate)


above all, keep this project low profile, I'm not building an application server here. glutton must be simple, stupid.
