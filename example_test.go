package xmlrpc_test

import (
	"fmt"
	"log"

	"github.com/RedisLabs/xmlrpc"
)

func Example() {
	client, err := xmlrpc.NewClient("https://example.com/xmlrpc", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	var result struct {
		Version string `xmlrpc:"version"`
	}
	if err := client.Call("App.version", nil, &result); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Version: %s\n", result.Version)
}

func ExampleEncodeMethodCall() {
	data, err := xmlrpc.EncodeMethodCall("Math.add", 1, 2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
	// Output: <?xml version="1.0" encoding="UTF-8"?><methodCall><methodName>Math.add</methodName><params><param><value><int>1</int></value></param><param><value><int>2</int></value></param></params></methodCall>
}
