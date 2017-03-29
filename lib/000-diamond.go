// Package diamond ⋄
package diamond

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	version = "0.2" // Version 0.2
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

	if mux != nil {
		s.mux = mux[0]
	} else {
		s.mux = http.DefaultServeMux
	}

	srv := &http.Server{Handler: s.mux}
	srv.ReadTimeout = time.Duration(time.Second)
	srv.ConnState = s.connState
	srv.ErrorLog = s.ErrorLog
	s.Server = srv

	// default config
	s.Config = ConfigFields{}
	s.Config.Addr = "127.0.0.1:8000"
	s.Config.Kickable = true
	s.Config.Kicks = true
	s.Config.Name = "Diamond ⋄"
	s.Config.Socket = os.TempDir() + "diamond.sock"
	s.Config.DoCycleTest = false
	s.Config.Level = 3
	return s
}

// SetMux server
func (s *Server) SetMux(mux http.Handler) {
	s.mux = mux
	srv := &http.Server{Handler: s.mux}
	srv.ReadTimeout = time.Duration(time.Second)
	srv.ConnState = s.connState
	srv.ErrorLog = s.ErrorLog
	s.Server = srv
}

// SetConfigPath path
func (s *Server) SetConfigPath(path string) {
	s.configpath = path
}

const (
	// CHMODDIR default permissions for directory create
	CHMODDIR = 0750

	// CHMODFILE default permissions for file create
	CHMODFILE = 0640
)

func (s *Server) ReloadConfig() error {
	// load config
	config, e := readconf(s.configpath)
	if e != nil {
		os.MkdirAll(filepath.Dir(s.configpath), CHMODDIR)
		n, err := config.Save(s.configpath)
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, n, "bytes written to", s.configpath)
	}
	s.Config = config

	if s.Config.Debug {
		s.ErrorLog.SetFlags(log.Lshortfile)
	}

	if s.Config.Socket == "" {
		tmpfile, er := ioutil.TempFile(os.TempDir(), "/diamond.Socket-")
		if er != nil {
			return er
		}
		os.Remove(tmpfile.Name())
		s.Config.Socket = tmpfile.Name()
	}
	if s.Config.Name == "" {
		s.Config.Name = "⋄ Diamond"
	}

	if s.Config.Level != 3 && s.Config.Level != 1 {
		s.Config.Level = 1
	}
	return nil
}

// Start the Diamond Construct. Should be done after Configuration.
// End with s.RunLevel(0) to close the socket properly.
func (s *Server) Start() (err error) {
	fmt.Fprintln(os.Stderr, "Diamond Construct ⋄", version, "Initialized")

	err = s.ReloadConfig()
	if err != nil {
		return err
	}
	// Socket listen timeout
	done := make(chan int, 1)
	go admin(done, s) // listen on unix socket
	select {
	case <-done:
		// good
	case <-time.After(3 * time.Second):
		fmt.Fprintln(os.Stderr, "Timeout waiting for UNIX socket to be released")
		os.Exit(2)
	}
	go s.telcom() // launch event handler
	if !s.socketed {
		fmt.Fprintln(os.Stderr, "Could not socket")
		os.Exit(2)
	}

	cycleTest := func() {
		s.ErrorLog.Printf("Cycle test")
		switch s.Config.Level {
		case 1:
			if s.Config.Debug {
				s.ErrorLog.Printf("Testing runlevel 3")
			}
			//	s.telinit <- 3 // test http port is available
		case 3, 4:
			s.telinit <- 1 // go to single user mode first
		default:
			fmt.Fprintln(os.Stderr, "Bad Config: default 'Level' should be 1 or 3 or 4")
			os.Exit(2)
		}
	}

	// If JSON config: "DoCycleTest":1,
	if s.Config.DoCycleTest {
		fmt.Fprintln(os.Stderr, "Doing cycle test")
		cycleTest()
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
	config := new(ConfigFields)
	err := json.Unmarshal(b, config)
	if err != nil {
		return err
	}
	return s.ReloadConfig()

}

func exit(i interface{}) {
	fmt.Fprintln(os.Stderr, i)
	os.Exit(2)
}
