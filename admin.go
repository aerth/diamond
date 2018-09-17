// +build ignore

package main

import (
	"flag"
	"fmt"
	"log"

	"gitlab.com/aerth/diamond"
)

func main() {
	var (
		path = flag.String("s", "diamond.s", "path to socket")
	)
	flag.Parse()

	// Establish the connection to the adddress of the
	// RPC server
	client, err := diamond.NewClient(*path)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Perform a procedure call (core.HandlerName == Handler.Execute)
	// with the Request as specified and a pointer to a response
	// to have our response back.
	var (
		req  = new(string)
		resp = new(string)
	)
	if err := client.Call("test.Hello", req, resp); err != nil {
		log.Println(err)
	}
	fmt.Println(*resp)
	if err := client.Call("test.Goodbye", req, resp); err != nil {
		log.Println(err)
	}
	fmt.Println(*resp)

}
