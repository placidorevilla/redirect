# HTTP service discovery

Simple, very fast, in a single binary, with web UI HTTP redirect service

# Install

Get [latest releases](https://github.com/reddec/redirect/releases), unpack and run.


# WEB panel

![web interface](http://reddec.github.io/images/redirect_ui.png)

# Usage

## Docker

Translate this ports as you wish:

* `10100` - main redirect server
* `10101` - UI web panel

Use exposed volume `/etc/redirect` to persist data
## CLI

### -bind

Redirect address (default "0.0.0.0:10100"). You can do any HTTP operation
to address http://your-server:10100/your/cool/service/name and it will be redirected to specified address

### -config

File to save configuration (default "./redir.json").
It will be loaded at startup (if exists) and saved after each modification operation over API

### -ui

Directory of static UI files (default "./ui").

### -ui-addr string

Address for UI (default "127.0.0.1:10101")

* `/` - Will be served as static directory from specified directory
* `/api/`  - API handlers

# Actions on redirect server

* `GET/POST/PUT/DELETE` - returns redirection with 302 Found status
* `HEAD` - returns only real service location in `Location` header with 200 OK status

# API

### GET

Get list of services or detailed information of one service if service name provided.
Adds into response headers (`X-Redir-Port`) port of main server

* Endpoint (all):  `http://ui-addr/api/`
* Endpoint (one):  `http://ui-addr/api/your/cool/service/name`

### POST

Add or update one service. If service already exists, hits will saved.
Expects this form fields:

* `service` - service name
* `template` - content of template

Each template must be valid expression of [Go template engine](https://golang.org/pkg/text/template/)
with [http request](https://golang.org/pkg/net/http/#Request) as environment.

#### Simple example

* `service` = google
* `template` = http://google.com

All requests to http://127.0.0.1:10100/google will be redirected to http://google.com

#### Complex example

* `service` = mdn
* `template`

    https://developer.mozilla.org/ru/docs/Web/JavaScript/Reference/Global_Objects/{{.URL.Query.Get "q"}}

All requests to http://127.0.0.1:10100/mdn?q=QUERY will be redirected to
https://developer.mozilla.org/ru/docs/Web/JavaScript/Reference/Global_Objects/QUERY

E.x.:

`http://127.0.0.1:10100/mdn?q=encodeuricomponent` maps to
`https://developer.mozilla.org/ru/docs/Web/JavaScript/Reference/Global_Objects/encodeuricomponent`

* Endpoint:  `http://ui-addr/api/`

### DELETE

Remove service if it exists

* Endpoint:  `http://ui-addr/api/your/cool/service/name`
