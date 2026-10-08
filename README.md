# glutton

[![Build Status](https://travis-ci.org/defectus/glutton.svg?branch=master)](https://travis-ci.org/defectus/glutton)
[![GoDoc](https://godoc.org/github.com/defectus/glutton/pkg?status.svg)](https://godoc.org/github.com/defectus/glutton/pkg)
[![Coverage status](https://codecov.io/github/defectus/glutton/coverage.svg?branch=master)](https://codecov.io/github/defectus/glutton?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/defectus/glutton)](https://goreportcard.com/report/github.com/defectus/glutton)



Glutton is a small HTTP server that can be called with *ANY* data and the data is stored.

## How to run glutton

* from source
   * run `make run` - this will spin up glutton on local port 4354
* docker
   * run `docker --rm -it -p 4354:4354 -v glutton:glutton defectus/glutton` - this will spin up glutton on local port 4354
   * more meaningful command would look like `run -d --name glutton --restart=always --log-driver=syslog --log-opt tag=glutton --env-file /etc/glutton/glutton.env -v /var/glutton/:/out/ -p 8888:8080 defectus/glutton:latest`. 

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
    sql_layout: "INSERT INTO payload(ts, remote, meta, payload) VALUES ($1, $2, $3, $4)" # $1 is the timestamp, $2 is the remote host, $3 is meta data map and $4 is the payload
    sql_connection_string: "postgres://root:root@localhost/postgres?sslmode=disable"
```

As you can see, the settings is fairly straight forward. When using the environment keys are:
* `DEBUG`
* `HOST`
* `PORT`
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

* `SQL_DRIVER`
* `SQL_LAYOUT`
* `SQL_CONNECTION_STRING`

SMTPNotifier settings

* `SMTP_SERVER`
* `SMTP_PORT`
* `SMTP_USE_TLS`
* `SMTP_FROM`
* `SMTP_TO`
* `SMTP_PASSWORD`

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

## Output

Requests are stored on a path defined by the `OUTPUT_FOLDER` variable. If ommited it defaults to `glutton`.

## Future

In the future releases you hopefully find the following features

✔️️️ saving to database

✔️ redirect on save 

️️✔️ auth tokens (allow saving with a valid token only)


above all, keep this project low profile, I'm not building an application server here. glutton must be simple, stupid.
