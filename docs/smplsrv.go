// serve single file
package main

import (
	"net/http"
	"io/ioutil"
  "os"
	diamond "github.com/aerth/diamond/lib"
)

func init() {
}

func main() {
  if len(os.Args) < 2 {
    println("fatal: need file to serve")
    println("usage: smplsrv <filename>")
    return
  }
  port := ":8033"
  if os.Getenv("PORT") != "" {
    port = os.Getenv("PORT")
  }

  filebytes, _ := ioutil.ReadFile(os.Args[1])
	mux := http.HandlerFunc(
		/* HandlerFunc */
		func(w http.ResponseWriter, r *http.Request) {
			w.Write(filebytes)
		},
	)
	s := diamond.NewServer(mux)
	s.Config.Addr = port
	err := s.Start()
  if err != nil {
    println(err.Error())
  }
	println(<-s.Done)
}
