// serve files from single directory
package main

import (
	"log"
	"net/http"
	"os"

	diamond "github.com/aerth/diamond/lib"
)

var mux http.Handler
var port = "8034"
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
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(os.Args[1])))
	s, err := diamond.NewServer(mux, "smplsrv.control")
	if err != nil {
		log.Println(err)
		os.Exit(111)
	}

	_, err = s.AddListener("tcp", ":"+port)
	if err != nil {
		s.Log.Println(err)
		os.Exit(111)
	}
	_, err = s.AddListener("unix", socket)
	if err != nil {
		s.Log.Println(err)
		os.Exit(111)
	}
	s.Runlevel(3)
	os.Exit(s.Wait())

}
