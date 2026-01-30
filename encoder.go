package xmlrpc

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Base64 is a string type that will be encoded as base64 in XML-RPC requests.
type Base64 string

func marshal(v any) ([]byte, error) {
	if v == nil {
		return []byte{}, nil
	}

	val := reflect.ValueOf(v)
	return encodeValue(val)
}

func encodeValue(val reflect.Value) ([]byte, error) {
	var b []byte
	var err error

	if val.Kind() == reflect.Pointer || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return []byte("<value/>"), nil
		}

		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		if t, ok := val.Interface().(time.Time); ok {
			b = fmt.Appendf(nil, "<dateTime.iso8601>%s</dateTime.iso8601>", t.Format(iso8601))
		} else {
			b, err = encodeStruct(val)
		}
	case reflect.Map:
		b, err = encodeMap(val)
	case reflect.Slice:
		b, err = encodeSlice(val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		b = fmt.Appendf(nil, "<int>%s</int>", strconv.FormatInt(val.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		b = fmt.Appendf(nil, "<i4>%s</i4>", strconv.FormatUint(val.Uint(), 10))
	case reflect.Float32, reflect.Float64:
		b = fmt.Appendf(nil, "<double>%s</double>",
			strconv.FormatFloat(val.Float(), 'f', -1, val.Type().Bits()))
	case reflect.Bool:
		if val.Bool() {
			b = []byte("<boolean>1</boolean>")
		} else {
			b = []byte("<boolean>0</boolean>")
		}
	case reflect.String:
		var buf bytes.Buffer

		xml.Escape(&buf, []byte(val.String()))

		if _, ok := val.Interface().(Base64); ok {
			b = fmt.Appendf(nil, "<base64>%s</base64>", buf.String())
		} else {
			b = fmt.Appendf(nil, "<string>%s</string>", buf.String())
		}
	default:
		return nil, fmt.Errorf("xmlrpc: unsupported type %s", val.Kind())
	}

	if err != nil {
		return nil, err
	}

	return fmt.Appendf(nil, "<value>%s</value>", string(b)), nil
}

func encodeStruct(structVal reflect.Value) ([]byte, error) {
	var b bytes.Buffer

	b.WriteString("<struct>")

	structType := structVal.Type()
	for i := 0; i < structType.NumField(); i++ {
		fieldVal := structVal.Field(i)
		fieldType := structType.Field(i)

		name := fieldType.Tag.Get("xmlrpc")
		// skip ignored fields.
		if name == "-" {
			continue
		}
		// if the tag has the omitempty property, skip it
		if strings.HasSuffix(name, ",omitempty") && fieldVal.IsZero() {
			continue
		}
		name = strings.TrimSuffix(name, ",omitempty")
		if name == "" {
			name = fieldType.Name
		}

		p, err := encodeValue(fieldVal)
		if err != nil {
			return nil, err
		}

		fmt.Fprintf(&b, "<member><name>%s</name>", name)
		b.Write(p)
		b.WriteString("</member>")
	}

	b.WriteString("</struct>")

	return b.Bytes(), nil
}

func encodeMap(val reflect.Value) ([]byte, error) {
	t := val.Type()

	if t.Key().Kind() != reflect.String {
		return nil, fmt.Errorf(
			"xmlrpc: map key type %s not supported, must be string",
			t.Key().Kind(),
		)
	}

	var b bytes.Buffer

	b.WriteString("<struct>")

	keys := val.MapKeys()
	slices.SortFunc(keys, func(a, b reflect.Value) int {
		return strings.Compare(a.String(), b.String())
	})

	for _, key := range keys {
		kval := val.MapIndex(key)

		fmt.Fprintf(&b, "<member><name>%s</name>", key.String())

		p, err := encodeValue(kval)
		if err != nil {
			return nil, err
		}

		b.Write(p)
		b.WriteString("</member>")
	}

	b.WriteString("</struct>")

	return b.Bytes(), nil
}

func encodeSlice(val reflect.Value) ([]byte, error) {
	var b bytes.Buffer

	b.WriteString("<array><data>")

	for i := 0; i < val.Len(); i++ {
		p, err := encodeValue(val.Index(i))
		if err != nil {
			return nil, err
		}

		b.Write(p)
	}

	b.WriteString("</data></array>")

	return b.Bytes(), nil
}
