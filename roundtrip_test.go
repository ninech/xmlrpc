package xmlrpc

import (
	"reflect"
	"testing"
	"time"
)

func TestRoundTripInt(t *testing.T) {
	t.Parallel()

	values := []int{0, 1, -1, 100, -100, 2147483647, -2147483648}

	for _, original := range values {
		encoded, err := marshal(original)
		if err != nil {
			t.Fatalf("marshal(%d) error: %v", original, err)
		}

		var decoded int
		if err := unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if original != decoded {
			t.Errorf("round-trip failed: original=%d, decoded=%d", original, decoded)
		}
	}
}

func TestRoundTripString(t *testing.T) {
	t.Parallel()

	values := []string{
		"",
		"hello",
		"hello world",
		"special <>&\"' chars",
		"unicode: 日本語 emoji: 🎉",
		"newline\nand\ttab",
	}

	for _, original := range values {
		encoded, err := marshal(original)
		if err != nil {
			t.Fatalf("marshal(%q) error: %v", original, err)
		}

		var decoded string
		if err := unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if original != decoded {
			t.Errorf("round-trip failed: original=%q, decoded=%q", original, decoded)
		}
	}
}

func TestRoundTripBool(t *testing.T) {
	t.Parallel()

	for _, original := range []bool{true, false} {
		encoded, err := marshal(original)
		if err != nil {
			t.Fatalf("marshal(%v) error: %v", original, err)
		}

		var decoded bool
		if err := unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if original != decoded {
			t.Errorf("round-trip failed: original=%v, decoded=%v", original, decoded)
		}
	}
}

func TestRoundTripFloat(t *testing.T) {
	t.Parallel()

	values := []float64{0, 1.0, -1.0, 3.14159, -2.71828, 1e10, 1e-10}

	for _, original := range values {
		encoded, err := marshal(original)
		if err != nil {
			t.Fatalf("marshal(%v) error: %v", original, err)
		}

		var decoded float64
		if err := unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if original != decoded {
			t.Errorf("round-trip failed: original=%v, decoded=%v", original, decoded)
		}
	}
}

func TestRoundTripTime(t *testing.T) {
	t.Parallel()

	values := []time.Time{
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		time.Date(1999, 6, 15, 12, 30, 45, 0, time.UTC),
	}

	for _, original := range values {
		encoded, err := marshal(original)
		if err != nil {
			t.Fatalf("marshal(%v) error: %v", original, err)
		}

		var decoded time.Time
		if err := unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if !original.Equal(decoded) {
			t.Errorf("round-trip failed: original=%v, decoded=%v", original, decoded)
		}
	}
}

func TestRoundTripSlice(t *testing.T) {
	t.Parallel()

	original := []int{1, 2, 3, 4, 5}

	encoded, err := marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded []int
	if err := unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip failed: original=%v, decoded=%v", original, decoded)
	}
}

func TestRoundTripStruct(t *testing.T) {
	t.Parallel()

	type TestStruct struct {
		Name   string `xmlrpc:"name"`
		Age    int    `xmlrpc:"age"`
		Active bool   `xmlrpc:"active"`
	}

	original := TestStruct{
		Name:   "John Doe",
		Age:    30,
		Active: true,
	}

	encoded, err := marshal(&original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded TestStruct
	if err := unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip failed:\noriginal=%+v\ndecoded=%+v", original, decoded)
	}
}

func TestRoundTripNestedStruct(t *testing.T) {
	t.Parallel()

	type Inner struct {
		Value string `xmlrpc:"value"`
	}
	type Outer struct {
		Name  string `xmlrpc:"name"`
		Inner Inner  `xmlrpc:"inner"`
	}

	original := Outer{
		Name:  "outer",
		Inner: Inner{Value: "inner value"},
	}

	encoded, err := marshal(&original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Outer
	if err := unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip failed:\noriginal=%+v\ndecoded=%+v", original, decoded)
	}
}

func TestRoundTripMap(t *testing.T) {
	t.Parallel()

	original := map[string]any{
		"string": "hello",
		"int":    42,
		"bool":   true,
	}

	encoded, err := marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded map[string]any
	if err := unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded["string"] != "hello" {
		t.Errorf("string mismatch: got %v", decoded["string"])
	}
	if decoded["bool"] != true {
		t.Errorf("bool mismatch: got %v", decoded["bool"])
	}
}
