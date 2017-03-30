package main

import (
	"fmt"
	"net/http"

	"github.com/aerth/diamond/lib"
	"github.com/julienschmidt/httprouter"
)

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func Hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}

func main() {
	router := httprouter.New()
	router.GET("/", Index)
	router.GET("/hello/:name", Hello)

	s := diamond.NewServer(router)
	
	s.ConfigPath("config.json")
	s.Start()
	select {
	case quitmsg := <- s.Done:
		println(quitmsg)
	}
}
