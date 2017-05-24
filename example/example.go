package main

import (
	"fmt"
	"net"
	"net/http"
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

func (b bar) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))

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

	// Add listeners
	// Listen on TCP port 2000 on all interfaces.
	l1, err := net.Listen("tcp", ":2000")
	if err != nil {
		srv.Log.Println(err)
		srv.Runlevel(0)
		return
	}
	l2, err := net.Listen("unix", "web.socket")
	if err != nil {
		srv.Log.Println(err)
		srv.Runlevel(0)
		return
	}

	// diamond will close listener
	srv.AddListener(l1)
	srv.AddListener(l2)
	srv.SetHandler(foo)
	// begin
	err = srv.Runlevel(1)
	if err != nil {
		println(err.Error())
		os.Exit(111)
	}

	// wait (or do stuff)
	srv.Wait() // wait until runlevel 0

}
