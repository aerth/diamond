//This example shows usage with httprouter library
//Build, run, and visit http://127.0.0.1:8080/hello/world

package main

import (
	"fmt"
	"net/http"

	"github.com/aerth/diamond/lib"
	"github.com/julienschmidt/httprouter"
)

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome to Diamond!\nOpen diamond-admin -s diamond.sock\n")
}

func hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}

func main() {
	router := httprouter.New()
	router.GET("/", index)
	router.GET("/hello/:name", hello)

	s := diamond.NewServer(router)
	s.Config.Socket = "diamond.sock"
	s.Config.Addr = ":8080"
	s.Start()
	select {
	case quitmsg := <-s.Done:
		println(quitmsg)
	}
}
