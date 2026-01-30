// Package xmlrpc provides an XML-RPC client implementation for Go.
//
// The package implements the client side of the XML-RPC protocol,
// allowing Go programs to make remote procedure calls to XML-RPC servers.
//
// Basic usage:
//
//	client, err := xmlrpc.NewClientWithOptions("https://example.com/xmlrpc",
//		xmlrpc.WithHeader("User-Agent", "my-app/1.0"),
//		xmlrpc.WithBasicAuth("user", "pass"),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Close()
//
//	var result string
//	err = client.Call("Method.Name", arg, &result)
package xmlrpc

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

// clientOptions holds configuration for the XML-RPC client.
type clientOptions struct {
	httpClient *http.Client
	transport  http.RoundTripper
	headers    http.Header
	cookieJar  http.CookieJar
	// useCookies distinguishes between "no jar set" and "explicitly disabled"
	useCookies *bool
}

// Option configures a [Client].
type Option func(*clientOptions)

// WithHTTPClient sets the HTTP client to use for requests.
// This takes precedence over [WithTransport].
func WithHTTPClient(client *http.Client) Option {
	return func(o *clientOptions) {
		o.httpClient = client
	}
}

// WithTransport sets the HTTP transport for requests.
// Ignored if [WithHTTPClient] is also used.
func WithTransport(transport http.RoundTripper) Option {
	return func(o *clientOptions) {
		o.transport = transport
	}
}

// WithHeader adds a header to all requests.
// Can be called multiple times to add multiple headers.
func WithHeader(key, value string) Option {
	return func(o *clientOptions) {
		if o.headers == nil {
			o.headers = make(http.Header)
		}
		o.headers.Add(key, value)
	}
}

// WithBasicAuth sets basic authentication for all requests.
func WithBasicAuth(username, password string) Option {
	return func(o *clientOptions) {
		if o.headers == nil {
			o.headers = make(http.Header)
		}
		req := &http.Request{Header: o.headers}
		req.SetBasicAuth(username, password)
	}
}

// WithCookieJar sets the cookie jar for the client.
// Pass nil to disable cookie handling.
func WithCookieJar(jar http.CookieJar) Option {
	return func(o *clientOptions) {
		o.cookieJar = jar
		useCookies := jar != nil
		o.useCookies = &useCookies
	}
}

// Client represents an XML-RPC client.
type Client struct {
	url        *url.URL
	httpClient *http.Client
	cookies    http.CookieJar
	headers    http.Header
}

// Close closes idle connections. The Client can still be used after calling Close.
func (c *Client) Close() error {
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	return nil
}

// NewClientWithOptions creates a new XML-RPC client for the given URL with the specified options.
func NewClientWithOptions(requrl string, opts ...Option) (*Client, error) {
	options := &clientOptions{}
	for _, opt := range opts {
		opt(options)
	}

	httpClient := options.httpClient
	if httpClient == nil {
		transport := options.transport
		if transport == nil {
			transport = http.DefaultTransport
		}
		httpClient = &http.Client{Transport: transport}
	}

	var jar http.CookieJar
	if options.useCookies == nil || *options.useCookies {
		if options.cookieJar != nil {
			jar = options.cookieJar
		} else {
			var err error
			jar, err = cookiejar.New(nil)
			if err != nil {
				return nil, err
			}
		}
	}

	u, err := url.Parse(requrl)
	if err != nil {
		return nil, err
	}

	return &Client{
		url:        u,
		httpClient: httpClient,
		cookies:    jar,
		headers:    options.headers,
	}, nil
}

// NewClient creates a new XML-RPC client for the given URL.
// The transport parameter specifies the [http.RoundTripper] to use for HTTP requests.
// If transport is nil, [http.DefaultTransport] is used.
//
// Deprecated: Use [NewClientWithOptions] instead.
func NewClient(requrl string, transport http.RoundTripper) (*Client, error) {
	if transport == nil {
		return NewClientWithOptions(requrl)
	}
	return NewClientWithOptions(requrl, WithTransport(transport))
}
