[![GoDoc](https://godoc.org/github.com/kolo/xmlrpc?status.svg)](https://godoc.org/github.com/kolo/xmlrpc)

## Overview

xmlrpc is an implementation of client side part of XMLRPC protocol in Go language.

## Status

This project is in minimal maintenance mode with no further development. Bug fixes
are accepted, but it might take some time until they are merged.

## Installation

To install xmlrpc package run `go get github.com/ninech/xmlrpc`. To use
it in application add `"github.com/ninech/xmlrpc"` string to `import`
statement.

## Usage

    client, _ := xmlrpc.NewClient("https://bugzilla.mozilla.org/xmlrpc.cgi", nil)
    result := struct{
      Version string `xmlrpc:"version"`
    }{}
    client.Call("Bugzilla.version", nil, &result)
    fmt.Printf("Version: %s\n", result.Version) // Version: 4.2.7+

Second argument of NewClient function is an object that implements
[http.RoundTripper](http://golang.org/pkg/net/http/#RoundTripper)
interface, it can be used to get more control over connection options.
By default it is initialized by http.DefaultTransport object.

### Arguments encoding

xmlrpc package supports encoding of native Go data types to method
arguments.

Data types encoding rules:

* int, int8, int16, int32, int64 encoded to int;
* float32, float64 encoded to double;
* bool encoded to boolean;
* string encoded to string;
* time.Time encoded to datetime.iso8601;
* xmlrpc.Base64 encoded to base64;
* slice encoded to array;

Structs are encoded to struct by the following rules:

* all public fields become struct members;
* field name becomes member name;
* if field has xmlrpc tag, its value becomes member name.
* for fields tagged with `",omitempty"`, empty values are omitted;
* fields tagged with `"-"` are omitted.

Server methods can accept multiple arguments, to handle this case there is
a special approach to handle slice of empty interfaces (`[]interface{}`).
Each value of such slice is encoded as a separate argument.

### Result decoding

Result of remote function is decoded to native Go data type.

Data types decoding rules:

* int, i4 decoded to int, int8, int16, int32, int64;
* double decoded to float32, float64;
* boolean decoded to bool;
* string decoded to string;
* array decoded to slice;
* structs are decoded following the rules described in previous section;
* datetime.iso8601 decoded as time.Time data type;
* base64 decoded to string.

## Testing

Run unit tests:

```bash
go test ./...
```

Run integration tests (requires Docker):

```bash
docker-compose up -d # Start the test server
go test -tags integration ./... # Run all tests including integration tests
docker-compose down # Stop the test server
```

## Contribution

See [project status](#status).

## Authors

Dmitry Maksimov (dmtmax@gmail.com)
