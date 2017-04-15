// serve single file
package main

import (
	"io/ioutil"
	"net/http"
	"os"

	diamond "github.com/aerth/diamond/lib"
)

var mux http.Handler
var port = ":8033"
var socket = os.TempDir() + "/singlesrv.sock"

func init() {
	if len(os.Args) < 2 {
		println("fatal: need file to serve")
		println("usage: singlesrv <filename>")
		println("example: env PORT=0.0.0.0:8080 singlesrv index.html")
		os.Exit(111)
	}

	filebytes, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		filebytes = []byte(err.Error())
	}
	mux = http.HandlerFunc(
		/* HandlerFunc */
		func(w http.ResponseWriter, r *http.Request) {
			w.Write(filebytes)
		},
	)
}

func main() {
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	if os.Getenv("SOCKET") != "" {
		socket = os.Getenv("SOCKET")
	}
	s := diamond.NewServer(mux)
	s.Config.Addr = port
	err := s.Start()
	if err != nil {
		println(err.Error())
	}
	println(<-s.Done)
}
