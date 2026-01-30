package xmlrpc

import (
	"testing"
)

func TestResponseErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		xml       string
		wantErr   bool
		faultCode int
	}{
		{
			name: "fault_response",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse>
  <fault>
    <value>
      <struct>
        <member>
          <name>faultString</name>
          <value><string>You must log in before using this part of Bugzilla.</string></value>
        </member>
        <member>
          <name>faultCode</name>
          <value><int>410</int></value>
        </member>
      </struct>
    </value>
  </fault>
</methodResponse>`,
			wantErr:   true,
			faultCode: 410,
		},
		{
			name: "success_response",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse>
  <params>
    <param>
      <value><string>success</string></value>
    </param>
  </params>
</methodResponse>`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := Response([]byte(tt.xml))
			err := resp.Err()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				fault, ok := err.(FaultError)
				if !ok {
					t.Fatalf("expected FaultError, got %T", err)
				}
				if fault.Code != tt.faultCode {
					t.Errorf("expected fault code %d, got %d", tt.faultCode, fault.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestResponseUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		xml      string
		target   any
		validate func(t *testing.T, v any)
	}{
		{
			name: "string",
			xml: `<?xml version="1.0"?>
<methodResponse>
  <params>
    <param>
      <value><string>hello world</string></value>
    </param>
  </params>
</methodResponse>`,
			target: new(string),
			validate: func(t *testing.T, v any) {
				if got := *v.(*string); got != "hello world" {
					t.Errorf("expected 'hello world', got %q", got)
				}
			},
		},
		{
			name: "struct_with_empty_value",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse>
  <params>
    <param>
      <value>
        <struct>
          <member>
            <name>user</name>
            <value><string>Joe Smith</string></value>
          </member>
          <member>
            <name>token</name>
            <value/>
          </member>
        </struct>
      </value>
    </param>
  </params>
</methodResponse>`,
			target: &struct {
				User  string `xmlrpc:"user"`
				Token string `xmlrpc:"token"`
			}{},
			validate: func(t *testing.T, v any) {
				result := v.(*struct {
					User  string `xmlrpc:"user"`
					Token string `xmlrpc:"token"`
				})
				if result.User != "Joe Smith" {
					t.Errorf("expected User 'Joe Smith', got %q", result.User)
				}
				if result.Token != "" {
					t.Errorf("expected Token '', got %q", result.Token)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := Response([]byte(tt.xml))
			if err := resp.Unmarshal(tt.target); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			tt.validate(t, tt.target)
		})
	}
}

func TestFaultErrorString(t *testing.T) {
	t.Parallel()

	fault := FaultError{Code: 123, String: "test error"}
	expected := "Fault(123): test error"
	if got := fault.Error(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
