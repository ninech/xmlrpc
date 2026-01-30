// Package xmlrpc provides an XML-RPC client implementation for Go.
//
// The package implements the client side of the XML-RPC protocol,
// allowing Go programs to make remote procedure calls to XML-RPC servers.
//
// Basic usage:
//
//	client, err := xmlrpc.NewClient("https://example.com/xmlrpc", nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Close()
//
//	var result string
//	err = client.Call("Method.Name", arg, &result)
package xmlrpc

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/rpc"
	"net/url"
	"sync"
)

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
		return errors.New("codec is closed")
	}
	response.Seq = seq

	codec.mutex.Lock()
	httpResponse := codec.responses[seq]
	delete(codec.responses, seq)
	codec.mutex.Unlock()

	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		response.Error = fmt.Sprintf("request error: bad status code - %d", httpResponse.StatusCode)
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

// NewClient creates a new XML-RPC client for the given URL.
// The transport parameter specifies the [http.RoundTripper] to use for HTTP requests.
// If transport is nil, [http.DefaultTransport] is used.
func NewClient(requrl string, transport http.RoundTripper) (*Client, error) {
	if transport == nil {
		transport = http.DefaultTransport
	}

	httpClient := &http.Client{Transport: transport}

	jar, err := cookiejar.New(nil)

	if err != nil {
		return nil, err
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
	}

	return &Client{rpc.NewClientWithCodec(&codec)}, nil
}
