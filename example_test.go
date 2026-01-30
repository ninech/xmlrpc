package xmlrpc_test

import (
	"fmt"
	"log"

	"github.com/RedisLabs/xmlrpc"
)

func Example() {
	client, err := xmlrpc.NewClientWithOptions("https://example.com/xmlrpc",
		xmlrpc.WithHeader("User-Agent", "my-app/1.0"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	var result struct {
		Version string `xmlrpc:"version"`
	}
	if err := client.Call("App.version", nil, &result); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Version: %s\n", result.Version)
}

func ExampleWithBasicAuth() {
	client, err := xmlrpc.NewClientWithOptions("https://example.com/xmlrpc",
		xmlrpc.WithBasicAuth("username", "password"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	var result string
	if err := client.Call("Secure.method", nil, &result); err != nil {
		log.Fatal(err)
	}
}

func ExampleEncodeMethodCall() {
	data, err := xmlrpc.EncodeMethodCall("Math.add", 1, 2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
	// Output: <?xml version="1.0" encoding="UTF-8"?><methodCall><methodName>Math.add</methodName><params><param><value><int>1</int></value></param><param><value><int>2</int></value></param></params></methodCall>
}
