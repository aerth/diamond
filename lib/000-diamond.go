// Package diamond ⋄
package diamond

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"os"
)

var (
	version = "1" // Version 1
)

// NewServer returns a new server, ready to be configured or started.
// s.ErrorLog is a logger ready to use, and switches to log file (defined in config) once fully booted.
func NewServer(mux ...http.Handler) *Server {
	n := new(Server)
	n.since = time.Now()
	n.level = 1
	n.signal = true
	n.telinit = make(chan int, 1)
	n.ErrorLog = log.New(os.Stdout, "[⋄] ", log.Ltime)

	if mux != nil {
		n.mux = mux[0]
	} else {
		n.mux = http.DefaultServeMux
	}

	srv := &http.Server{Handler: n.mux}
	srv.ReadTimeout = time.Duration(time.Second)
	srv.ConnState = n.connState
	srv.ErrorLog = n.ErrorLog
	n.Server = srv

	return n
}

// Start the Diamond Construct. Should be done after Configuration.
// End with s.RunLevel(0) to close the socket properly.
func (s *Server) Start() error {
	fmt.Println("Diamond Construct ⋄", version)
	if !s.configured {
		s.ErrorLog.Print("Diamond started without configuration.")
		config, e := readconf(s.configpath)
		if e != nil {
			s.ErrorLog.Print("Bad config:", e)
			os.Exit(2)
		}

		s.doconfig(config)

	}

	if s.config.debug {
		fmt.Println(s.config)
	}

	//s.quitchan = make(chan os.Signal, 1)
	//signal.Notify(s.quitchan, os.Interrupt, syscall.SIGHUP, syscall.SIGQUIT)

	done := make(chan int, 1)
	go admin(done, s) // listen on unix socket
	select {
	case <-done:
		//
	case <-time.After(3 * time.Second):
		fmt.Println("Timeout waiting for UNIX socket to be released")
		os.Exit(2)
	}
	go s.telcom() // launch event handler
	if !s.socketed {
		fmt.Println("Could not socket")
		os.Exit(2)
	}

	s.ErrorLog.Printf("Cycle test")
	switch s.config.level {
	case 1:
		if s.config.debug {
			s.ErrorLog.Printf("Testing runlevel 3")
		}
		s.telinit <- 3 // test http port is available
	case 3:
		s.telinit <- 1 // go to single user mode first
	default:
		fmt.Println("Bad Config: 'RunLevel' should be 1 or 3")
		os.Exit(2)
	}

	s.telinit <- s.config.level // go to default runlevel

	return nil // no errors
}

// NoSignals prevents the server from listening for signals (like SIGHUP)
// Otherwise, it waits for ctrl+c and exits properly, going to runtime 0
func (s *Server) NoSignals() {
	s.signal = false
}

// Runlevel switches the current diamond server's runlevel
func (s *Server) Runlevel(i int) {
	if i == 0 {
		s.runlevel0()
		return
	}
	s.ErrorLog.Print("Got runlevel:", i)
	s.telinit <- i
	s.ErrorLog.Printf("Send runlevel %v to telinit", i)
}

// ConfigPath adds a path to a JSON file to be used as the config file
func (s *Server) ConfigPath(path string) error {
	conf, e := readconf(path)
	if e != nil {
		return e
	}
	e = parseconf(conf)
	if e != nil {
		return e
	}

	s.configpath = path

	return s.doconfig(conf)
}

// Config a server using json []byte
// If server s is created, and then s.Config(b) is used before Start(), config.json is not read.
// If s.Config(b) is not used, config.json or -config flag will be used.
func (s *Server) Config(b []byte) error {
	config, e := readconfigJSON(b)
	if e != nil {
		return e
	}

	return s.doconfig(config)

}

func exit(i interface{}) {
	fmt.Println(i)
	os.Exit(2)
}
