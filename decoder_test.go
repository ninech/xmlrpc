package xmlrpc

import (
	"errors"
	"io"
	"os"
	"reflect"
	"testing"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

type book struct {
	Title  string
	Amount int
}

type bookUnexported struct {
	title  string //lint:ignore U1000 intentionally unexported for testing unmarshalling behavior
	amount int    //lint:ignore U1000 intentionally unexported for testing unmarshalling behavior
}

func testTime(year int, month time.Month, day, hour, min, sec int, loc *time.Location) time.Time {
	return time.Date(year, month, day, hour, min, sec, 0, loc)
}

var unmarshalTests = []struct {
	name  string
	value any
	ptr   any
	xml   string
}{
	// int, i4, i8
	{"int/empty", 0, new(*int), "<value><int></int></value>"},
	{"int/positive", 100, new(*int), "<value><int>100</int></value>"},
	{"i4", 389451, new(*int), "<value><i4>389451</i4></value>"},
	{"i8", int64(45659074), new(*int64), "<value><i8>45659074</i8></value>"},

	// string
	{
		"string/simple",
		"Once upon a time",
		new(*string),
		"<value><string>Once upon a time</string></value>",
	},
	{
		"string/escaped",
		"Mike & Mick <London, UK>",
		new(*string),
		"<value><string>Mike &amp; Mick &lt;London, UK&gt;</string></value>",
	},
	{"string/implicit", "Once upon a time", new(*string), "<value>Once upon a time</value>"},

	// base64
	{
		"base64",
		"T25jZSB1cG9uIGEgdGltZQ==",
		new(*string),
		"<value><base64>T25jZSB1cG9uIGEgdGltZQ==</base64></value>",
	},

	// boolean
	{"boolean/true", true, new(*bool), "<value><boolean>1</boolean></value>"},
	{"boolean/false", false, new(*bool), "<value><boolean>0</boolean></value>"},

	// double
	{"double/positive", 12.134, new(*float32), "<value><double>12.134</double></value>"},
	{"double/negative", -12.134, new(*float32), "<value><double>-12.134</double></value>"},

	// datetime.iso8601
	{
		"datetime/basic",
		testTime(2013, 12, 9, 21, 0, 12, time.UTC),
		new(*time.Time),
		"<value><dateTime.iso8601>20131209T21:00:12</dateTime.iso8601></value>",
	},
	{
		"datetime/Z",
		testTime(2013, 12, 9, 21, 0, 12, time.UTC),
		new(*time.Time),
		"<value><dateTime.iso8601>20131209T21:00:12Z</dateTime.iso8601></value>",
	},
	{
		"datetime/negative_offset",
		testTime(2013, 12, 9, 21, 0, 12, time.FixedZone("", -3600)),
		new(*time.Time),
		"<value><dateTime.iso8601>20131209T21:00:12-01:00</dateTime.iso8601></value>",
	},
	{
		"datetime/positive_offset",
		testTime(2013, 12, 9, 21, 0, 12, time.FixedZone("", 3600)),
		new(*time.Time),
		"<value><dateTime.iso8601>20131209T21:00:12+01:00</dateTime.iso8601></value>",
	},
	{
		"datetime/hyphen",
		testTime(2013, 12, 9, 21, 0, 12, time.UTC),
		new(*time.Time),
		"<value><dateTime.iso8601>2013-12-09T21:00:12</dateTime.iso8601></value>",
	},
	{
		"datetime/hyphen_Z",
		testTime(2013, 12, 9, 21, 0, 12, time.UTC),
		new(*time.Time),
		"<value><dateTime.iso8601>2013-12-09T21:00:12Z</dateTime.iso8601></value>",
	},
	{
		"datetime/hyphen_negative_offset",
		testTime(2013, 12, 9, 21, 0, 12, time.FixedZone("", -3600)),
		new(*time.Time),
		"<value><dateTime.iso8601>2013-12-09T21:00:12-01:00</dateTime.iso8601></value>",
	},
	{
		"datetime/hyphen_positive_offset",
		testTime(2013, 12, 9, 21, 0, 12, time.FixedZone("", 3600)),
		new(*time.Time),
		"<value><dateTime.iso8601>2013-12-09T21:00:12+01:00</dateTime.iso8601></value>",
	},

	// array
	{
		"array/int",
		[]int{1, 5, 7},
		new(*[]int),
		"<value><array><data><value><int>1</int></value><value><int>5</int></value><value><int>7</int></value></data></array></value>",
	},
	{
		"array/any_strings",
		[]any{"A", "5"},
		new(any),
		"<value><array><data><value><string>A</string></value><value><string>5</string></value></data></array></value>",
	},
	{
		"array/any_mixed",
		[]any{"A", int64(5)},
		new(any),
		"<value><array><data><value><string>A</string></value><value><int>5</int></value></data></array></value>",
	},

	// struct
	{
		"struct/book",
		book{"War and Piece", 20},
		new(*book),
		"<value><struct><member><name>Title</name><value><string>War and Piece</string></value></member><member><name>Amount</name><value><int>20</int></value></member></struct></value>",
	},
	{
		"struct/unexported",
		bookUnexported{},
		new(*bookUnexported),
		"<value><struct><member><name>title</name><value><string>War and Piece</string></value></member><member><name>amount</name><value><int>20</int></value></member></struct></value>",
	},
	{
		"struct/to_map",
		map[string]any{"Name": "John Smith"},
		new(any),
		"<value><struct><member><name>Name</name><value><string>John Smith</string></value></member></struct></value>",
	},
	{"struct/empty", map[string]any{}, new(any), "<value><struct></struct></value>"},
}

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	for _, tt := range unmarshalTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := reflect.New(reflect.TypeOf(tt.value))
			if err := unmarshal([]byte(tt.xml), v.Interface()); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			v = v.Elem()

			if v.Kind() == reflect.Slice {
				vv := reflect.ValueOf(tt.value)
				if vv.Len() != v.Len() {
					t.Fatalf(
						"unmarshal error:\nexpected: %v\n     got: %v",
						tt.value,
						v.Interface(),
					)
				}
				for i := 0; i < v.Len(); i++ {
					if v.Index(i).Interface() != vv.Index(i).Interface() {
						t.Fatalf(
							"unmarshal error:\nexpected: %v\n     got: %v",
							tt.value,
							v.Interface(),
						)
					}
				}
			} else if t1, ok := v.Interface().(time.Time); ok {
				t2 := tt.value.(time.Time)
				if !t1.Equal(t2) {
					t.Fatalf(
						"unmarshal error:\nexpected: %v\n     got: %v",
						tt.value,
						v.Interface(),
					)
				}
			} else {
				a1 := v.Interface()
				a2 := any(tt.value)

				if !reflect.DeepEqual(a1, a2) {
					t.Fatalf(
						"unmarshal error:\nexpected: %v\n     got: %v",
						tt.value,
						v.Interface(),
					)
				}
			}
		})
	}
}

