package xmlrpc

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCall(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
			<methodResponse>
				<params>
					<param>
						<value><string>hello</string></value>
					</param>
				</params>
			</methodResponse>`); err != nil {
			t.Fatalf("io.WriteString error: %v", err)
		}
	})

	client, err := NewClientWithOptions(ts.URL)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}
	defer client.Close()

	var result string
	if err := client.Call("test.method", nil, &result); err != nil {
		t.Fatalf("Call error: %v", err)
	}

	if result != "hello" {
		t.Fatalf("expected 'hello', got '%s'", result)
	}
}

func TestCallContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		handler    http.HandlerFunc
		setupCtx   func() (context.Context, context.CancelFunc)
		wantErr    bool
		errContain string
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if _, err := io.WriteString(
					w,
					`<?xml version="1.0"?><methodResponse><params><param><value><string>ok</string></value></param></params></methodResponse>`,
				); err != nil {
					t.Fatalf("io.WriteString error: %v", err)
				}
			},
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(t.Context(), 5*time.Second)
			},
			wantErr: false,
		},
		{
			name: "cancelled",
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(100 * time.Millisecond)
				if _, err := io.WriteString(
					w,
					`<?xml version="1.0"?><methodResponse><params><param><value><string>ok</string></value></param></params></methodResponse>`,
				); err != nil {
					t.Fatalf("io.WriteString error: %v", err)
				}
			},
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(t.Context())
				cancel() // Cancel immediately
				return ctx, func() {}
			},
			wantErr:    true,
			errContain: "context canceled",
		},
		{
			name: "timeout",
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(200 * time.Millisecond)
				if _, err := io.WriteString(
					w,
					`<?xml version="1.0"?><methodResponse><params><param><value><string>ok</string></value></param></params></methodResponse>`,
				); err != nil {
					t.Fatalf("io.WriteString error: %v", err)
				}
			},
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(t.Context(), 50*time.Millisecond)
			},
			wantErr:    true,
			errContain: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := newTestServer(t, tt.handler)
			client, err := NewClientWithOptions(ts.URL)
			if err != nil {
				t.Fatalf("NewClientWithOptions error: %v", err)
			}
			defer client.Close()

			ctx, cancel := tt.setupCtx()
			defer cancel()

			var result string
			err = client.CallContext(ctx, "test.method", nil, &result)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Fatalf("expected error containing %q, got: %v", tt.errContain, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCallWithHeaders(t *testing.T) {
	t.Parallel()

	var receivedHeaders http.Header
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		if _, err := io.WriteString(
			w,
			`<?xml version="1.0"?><methodResponse><params><param><value><string>ok</string></value></param></params></methodResponse>`,
		); err != nil {
			t.Fatal(err)
		}
	})

	client, err := NewClientWithOptions(ts.URL,
		WithHeader("X-Custom-Header", "custom-value"),
		WithHeader("X-Another", "another-value"),
	)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}
	defer client.Close()

	var result string
	if err := client.Call("test.method", nil, &result); err != nil {
		t.Fatalf("Call error: %v", err)
	}

	if got := receivedHeaders.Get("X-Custom-Header"); got != "custom-value" {
		t.Errorf("X-Custom-Header: expected 'custom-value', got '%s'", got)
	}
	if got := receivedHeaders.Get("X-Another"); got != "another-value" {
		t.Errorf("X-Another: expected 'another-value', got '%s'", got)
	}
}

func TestCallWithBasicAuth(t *testing.T) {
	t.Parallel()

	var receivedAuth string
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		if _, err := io.WriteString(
			w,
			`<?xml version="1.0"?><methodResponse><params><param><value><string>ok</string></value></param></params></methodResponse>`,
		); err != nil {
			t.Fatal(err)
		}
	})

	client, err := NewClientWithOptions(ts.URL, WithBasicAuth("testuser", "testpass"))
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}
	defer client.Close()

	var result string
	if err := client.Call("test.method", nil, &result); err != nil {
		t.Fatalf("Call error: %v", err)
	}

	// Basic auth header should be "Basic dGVzdHVzZXI6dGVzdHBhc3M=" (base64 of "testuser:testpass")
	if receivedAuth != "Basic dGVzdHVzZXI6dGVzdHBhc3M=" {
		t.Errorf("expected basic auth header, got '%s'", receivedAuth)
	}
}

func TestCallBadStatus(t *testing.T) {
	t.Parallel()

	statusCodes := []int{400, 401, 403, 404, 500, 502, 503}

	for _, code := range statusCodes {
		t.Run(http.StatusText(code), func(t *testing.T) {
			t.Parallel()

			ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "error", code)
			})

			client, err := NewClientWithOptions(ts.URL)
			if err != nil {
				t.Fatalf("NewClientWithOptions error: %v", err)
			}
			defer client.Close()

			var result string
			err = client.Call("test.method", nil, &result)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), string(rune('0'+code/100))) {
				t.Fatalf("expected status code %d in error, got: %v", code, err)
			}
		})
	}
}

func TestCallFaultResponse(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
			<methodResponse>
				<fault>
					<value>
						<struct>
							<member>
								<name>faultCode</name>
								<value><int>4</int></value>
							</member>
							<member>
								<name>faultString</name>
								<value><string>Too many parameters.</string></value>
							</member>
						</struct>
					</value>
				</fault>
			</methodResponse>`); err != nil {
			t.Fatal(err)
		}
	})

	client, err := NewClientWithOptions(ts.URL)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}
	defer client.Close()

	var result string
	err = client.Call("test.method", nil, &result)
	if err == nil {
		t.Fatal("expected fault error, got nil")
	}

	fault, ok := err.(FaultError)
	if !ok {
		t.Fatalf("expected FaultError, got %T: %v", err, err)
	}
	if fault.Code != 4 {
		t.Errorf("expected fault code 4, got %d", fault.Code)
	}
	if !strings.Contains(fault.String, "Too many parameters") {
		t.Errorf("expected 'Too many parameters' in fault string, got %q", fault.String)
	}
}

