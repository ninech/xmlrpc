package xmlrpc

import (
	"fmt"
	"regexp"
)

var (
	faultRx = regexp.MustCompile(`<fault>(\s|\S)+</fault>`)
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
type Response []byte

// Err checks if the response contains a fault and returns it as a [FaultError].
// If the response is not a fault, Err returns nil.
func (r Response) Err() error {
	if !faultRx.Match(r) {
		return nil
	}
	var fault FaultError
	if err := unmarshal(r, &fault); err != nil {
		return err
	}
	return fault
}

// Unmarshal decodes the XML-RPC response into v.
func (r Response) Unmarshal(v any) error {
	if err := unmarshal(r, v); err != nil {
		return err
	}

	return nil
}
