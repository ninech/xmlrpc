[![Go Reference](https://pkg.go.dev/badge/github.com/ninech/xmlrpc#section-readme.svg)](https://pkg.go.dev/github.com/ninech/xmlrpc#section-readme) [![Go Report Card](https://goreportcard.com/badge/github.com/ninech/xmlrpc)](https://goreportcard.com/report/github.com/ninech/xmlrpc)

## Overview

xmlrpc is an implementation of client side part of XML-RPC protocol in Go.

## Status

This project is in minimal maintenance mode with no further development. Bug fixes
are accepted, but it might take some time until they are merged.

## Installation

```bash
go get github.com/ninech/xmlrpc
```

## Usage

```go
client, err := xmlrpc.NewClientWithOptions("https://bugzilla.mozilla.org/xmlrpc.cgi")
if err != nil {
    log.Fatal(err)
}
defer client.Close()

var result struct {
    Version string `xmlrpc:"version"`
}
if err := client.Call("Bugzilla.version", nil, &result); err != nil {
    log.Fatal(err)
}
fmt.Printf("Version: %s\n", result.Version) // Version: 4.2.7+
```

### With Context

Use `CallContext` for requests with timeout or cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

var result string
if err := client.CallContext(ctx, "App.status", nil, &result); err != nil {
    log.Fatal(err)
}
```

### Client Options

Configure the client with functional options:

```go
client, err := xmlrpc.NewClientWithOptions(url,
    xmlrpc.WithBasicAuth("username", "password"),
    xmlrpc.WithHeader("User-Agent", "my-app/1.0"),
)
```

Available options:

- `WithHTTPClient(*http.Client)` - use a custom HTTP client
- `WithTransport(http.RoundTripper)` - set a custom transport
- `WithHeader(key, value string)` - add a header to all requests
- `WithBasicAuth(user, pass string)` - set basic authentication
- `WithCookieJar(http.CookieJar)` - set a custom cookie jar

### Arguments encoding

xmlrpc supports encoding of native Go data types to method arguments.

Data types encoding rules:

- `int`, `int8`, `int16`, `int32`, `int64` encoded to `int`
- `float32`, `float64` encoded to `double`
- `bool` encoded to `boolean`
- `string` encoded to `string`
- `time.Time` encoded to `dateTime.iso8601`
- `xmlrpc.Base64` encoded to `base64`
- slices encoded to `array`

Structs are encoded to `struct` by the following rules:

- all public fields become struct members
- field name becomes member name
- if field has `xmlrpc` tag, its value becomes member name
- for fields tagged with `omitempty`, empty values are omitted
- fields tagged with `-` are omitted

Example:

```go
type Book struct {
    Title  string `xmlrpc:"title"`
    Author string `xmlrpc:"author,omitempty"`
    ISBN   string `xmlrpc:"-"`
}
```

Server methods can accept multiple arguments. To handle this case, use a slice
of `[]any`. Each value of such slice is encoded as a separate argument.

### Result decoding

Result of remote function is decoded to native Go data type.

Data types decoding rules:

- `int`, `i4` decoded to `int`, `int8`, `int16`, `int32`, `int64`
- `double` decoded to `float32`, `float64`
- `boolean` decoded to `bool`
- `string` decoded to `string`
- `array` decoded to slice
- `struct` decoded following the rules described in previous section
- `dateTime.iso8601` decoded to `time.Time`
- `base64` decoded to `string`

## Testing

Run unit tests:

```bash
go test ./...
```

Run integration tests (requires Docker):

```bash
docker-compose up -d
go test -tags integration ./...
docker-compose down
```

## Contribution

See [project status](#status).

## Authors

Dmitry Maksimov (dmtmax@gmail.com)
