// +build ignore

package main

import (
	"log"
	"os"
	"time"

	"github.com/aerth/seconfig"
	"gitlab.com/aerth/diamond"
)

type adminAPI struct {
	dbkey seconfig.Key
}

func (a *adminAPI) Hello(args, response *string) error {
	*response = "hello"
	return nil
}

func (a *adminAPI) Goodbye(args, response *string) error {
	*response = "goodbye"
	go func() {
		<-time.After(time.Second * 3)
		os.Remove("diamond.s")
		os.Exit(111)
	}()
	return nil
}

func LoadDiamond() {
	admin := new(adminAPI)
	d := diamond.New("test", "diamond.s", admin)
	go d.ListenFatal()
}

func main() {
	LoadDiamond()
	// do stuff
	for _ = range time.Tick(time.Second * 10) {
		log.Println(time.Now())
	}
}
