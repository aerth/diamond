package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	diamond "github.com/aerth/diamond/lib"
)

func catchSignals(s *diamond.Server) {
	go func(srv *diamond.Server) {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)
		select {
		case s := <-c:
			println("caught signal:", s.String())
			srv.Runlevel(0)
			os.Exit(0)
		}
	}(s)
}

func runlevel0() error {
	fmt.Println(time.Now(), "demo runlevel 0\ngoodbye!")
	go func() {
		<-time.After(time.Second)
		os.Exit(0)
	}()
	return nil
}

func runlevel1() error {
	fmt.Println(time.Now(), "demo runlevel 1")
	return nil
}

// example type with unexported method 'runlevel3'
type bar struct {
	created time.Time
}

var foo = bar{
	created: time.Now(),
}

// yes, runlevel can be a method!
func (b bar) runlevel3() error {
	fmt.Println(time.Now(), "demo runlevel 3")
	fmt.Println("foo was created at", b.created)
	return nil
}

func main() {

	// create
	srv, err := diamond.New("diamond.socket")
	if err != nil {
		println(err.Error())
		os.Exit(111)
	}

	// setup
	catchSignals(srv)
	srv.Config.Verbose = true
	srv.SetRunlevel(0, runlevel0)
	srv.SetRunlevel(1, runlevel1)
	srv.SetRunlevel(3, foo.runlevel3)

	// begin
	err = srv.Runlevel(1)
	if err != nil {
		println(err.Error())
		os.Exit(111)
	}

	// wait (or do stuff)
	srv.Wait() // wait until runlevel 0

}
