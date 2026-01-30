package xmlrpc

import (
	"testing"
	"time"
)

var marshalTests = []struct {
	name  string
	value any
	xml   string
}{
	// primitives
	{"int", 100, "<value><int>100</int></value>"},
	{"string/simple", "Once upon a time", "<value><string>Once upon a time</string></value>"},
	{
		"string/escaped",
		"Mike & Mick <London, UK>",
		"<value><string>Mike &amp; Mick &lt;London, UK&gt;</string></value>",
	},
	{
		"base64",
		Base64("T25jZSB1cG9uIGEgdGltZQ=="),
		"<value><base64>T25jZSB1cG9uIGEgdGltZQ==</base64></value>",
	},
	{"bool/true", true, "<value><boolean>1</boolean></value>"},
	{"bool/false", false, "<value><boolean>0</boolean></value>"},
	{"double/positive", 12.134, "<value><double>12.134</double></value>"},
	{"double/negative", -12.134, "<value><double>-12.134</double></value>"},
	{"double/large", 738777323.0, "<value><double>738777323</double></value>"},
	{
		"datetime",
		time.Unix(1386622812, 0).UTC(),
		"<value><dateTime.iso8601>20131209T21:00:12</dateTime.iso8601></value>",
	},

	// array
	{
		"array/mixed",
		[]any{1, "one"},
		"<value><array><data><value><int>1</int></value><value><string>one</string></value></data></array></value>",
	},

	// struct
	{"struct/simple", &struct {
		Title  string
		Amount int
	}{"War and Piece", 20}, "<value><struct><member><name>Title</name><value><string>War and Piece</string></value></member><member><name>Amount</name><value><int>20</int></value></member></struct></value>"},

	{"struct/nil_value", &struct {
		Value any `xmlrpc:"value"`
	}{}, "<value><struct><member><name>value</name><value/></member></struct></value>"},

	{"struct/omitempty_empty", &struct {
		Title  string
		Amount int
		Author string `xmlrpc:"author,omitempty"`
	}{Title: "War and Piece", Amount: 20}, "<value><struct><member><name>Title</name><value><string>War and Piece</string></value></member><member><name>Amount</name><value><int>20</int></value></member></struct></value>"},

	{"struct/omitempty_set", &struct {
		Title  string
		Amount int
		Author string `xmlrpc:"author,omitempty"`
	}{Title: "War and Piece", Amount: 20, Author: "Leo Tolstoy"}, "<value><struct><member><name>Title</name><value><string>War and Piece</string></value></member><member><name>Amount</name><value><int>20</int></value></member><member><name>author</name><value><string>Leo Tolstoy</string></value></member></struct></value>"},

	{"struct/empty", &struct{}{}, "<value><struct></struct></value>"},

	{"struct/skip_field", &struct {
		ID   int    `xmlrpc:"id"`
		Name string `xmlrpc:"-"`
	}{ID: 123, Name: "kolo"}, "<value><struct><member><name>id</name><value><int>123</int></value></member></struct></value>"},

	// map
	{
		"map/simple",
		map[string]any{"title": "War and Piece", "amount": 20},
		"<value><struct><member><name>amount</name><value><int>20</int></value></member><member><name>title</name><value><string>War and Piece</string></value></member></struct></value>",
	},

	{
		"map/nested",
		map[string]any{
			"Name":  "John Smith",
			"Age":   6,
			"Wight": []float32{66.67, 100.5},
			"Dates": map[string]any{
				"Birth": time.Date(1829, time.November, 10, 23, 0, 0, 0, time.UTC),
				"Death": time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
			},
		},
		"<value><struct><member><name>Age</name><value><int>6</int></value></member><member><name>Dates</name><value><struct><member><name>Birth</name><value><dateTime.iso8601>18291110T23:00:00</dateTime.iso8601></value></member><member><name>Death</name><value><dateTime.iso8601>20091110T23:00:00</dateTime.iso8601></value></member></struct></value></member><member><name>Name</name><value><string>John Smith</string></value></member><member><name>Wight</name><value><array><data><value><double>66.67</double></value><value><double>100.5</double></value></data></array></value></member></struct></value>",
	},
}

func TestMarshal(t *testing.T) {
	t.Parallel()

	for _, tt := range marshalTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b, err := marshal(tt.value)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}

			if string(b) != tt.xml {
				t.Fatalf("marshal error:\nexpected: %s\n     got: %s", tt.xml, string(b))
			}
		})
	}
}

func BenchmarkMarshal(b *testing.B) {
	benchmarks := []struct {
		name  string
		value any
	}{
		{"int", 12345},
		{"string", "Hello, World!"},
		{"bool", true},
		{"struct", &struct {
			Title  string
			Amount int
		}{"Test Book", 100}},
		{"array", []int{1, 2, 3, 4, 5}},
		{"map", map[string]any{"key1": "value1", "key2": 123}},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := marshal(bm.value); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func FuzzMarshal(f *testing.F) {
	f.Add("hello")
	f.Add("")
	f.Add("special <>&\"' chars")
	f.Add("unicode: 日本語")

	f.Fuzz(func(t *testing.T, s string) {
		_, _ = marshal(s)
	})
}
