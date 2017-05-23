/*
* The MIT License (MIT)
*
* Copyright (c) 2016,2017  aerth <aerth@riseup.net>
*
* Permission is hereby granted, free of charge, to any person obtaining a copy
* of this software and associated documentation files (the "Software"), to deal
* in the Software without restriction, including without limitation the rights
* to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
* copies of the Software, and to permit persons to whom the Software is
* furnished to do so, subject to the following conditions:
*
* The above copyright notice and this permission notice shall be included in all
* copies or substantial portions of the Software.
*
* THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
* IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
* FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
* AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
* LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
* OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
* SOFTWARE.
 */

// Package diamond adds runlevels to a web application
/*

	0 is off

	1 is single user, using a unix socket for admin commands

	3 is multiuser, opening tcp listener(s) (http, https, http on unix socket)

Assuming your http.Handler is named mux, this is how to create a new diamond server:

	s := diamond.NewServer(mux)

Before starting the server, it should be configured. See 'godoc github.com/aerth/diamond/lib ConfigFields'



*/
package diamond

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/aerth/spawn"
)

var version = "0.6.1"

// Server runlevels
//
//   0 = halt (NOT os.Exit(0), call that yourself after s.Done is sent)
//
//   1 = single user mode (default) kills listenerTCPs
//
//   3 = multiuser mode (public http) boots listenerTCPs
type Server struct {
	Server *http.Server `json:"-"`

	Context *context.Context `json:"-"` // unimplemented

	// ErrorLog is a log.Logger and has SetOutput, SetFlags, and SetPrefix etc.
	ErrorLog *log.Logger `json:"-"`

	Config ConfigFields

	// Usage of s.Done is as follows:
	//
	//   s := diamond.NewServer()
	//   s.Start()
	//   select {
	// 	  case <- s.Done // reached runlevel 0
	//   }
	//
	Done chan string `json:"-"`

	Mode runmode

	// boot time, used for uptime duration
	since time.Time

	// current runlevel
	level     int
	levellock sync.Mutex // guards only shifting between runlevels
	telinit   chan int   // accepts runlevel requests

	// Socket listener that accepts admin commands
	listenerSocket  net.Listener
	socketed        bool // true if we have started listening on a socket
	customCommander func(args string, reply *string) error

	// TCP Listener that can be stopped
	listenerTCP  net.Listener
	listenerTLS  net.Listener
	listenerUnix net.Listener
	counters     mucount // safe concurrent map
	deferred     func()
	signal       bool      // handle signals like SIGTERM gracefully
	once         sync.Once // do socket once
}

type runmode uint8

// Development ...
const Development = runmode(0)

// Production ...
const Production = runmode(1)

// ToolUpdate if defined will be called after admin command: 'update', can be DefaultToolUpdate
// Returned output and err will be sent to admin as socket reply
var ToolUpdate func() (output string, err error)

// ToolRebuild if defined will be called after admin command: 'update', can be DefaultToolRebuild
// Returned output and err will be sent to admin as socket reply
var ToolRebuild func() (output string, err error)

// ToolUpgrade if defined will be called after admin command: 'upgrade', can be DefaultToolUpgrade
// Returned output and err will be sent to admin as socket reply
var ToolUpgrade func() (output string, err error)

// NewServer returns a new server, ready to be configured or started.
// s.ErrorLog is a logger ready to use, and switches to log file.
func NewServer(mux ...http.Handler) *Server {

	// mux was not given, give default
	if mux == nil {
		mux = []http.Handler{http.DefaultServeMux}
	}

	// log to stdout by default
	elog := log.New(os.Stdout, "[⋄] ", log.Ltime)
	counter := mucount{m: make(map[string]uint64)}
	return &Server{
		deferred: func() {},
		since:    time.Now(),
		telinit:  make(chan int, 1),
		counters: counter,
		level:    1,
		signal:   true,
		ErrorLog: elog,
		Done:     make(chan string, 1),
		// s.Server
		Server: &http.Server{
			ErrorLog:    elog,
			Handler:     mux[0],
			ReadTimeout: time.Second,
			ConnState: func(c net.Conn, state http.ConnState) {
				if Debug {
					elog.Println(state, c.LocalAddr(), c.RemoteAddr())
				}
				switch state {
				case http.StateActive: // increment counters
					go counter.Up("total", "active")
				case http.StateClosed:
					go func() { // make the active connections counter a little less boring
						<-time.After(durationactive)
						counter.Down("active")
					}()
					c.Close() // dont wait around to close a connection
				case http.StateIdle:
					c.Close() // dont wait around for stale clients to close a connection
				case http.StateNew:
				default:
					elog.Println("Got new state:", state.String())
				}
			},
		},
		// s.Config
		Config: ConfigFields{
			Addr:        "127.0.0.1:8777",
			Kickable:    true,
			Kicks:       true,
			Name:        "Diamond ⋄ " + version,
			Socket:      "./diamond.socket",
			DoCycleTest: false,
			Level:       3,
		},
	}
}

