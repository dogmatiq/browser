# Environment Variables

This document describes the environment variables used by `browser`.

| Name                       | Optionality         | Description                                                                |
| -------------------------- | ------------------- | -------------------------------------------------------------------------- |
| [`CONCURRENT_ANALYZERS`]   | defaults to `+8`    | the maximum number of Go modules to analyze concurrently                   |
| [`CONCURRENT_DOWNLOADS`]   | defaults to `+80`   | the maximum number of Go modules to download concurrently                  |
| [`DEBUG`]                  | defaults to `false` | enable debug logging                                                       |
| [`GITHUB_APP_CLIENT_ID`]   | required            | the client ID of the GitHub application used to read repository content    |
| [`GITHUB_APP_HOOK_SECRET`] | required            | the secret used to verify GitHub web-hook requests are genuine             |
| [`GITHUB_APP_PRIVATE_KEY`] | required            | the private key for the GitHub application used to read repository content |
| [`GITHUB_URL`]             | optional            | the base URL of the GitHub API                                             |
| [`HTTP_LISTEN_PORT`]       | defaults to `8080`  | the port to listen on for HTTP requests                                    |

⚠️ `browser` may consume other undocumented environment variables. This document
only shows variables declared using [Ferrite].

## Specification

All environment variables described below must meet the stated requirements.
Otherwise, `browser` prints usage information to `STDERR` then exits.
**Undefined** variables and **empty** values are equivalent.

⚠️ This section includes **non-normative** example values. These examples are
syntactically valid, but may not be meaningful to `browser`.

The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**,
**SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this
document are to be interpreted as described in [RFC 2119].

### `CONCURRENT_ANALYZERS`

> the maximum number of Go modules to analyze concurrently

The `CONCURRENT_ANALYZERS` variable **MAY** be left undefined, in which case the
default value of `+8` is used. Otherwise, the value **MUST** be `+1` or greater.

```bash
export CONCURRENT_ANALYZERS=+8 # (default)
export CONCURRENT_ANALYZERS=+1 # (non-normative) the minimum accepted value
```

<details>
<summary>Signed integer syntax</summary>

Signed integers can only be specified using decimal notation. A leading positive
sign (`+`) is **OPTIONAL**. A leading negative sign (`-`) is **REQUIRED** in
order to specify a negative value.

Internally, the `CONCURRENT_ANALYZERS` variable is represented using a signed
64-bit integer type (`int`); any value that overflows this data-type is invalid.

</details>

### `CONCURRENT_DOWNLOADS`

> the maximum number of Go modules to download concurrently

The `CONCURRENT_DOWNLOADS` variable **MAY** be left undefined, in which case the
default value of `+80` is used. Otherwise, the value **MUST** be `+1` or
greater.

```bash
export CONCURRENT_DOWNLOADS=+80 # (default)
export CONCURRENT_DOWNLOADS=+1  # (non-normative) the minimum accepted value
```

<details>
<summary>Signed integer syntax</summary>

Signed integers can only be specified using decimal notation. A leading positive
sign (`+`) is **OPTIONAL**. A leading negative sign (`-`) is **REQUIRED** in
order to specify a negative value.

Internally, the `CONCURRENT_DOWNLOADS` variable is represented using a signed
64-bit integer type (`int`); any value that overflows this data-type is invalid.

</details>

### `DEBUG`

> enable debug logging

The `DEBUG` variable **MAY** be left undefined, in which case the default value
of `false` is used. Otherwise, the value **MUST** be either `true` or `false`.

```bash
export DEBUG=true
export DEBUG=false # (default)
```

### `GITHUB_APP_CLIENT_ID`

> the client ID of the GitHub application used to read repository content

The `GITHUB_APP_CLIENT_ID` variable **MUST NOT** be left undefined.

```bash
export GITHUB_APP_CLIENT_ID=foo # (non-normative)
```

### `GITHUB_APP_HOOK_SECRET`

> the secret used to verify GitHub web-hook requests are genuine

The `GITHUB_APP_HOOK_SECRET` variable **MUST NOT** be left undefined.

⚠️ This variable is **sensitive**; its value may contain private information.

### `GITHUB_APP_PRIVATE_KEY`

> the private key for the GitHub application used to read repository content

The `GITHUB_APP_PRIVATE_KEY` variable **MUST NOT** be left undefined.

⚠️ This variable is **sensitive**; its value may contain private information.

### `GITHUB_URL`

> the base URL of the GitHub API

The `GITHUB_URL` variable **MAY** be left undefined. Otherwise, the value
**MUST** be a fully-qualified URL.

```bash
export GITHUB_URL=https://example.org/path # (non-normative) a typical URL for a web page
```

<details>
<summary>URL syntax</summary>

A fully-qualified URL includes both a scheme (protocol) and a hostname. URLs are
not necessarily web addresses; `https://example.org` and
`mailto:contact@example.org` are both examples of fully-qualified URLs.

</details>

### `HTTP_LISTEN_PORT`

> the port to listen on for HTTP requests

The `HTTP_LISTEN_PORT` variable **MAY** be left undefined, in which case the
default value of `8080` is used. Otherwise, the value **MUST** be a valid
network port.

```bash
export HTTP_LISTEN_PORT=8080  # (default)
export HTTP_LISTEN_PORT=8000  # (non-normative) a port commonly used for private web servers
export HTTP_LISTEN_PORT=https # (non-normative) the IANA service name that maps to port 443
```

<details>
<summary>Network port syntax</summary>

Ports may be specified as a numeric value no greater than `65535`.
Alternatively, a service name can be used. Service names are resolved against
the system's service database, typically located in the `/etc/service` file on
UNIX-like systems. Standard service names are published by IANA.

</details>

<!-- references -->

[`concurrent_analyzers`]: #CONCURRENT_ANALYZERS
[`concurrent_downloads`]: #CONCURRENT_DOWNLOADS
[`debug`]: #DEBUG
[ferrite]: https://github.com/dogmatiq/ferrite
[`github_app_client_id`]: #GITHUB_APP_CLIENT_ID
[`github_app_hook_secret`]: #GITHUB_APP_HOOK_SECRET
[`github_app_private_key`]: #GITHUB_APP_PRIVATE_KEY
[`github_url`]: #GITHUB_URL
[`http_listen_port`]: #HTTP_LISTEN_PORT
[rfc 2119]: https://www.rfc-editor.org/rfc/rfc2119.html
