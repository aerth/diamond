// +build ignore

package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/aerth/diamond"
)

func myHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello from diamond example, we must be in runlevel 3 or 4!")
}

type h = http.HandlerFunc

func main() {
	s, err := diamond.New("deleteme.socket", new(Thing))
	if err != nil {
		log.Fatalln(err)
	}
	s.AddHTTPHandler(":8080", h(myHandler))
	s.AddHTTPHandler(":8081", h(myHandler))
	s.AddHTTPHandler(":8082", h(myHandler))
	s.Runlevel(3)
	log.Fatalln(s.Wait())
}

type Thing struct{}

func (t Thing) TESTONE(in string, out *string) error {
	log.Printf("TESTONE: input=%q", in)
	*out = "it works"
	return nil
}
