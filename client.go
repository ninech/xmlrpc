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
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/rpc"
	"net/url"
	"sync"
)

// ErrCodecClosed is returned when attempting to read from a closed codec.
var ErrCodecClosed = fmt.Errorf("xmlrpc: codec is closed")

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

// Client represents an XML-RPC client. It embeds [rpc.Client] and
// provides all of its methods, including Call and Close.
type Client struct {
	*rpc.Client
}

// clientCodec is rpc.ClientCodec interface implementation.
type clientCodec struct {
	// url presents url of xmlrpc service
	url *url.URL

	// httpClient works with HTTP protocol
	httpClient *http.Client

	// cookies stores cookies received on last request
	cookies http.CookieJar

	// headers are added to each request
	headers http.Header

	// responses presents map of active requests. It is required to return request id, that
	// rpc.Client can mark them as done.
	responses map[uint64]*http.Response
	mutex     sync.Mutex

	response Response

	// ready presents channel, that is used to link request and it`s response.
	ready chan uint64

	// close notifies codec is closed.
	close chan uint64
}

func (codec *clientCodec) WriteRequest(request *rpc.Request, args any) (err error) {
	httpRequest, err := NewRequest(codec.url.String(), request.ServiceMethod, args)
	if err != nil {
		return err
	}

	for key, values := range codec.headers {
		for _, value := range values {
			httpRequest.Header.Add(key, value)
		}
	}

	if codec.cookies != nil {
		for _, cookie := range codec.cookies.Cookies(codec.url) {
			httpRequest.AddCookie(cookie)
		}
	}

	var httpResponse *http.Response
	httpResponse, err = codec.httpClient.Do(httpRequest)
	if err != nil {
		return err
	}

	if codec.cookies != nil {
		codec.cookies.SetCookies(codec.url, httpResponse.Cookies())
	}

	codec.mutex.Lock()
	codec.responses[request.Seq] = httpResponse
	codec.mutex.Unlock()

	codec.ready <- request.Seq

	return nil
}

func (codec *clientCodec) ReadResponseHeader(response *rpc.Response) (err error) {
	var seq uint64
	select {
	case seq = <-codec.ready:
	case <-codec.close:
		return ErrCodecClosed
	}
	response.Seq = seq

	codec.mutex.Lock()
	httpResponse := codec.responses[seq]
	delete(codec.responses, seq)
	codec.mutex.Unlock()

	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		response.Error = fmt.Sprintf("xmlrpc: unexpected status code %d", httpResponse.StatusCode)
		return nil
	}

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		response.Error = err.Error()
		return nil
	}

	resp := Response(body)
	if err := resp.Err(); err != nil {
		response.Error = err.Error()
		return nil
	}

	codec.response = resp

	return nil
}

func (codec *clientCodec) ReadResponseBody(v any) (err error) {
	if v == nil {
		return nil
	}
	return codec.response.Unmarshal(v)
}

func (codec *clientCodec) Close() error {
	if transport, ok := codec.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	close(codec.close)

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

	codec := clientCodec{
		url:        u,
		httpClient: httpClient,
		close:      make(chan uint64),
		ready:      make(chan uint64),
		responses:  make(map[uint64]*http.Response),
		cookies:    jar,
		headers:    options.headers,
	}

	return &Client{rpc.NewClientWithCodec(&codec)}, nil
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