func TestUnmarshalToNil(t *testing.T) {
	t.Parallel()

	for _, tt := range unmarshalTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := unmarshal([]byte(tt.xml), tt.ptr); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}
		})
	}
}

func TestTypeMismatchError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		xml     string
		target  any
		wantErr bool
	}{
		{"int_to_string", "<value><int>100</int></value>", new(string), true},
		{"string_to_int", "<value><string>hello</string></value>", new(int), true},
		{"bool_to_string", "<value><boolean>1</boolean></value>", new(string), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := unmarshal([]byte(tt.xml), tt.target)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if _, ok := err.(TypeMismatchError); !ok {
					t.Fatalf("expected TypeMismatchError, got %T: %v", err, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUnmarshalEmptyValueTag(t *testing.T) {
	t.Parallel()

	var v int
	if err := unmarshal([]byte("<value/>"), &v); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
}

func TestUnmarshalEmptyStruct(t *testing.T) {
	t.Parallel()

	const xml = `<value><struct></struct></value>`

	var v any
	if err := unmarshal([]byte(xml), &v); err != nil {
		t.Fatal(err)
	}
	if v == nil {
		t.Fatalf("got nil map")
	}
}

func TestUnmarshalExistingArray(t *testing.T) {
	t.Parallel()

	const xml = `
<value>
  <array>
    <data>
      <value><int>234</int></value>
      <value><boolean>1</boolean></value>
      <value><string>Hello World</string></value>
      <value><string>Extra Value</string></value>
    </data>
  </array>
</value>`

	var (
		v1 int
		v2 bool
		v3 string
		v  = []any{&v1, &v2, &v3}
	)

	if err := unmarshal([]byte(xml), &v); err != nil {
		t.Fatal(err)
	}

	if want := 234; v1 != want {
		t.Fatalf("v1: want %d, got %d", want, v1)
	}
	if want := true; v2 != want {
		t.Fatalf("v2: want %t, got %t", want, v2)
	}
	if want := "Hello World"; v3 != want {
		t.Fatalf("v3: want %s, got %s", want, v3)
	}
	if n := len(v); n != 4 {
		t.Fatalf("missing appended result, len=%d", n)
	}
	if got, ok := v[3].(string); !ok || got != "Extra Value" {
		t.Fatalf("v[3]: got %s, want %s", got, "Extra Value")
	}
}

func TestDecodeNonUTF8Response(t *testing.T) {
	data, err := os.ReadFile("testdata/fixtures/cp1251.xml")
	if err != nil {
		t.Fatal(err)
	}

	CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		if charset != "cp1251" {
			return nil, errors.New("unsupported charset: " + charset)
		}
		return transform.NewReader(input, charmap.Windows1251.NewDecoder()), nil
	}
	t.Cleanup(func() { CharsetReader = nil })

	var s string
	if err = unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	expected := "Л.Н. Толстой - Война и Мир"
	if s != expected {
		t.Fatalf("unmarshal error:\nexpected: %v\n     got: %v", expected, s)
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	benchmarks := []struct {
		name string
		xml  string
		ptr  any
	}{
		{"int", "<value><int>12345</int></value>", new(int)},
		{"string", "<value><string>Hello, World!</string></value>", new(string)},
		{"bool", "<value><boolean>1</boolean></value>", new(bool)},
		{
			"struct",
			"<value><struct><member><name>Title</name><value><string>Test</string></value></member><member><name>Amount</name><value><int>100</int></value></member></struct></value>",
			new(book),
		},
		{
			"array",
			"<value><array><data><value><int>1</int></value><value><int>2</int></value><value><int>3</int></value></data></array></value>",
			new([]int),
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			data := []byte(bm.xml)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := unmarshal(data, bm.ptr); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func FuzzUnmarshalInt(f *testing.F) {
	f.Add([]byte(`<value><int>0</int></value>`))
	f.Add([]byte(`<value><int>123</int></value>`))
	f.Add([]byte(`<value><int>-456</int></value>`))
	f.Add([]byte(`<value><i4>789</i4></value>`))
	f.Add([]byte(`<value><i8>9223372036854775807</i8></value>`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var result int64
		_ = unmarshal(data, &result)
	})
}

func FuzzUnmarshalString(f *testing.F) {
	f.Add([]byte(`<value><string>hello</string></value>`))
	f.Add([]byte(`<value><string></string></value>`))
	f.Add([]byte(`<value><string>a &amp; b</string></value>`))
	f.Add([]byte(`<value>implicit string</value>`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var result string
		_ = unmarshal(data, &result)
	})
}

func FuzzUnmarshalStruct(f *testing.F) {
	f.Add([]byte(`<value><struct></struct></value>`))
	f.Add(
		[]byte(
			`<value><struct><member><name>foo</name><value><string>bar</string></value></member></struct></value>`,
		),
	)

	f.Fuzz(func(t *testing.T, data []byte) {
		var result map[string]any
		_ = unmarshal(data, &result)
	})
}

func FuzzUnmarshalArray(f *testing.F) {
	f.Add([]byte(`<value><array><data></data></array></value>`))
	f.Add([]byte(`<value><array><data><value><int>1</int></value></data></array></value>`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var result []any
		_ = unmarshal(data, &result)
	})
}

func FuzzUnmarshalAny(f *testing.F) {
	f.Add([]byte(`<value><int>1</int></value>`))
	f.Add([]byte(`<value><string>test</string></value>`))
	f.Add([]byte(`<value><boolean>1</boolean></value>`))
	f.Add([]byte(`<value><double>1.5</double></value>`))
	f.Add([]byte(`<value><struct></struct></value>`))
	f.Add([]byte(`<value><array><data></data></array></value>`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var result any
		_ = unmarshal(data, &result)
	})
}
