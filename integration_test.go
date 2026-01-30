//go:build integration

package xmlrpc

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestIntegration_CallWithoutArgs(t *testing.T) {
	client := newIntegrationClient(t)
	defer client.Close()

	var result time.Time
	if err := client.Call("service.time", nil, &result); err != nil {
		t.Fatalf("service.time error: %v", err)
	}
	t.Logf("service.time: %v", result)
}

func TestIntegration_CallWithOneArg(t *testing.T) {
	client := newIntegrationClient(t)
	defer client.Close()

	var result string
	if err := client.Call("service.upcase", "xmlrpc", &result); err != nil {
		t.Fatalf("service.upcase error: %v", err)
	}

	if result != "XMLRPC" {
		t.Fatalf("expected XMLRPC, got %s", result)
	}
}

func TestIntegration_CallWithTwoArgs(t *testing.T) {
	client := newIntegrationClient(t)
	defer client.Close()

	var sum int
	if err := client.Call("service.sum", []any{2, 3}, &sum); err != nil {
		t.Fatalf("service.sum error: %v", err)
	}

	if sum != 5 {
		t.Fatalf("expected 5, got %d", sum)
	}
}

func TestIntegration_FaultError(t *testing.T) {
	client := newIntegrationClient(t)
	defer client.Close()

	var result int
	err := client.Call("service.error", nil, &result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	fault, ok := err.(FaultError)
	if !ok {
		t.Fatalf("expected FaultError, got %T: %v", err, err)
	}
	t.Logf("Got expected fault: %v", fault)
}

func TestIntegration_CallContext(t *testing.T) {
	client := newIntegrationClient(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	var result string
	if err := client.CallContext(ctx, "service.upcase", "test", &result); err != nil {
		t.Fatalf("CallContext error: %v", err)
	}

	if result != "TEST" {
		t.Fatalf("expected TEST, got %s", result)
	}
}

func TestIntegration_Concurrent(t *testing.T) {
	client := newIntegrationClient(t)
	defer client.Close()

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var result time.Time
			if err := client.Call("service.time", nil, &result); err != nil {
				t.Errorf("concurrent call error: %v", err)
			}
		}()
	}
	wg.Wait()
}

func newIntegrationClient(t *testing.T) *Client {
	client, err := NewClientWithOptions("http://localhost:5001")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}