func TestCallBadStatusRecovery(t *testing.T) {
	t.Parallel()

	callCount := 0
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			http.Error(w, "bad status", http.StatusInternalServerError)
			return
		}
		if _, err := io.WriteString(
			w,
			`<?xml version="1.0"?><methodResponse><params><param><value><struct></struct></value></param></params></methodResponse>`,
		); err != nil {
			t.Fatal(err)
		}
	})

	client, err := NewClient(ts.URL, nil)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	defer client.Close()

	var result any

	// First call should fail
	if err := client.Call("method", nil, &result); err == nil {
		t.Fatal("expected error on first call")
	}

	// Second call should succeed
	if err := client.Call("method", nil, &result); err != nil {
		t.Fatalf("expected success on second call, got: %v", err)
	}
}

func TestCallConcurrent(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(
			w,
			`<?xml version="1.0"?><methodResponse><params><param><value><string>ok</string></value></param></params></methodResponse>`,
		); err != nil {
			t.Fatal(err)
		}
	})

	client, err := NewClientWithOptions(ts.URL)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}
	defer client.Close()

	const numGoroutines = 100
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for range numGoroutines {
		wg.Go(func() {
			var result string
			if err := client.Call("test.method", nil, &result); err != nil {
				errors <- err
			}
		})
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent call error: %v", err)
	}
}

// Helper function to create a test server
func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return ts
}

func BenchmarkCall(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(
			w,
			`<?xml version="1.0"?><methodResponse><params><param><value><string>ok</string></value></param></params></methodResponse>`,
		); err != nil {
			b.Fatal(err)
		}
	}))
	defer ts.Close()

	client, err := NewClientWithOptions(ts.URL)
	if err != nil {
		b.Fatalf("NewClientWithOptions error: %v", err)
	}
	defer client.Close()

	b.ResetTimer()
	for b.Loop() {
		var result string
		if err := client.Call("test.method", nil, &result); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCallParallel(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(
			w,
			`<?xml version="1.0"?><methodResponse><params><param><value><string>ok</string></value></param></params></methodResponse>`,
		); err != nil {
			b.Fatal(err)
		}
	}))
	defer ts.Close()

	client, err := NewClientWithOptions(ts.URL)
	if err != nil {
		b.Fatalf("NewClientWithOptions error: %v", err)
	}
	defer client.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var result string
			if err := client.Call("test.method", nil, &result); err != nil {
				b.Fatal(err)
			}
		}
	})
}
