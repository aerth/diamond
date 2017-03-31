// Package diamond ⋄
package diamond

import (
	"encoding/json"
	//	"io/ioutil"
	"log"
	"net/http"
	"os"
	//"path/filepath"
	"time"
)

var (
	// Version 0.4
	version = "0.4"

	// CHMODDIR default permissions for directory create
	CHMODDIR os.FileMode = 0750

	// CHMODFILE default permissions for file create
	CHMODFILE os.FileMode = 0640
)

// NewServer returns a new server, ready to be configured or started.
// s.ErrorLog is a logger ready to use, and switches to log file.
func NewServer(mux ...http.Handler) *Server {
	s := new(Server)
	s.since = time.Now()
	s.level = 1
	s.signal = true
	s.telinit = make(chan int, 1)
	s.ErrorLog = log.New(os.Stdout, "[⋄] ", log.Ltime)

	if mux == nil {
		mux = []http.Handler{http.DefaultServeMux}
	}
	s.Done = make(chan string, 1)
	s.SetMux(mux[0])

	// default config
	s.Config = ConfigFields{}
	s.Config.Addr = "127.0.0.1:8000"
	s.Config.Kickable = true
	s.Config.Kicks = true
	s.Config.Name = "Diamond ⋄ 4"
	s.Config.Socket = os.TempDir() + "/diamond.sock"
	s.Config.DoCycleTest = false
	s.Config.Level = 3
	return s
}

// SetMux server
func (s *Server) SetMux(mux http.Handler) {
	srv := &http.Server{Handler: mux}
	srv.ReadTimeout = time.Duration(time.Second)
	srv.ConnState = s.connState
	srv.ErrorLog = s.ErrorLog
	s.Server = srv
}

// SetConfigPath path
func (s *Server) SetConfigPath(path string) {
	s.configpath = path
}

// Start the Diamond Construct. Should be done after Configuration.
// End with s.RunLevel(0) to close the socket properly.
func (s *Server) Start() (err error) {
	s.ErrorLog.Println("Diamond Construct ⋄", version, "Initialized")
	// Socket listen timeout
	done := make(chan int, 1)

	go admin(done, s) // listen on unix socket
	select {
	case <-done:
		// good
	case <-time.After(3 * time.Second):
		s.ErrorLog.Println("timeout waiting for socket")
		os.Exit(2)
	}
	go s.telcom() // launch event handler
	if !s.socketed {
		s.ErrorLog.Println("could not socket")
		os.Exit(2)
	}

	/*
	 * WARNING: No os.Exit() beyond this point
	 * Use s.Runlevel(0)
	 */

	cycleTest := func() string {
		s.ErrorLog.Printf("Cycle test")
		switch s.Config.Level {
		case 1:
		case 3, 4:
			s.telinit <- 1 // go to single user mode first
		default:
			return "bad default level"
		}
		return ""
	}

	if s.Config.DoCycleTest {
		s.ErrorLog.Println("Doing cycle test")
		if got := cycleTest(); got != "gold" {
			s.ErrorLog.Println(got)
			s.Runlevel(0)
		}
	}

	s.telinit <- s.Config.Level // go to default runlevel

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
	s.ErrorLog.Print("Got runlevel:", i, "[done]")

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
	return nil
}

// Configure a server using json []byte
// If server s is created, and then s.Config(b) is used before Start(), config.json is not read.
// If s.Config(b) is not used, config.json or -config flag will be used.
func (s *Server) Configure(b []byte) error {
	var config ConfigFields
	err := json.Unmarshal(b, &config)
	if err != nil {
		return err
	}
	s.Config = config
	return nil
}
