// serve files from single directory
package main

import (
	"net/http"
	"os"

	diamond "github.com/aerth/diamond/lib"
)

var mux http.Handler
var port = ":8034"
var socket = os.TempDir() + "/smplsrv.sock"

func init() {
	if len(os.Args) < 2 {
		println("fatal: need filesystem to serve")
		println("usage: smplsrv <directory>")
		println("set PORT variable to specify port")
		println("'env PORT=0.0.0.0:8034 smplsrv .'")
		os.Exit(111)
	}
}

func main() {
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	if os.Getenv("SOCKET") != "" {
		socket = os.Getenv("SOCKET")
	}
	mux = http.FileServer(http.Dir(os.Args[1]))
	s := diamond.NewServer(mux)
	s.Config.Addr = port
	err := s.Start()
	if err != nil {
		println(err.Error())
	}
	println(<-s.Done)
}
