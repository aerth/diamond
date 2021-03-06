// +build ignore

package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/aerth/diamond"
)

func myHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello from diamond example, we must be in runlevel 3 or 4!")
}
func main() {
	s, err := diamond.New("deleteme.socket", new(Thing))
	if err != nil {
		log.Fatalln(err)
	}
	s.HookLevel3 = func() []net.Listener {
		log.Println("LEVEL 3")
		var listeners []net.Listener
		for _, v := range []string{":8080", ":8081"} {
			log.Println("3: listening on", v)
			l, err := net.Listen("tcp", v)
			if err != nil {
				log.Println("error adding listener:", err)
				continue
			}
			log.Println("3: httpd on", v)
			go func() { log.Println(http.Serve(l, http.HandlerFunc(myHandler))) }()
			listeners = append(listeners, l)
		}
		return listeners
	}
	s.HookLevel4 = nil
	log.Fatalln(s.Wait())
}

type Thing struct{}

func (t Thing) TESTONE(in string, out *string) error {
	log.Printf("TESTONE: input=%q", in)
	*out = "it works"
	return nil
}
