// +build ignore

package main

import (
	"log"

	"github.com/aerth/diamond"
)

func main() {
	s, err := diamond.New("deleteme.socket", new(Thing))
	if err != nil {
		log.Fatalln(err)
	}
	log.Fatalln(s.Wait())
}

type Thing struct{}

func (t Thing) TESTONE(in string, out *string) error {
	log.Printf("TESTONE: input=%q", in)
	*out = "it works"
	return nil
}
