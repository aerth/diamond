// serve files from single directory
package main

import (
	"log"
	"net/http"
	"os"

	diamond "github.com/aerth/diamond"
)

var mux http.Handler
var port = "8034"
var socket = os.TempDir() + "/smplsrv.sock"

func init() {
	if len(os.Args) < 2 {
		println("fatal: need filesystem to serve")
		println("usage: smplsrv <directory>")
		println("set PORT variable to specify port")
		println("'env SOCKET=/tmp/test.socket PORT=8034 smplsrv ~/public_html'")
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
	mux := http.NewServeMux()
	// handle all urls
	mux.Handle("/", http.FileServer(http.Dir(os.Args[1])))
	s, err := diamond.New(socket)
	if err != nil {
		log.Println(err)
		os.Exit(111)
	}
	s.AddHTTPHandler("127.0.0.1:"+port, mux)
	s.Runlevel(3)
	if err := s.Wait(); err != nil {
		log.Fatalln(err)
	}
}
