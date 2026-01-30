package xmlrpc

import (
	"bytes"
	"fmt"
)

// FaultError represents an XML-RPC fault response from the server.
type FaultError struct {
	Code   int    `xmlrpc:"faultCode"`
	String string `xmlrpc:"faultString"`
}

// Error returns the string representation of the fault.
func (e FaultError) Error() string {
	return fmt.Sprintf("Fault(%d): %s", e.Code, e.String)
}

// Response represents a raw XML-RPC response body.
//
// Deprecated: Response is no longer used internally.
// Use [Client.Call] or [Client.CallContext] instead.
type Response []byte

// Err checks if the response contains a fault and returns it as a [FaultError].
// If the response is not a fault, Err returns nil.
//
// Deprecated: Use [Client.Call] or [Client.CallContext] instead,
// which return [FaultError] directly.
func (r Response) Err() error {
	if !bytes.Contains(r, []byte("<fault>")) {
		return nil
	}
	var fault FaultError
	if err := unmarshal(r, &fault); err != nil {
		return fmt.Errorf("xmlrpc: failed to parse fault response: %w", err)
	}
	return fault
}

// Unmarshal decodes the XML-RPC response into v.
//
// Deprecated: Use [Client.Call] or [Client.CallContext] instead.
func (r Response) Unmarshal(v any) error {
	return unmarshal(r, v)
}
