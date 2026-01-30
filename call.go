package xmlrpc

import (
	"context"
	"fmt"
)

// Call invokes the named method, waits for it to complete, and returns its error status.
// This is equivalent to CallContext with [context.Background].
func (c *Client) Call(serviceMethod string, args any, reply any) error {
	return c.CallContext(context.Background(), serviceMethod, args, reply)
}

// CallContext invokes the named method with context support.
// The context controls cancellation and timeout of the HTTP request.
func (c *Client) CallContext(ctx context.Context, serviceMethod string, args any, reply any) error {
	httpRequest, err := NewRequestContext(ctx, c.url.String(), serviceMethod, args)
	if err != nil {
		return err
	}

	for key, values := range c.headers {
		for _, value := range values {
			httpRequest.Header.Add(key, value)
		}
	}

	if c.cookies != nil {
		for _, cookie := range c.cookies.Cookies(c.url) {
			httpRequest.AddCookie(cookie)
		}
	}

	resp, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if c.cookies != nil {
		c.cookies.SetCookies(c.url, resp.Cookies())
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("xmlrpc: unexpected status code %d", resp.StatusCode)
	}

	if reply == nil {
		reply = new(any)
	}
	return unmarshalResponse(resp.Body, reply)
}
