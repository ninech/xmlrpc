package xmlrpc

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
)

// NewRequest creates an [http.Request] for an XML-RPC call to the given URL.
// The method parameter is the XML-RPC method name, and args contains the arguments
// to pass to the remote method.
func NewRequest(url string, method string, args any) (*http.Request, error) {
	return NewRequestContext(context.Background(), url, method, args)
}

// NewRequestContext creates an [http.Request] with context for an XML-RPC call to the given URL.
// The method parameter is the XML-RPC method name, and args contains the arguments
// to pass to the remote method.
func NewRequestContext(
	ctx context.Context,
	url string,
	method string,
	args any,
) (*http.Request, error) {
	var t []any
	var ok bool
	if t, ok = args.([]any); !ok {
		if args != nil {
			t = []any{args}
		}
	}

	body, err := EncodeMethodCall(method, t...)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "text/xml")
	request.Header.Set("Content-Length", strconv.Itoa(len(body)))

	return request, nil
}

// EncodeMethodCall encodes an XML-RPC method call with the given method name
// and arguments into XML bytes.
func EncodeMethodCall(method string, args ...any) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><methodCall><methodName>`)
	if err := xml.EscapeText(&b, []byte(method)); err != nil {
		return nil, fmt.Errorf("xmlrpc: failed to encode method name: %w", err)
	}
	b.WriteString("</methodName>")

	if args != nil {
		b.WriteString("<params>")

		for _, arg := range args {
			p, err := marshal(arg)
			if err != nil {
				return nil, fmt.Errorf("xmlrpc: failed to encode argument: %w", err)
			}

			b.WriteString("<param>")
			b.Write(p)
			b.WriteString("</param>")
		}

		b.WriteString("</params>")
	}

	b.WriteString("</methodCall>")

	return b.Bytes(), nil
}