// Start the admin socket, and enter runlevel: s.Config.Level
// End with s.RunLevel(0) to close the socket properly.
func (s *Server) Start() (err error) {
	s.ErrorLog.Println("Diamond ⋄", version)
	if s.Config.Debug {
		s.ErrorLog.SetFlags(log.Lshortfile)
	}
	getsocket := admin(s)
	go s.once.Do(func() {
		// Socket listen timeout
		go s.signalcatch()
	})

	// timeout waiting for socket
	select {
	case <-getsocket:
		// good
	case <-time.After(3 * time.Second):
		err := fmt.Errorf("fatal: timeout waiting for socket")
		s.ErrorLog.Println(err)
		return err
	}

	go s.telcom() // launch telinit handler
	if !s.socketed {
		err := fmt.Errorf("fatal: could not socket")
		s.ErrorLog.Println(err)
		return err
	}

	/*
	 * WARNING: No os.Exit() beyond this point
	 * Use s.Runlevel(0) to exit and clean up properly
	 */
	if s.Config.Level == 0 {
		s.ErrorLog.Println("Default level 0 -> 1")
		s.Config.Level = 1
	}

	s.telinit <- s.Config.Level // go to default runlevel
	return nil                  // no errors
}

func (s *Server) signalcatch() {
	if !s.signal {
		return
	}
	quitchan := make(chan os.Signal, 1)

	//signal.Notify(quitchan, os.Interrupt, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)
	signal.Notify(quitchan) // all signals
	sig := <-quitchan
	// output to stderr
	println("Diamond got signal:", sig.String()) // stderr

	// output to log, (may be stderr, in which case there is a duplicate line, thats okay.)
	s.ErrorLog.Println("Diamond got signal:", sig.String())

	// enter runlevel 0, calling deferred functions before HookLevel0
	s.Runlevel(0)
	return
}

// Human readable
func listnstr(i int) string {
	if i >= 3 {
		return "Listening"
	}
	return "Not Listening"
}

// mucount is a map[string]uint64 counter
type mucount struct {
	m  map[string]uint64
	mu sync.Mutex // guards map
}

func (m *mucount) Up(t ...string) (current uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, i := range t {
		m.m[i]++
		current = m.m[i]
	}
	return
}
func (m *mucount) Zero(t ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, i := range t {
		m.m[i] = 0
	}
	return
}

func (m *mucount) Down(t ...string) (current uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, i := range t {
		if m.m[i] >= 1 {
			m.m[i]--
		}
		current = m.m[i]
	}
	return
}

func (m *mucount) Uint64(t string) (current uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	current = m.m[t]
	return
}

func (s *Server) respawn() {
	s.ErrorLog.Printf("Respawning %s", time.Now())
	spawn.Spawn()
}

// Status returns a status report string
func (s *Server) Status() string {
	if s == nil {
		return ""
	}
	var out string
	out += fmt.Sprintf("Server Name: %s\n", s.Config.Name)
	out += fmt.Sprintf("Diamond Version: %s\n", version)
	out += fmt.Sprintf("Default Runlevel: %v\n", s.Config.Level)

	s.levellock.Lock()
	out += fmt.Sprintf("Current Runlevel: %v\n", s.level)
	str := listnstr(s.level)
	s.levellock.Unlock()
	if s.Config.Addr != "" {
		out += fmt.Sprintf("Addr: %s (%s)\n", s.Config.Addr, str)
	}
	if s.Config.TLSAddr != "" {
		out += fmt.Sprintf("TLS Addr: %s (%s)\n", s.Config.TLSAddr, str)
	}
	if s.Config.SocketHTTP != "" {
		out += fmt.Sprintf("Socket Addr: %s (%s)\n", s.Config.SocketHTTP, str)
	}
	if s.Config.Addr == "" && s.Config.TLSAddr == "" && s.Config.SocketHTTP == "" {
		out += "Not listening: no listeners in config"
	}
	out += fmt.Sprintf("Uptime: %s\n", time.Since(s.since))
	out += fmt.Sprintf("Recent Connections: %v\n", s.counters.Uint64("active"))
	out += fmt.Sprintf("Total Connections: %v\n", s.counters.Uint64("total"))

	if s.Config.Debug {
		out += fmt.Sprintf("Debug: %v\n", s.Config.Debug)
		exe, wd, args := spawn.Exe()
		if wd != "" {
			out += fmt.Sprintf("Working Directory: %s\n", wd)
		}
		if exe != "" {
			out += fmt.Sprintf("Executable: %s\n", exe)
		}
		if len(args) > 0 {
			out += fmt.Sprintf("Arguments: %s\n", args)
		}

	}
	return out
}
