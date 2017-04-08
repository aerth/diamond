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

	3 is multiuser, opening tcp listener(s)

Assuming your http.Handler is named mux, this is how to create a new diamond server:

	s := diamond.NewServer(mux)

Before starting the server, it should be configured. See /config.go



*/
package diamond

import (
	"fmt"
	"github.com/aerth/spawn"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var version = "0.6"

// Server runlevels
//
//   0 = halt (NOT os.Exit(0), call that yourself after s.Done is sent)
//
//   1 = single user mode (default) kills listenerTCPs
//
//   3 = multiuser mode (public http) boots listenerTCPs
type Server struct {

	// s.Server is created immediately before serving in runlevel 3
	Server *http.Server `json:"-"`

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
	listenerTCP net.Listener
	listenerTLS net.Listener

	counters mucount // concurrent map

	signal bool // handle signals like SIGTERM gracefully

}

// ToolUpdate if defined will be called after admin command: 'update', can be DefaultToolUpdate
var ToolUpdate func() (output string, err error)

// ToolRebuild if defined will be called after admin command: 'update', can be DefaultToolRebuild
var ToolRebuild func() (output string, err error)

// ToolUpgrade if defined will be called after admin command: 'upgrade', can be DefaultToolUpgrade
var ToolUpgrade func() (output string, err error)

// NewServer returns a new server, ready to be configured or started.
// s.ErrorLog is a logger ready to use, and switches to log file.
func NewServer(mux ...http.Handler) *Server {
	s := new(Server)

	// initialize things
	s.since = time.Now()
	s.telinit = make(chan int, 1)
	s.counters = mucount{m: make(map[string]uint64)}
	s.ErrorLog = log.New(os.Stdout, "[⋄] ", log.Ltime)
	s.level = 1
	s.signal = true

	if mux == nil {
		mux = []http.Handler{http.DefaultServeMux}
	}
	s.Done = make(chan string, 1)
	s.SetMux(mux[0])

	// default config
	s.Config = ConfigFields{}
	s.Config.Addr = "127.0.0.1:8777"
	s.Config.Kickable = true
	s.Config.Kicks = true
	s.Config.Name = "Diamond ⋄ " + version
	s.Config.Socket = "./diamond.socket"
	s.Config.DoCycleTest = false
	s.Config.Level = 3
	return s
}

// Start the admin socket, and enter runlevel: s.Config.Level
// End with s.RunLevel(0) to close the socket properly.
func (s *Server) Start() (err error) {
	s.ErrorLog.Println("Diamond ⋄", version)
	if s.Config.Debug {
		s.ErrorLog.SetFlags(log.Lshortfile)
	}
	// Socket listen timeout
	getsocket := make(chan int, 1)
	go admin(getsocket, s) // listen on unix socket
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
	 * Use s.Runlevel(0)
	 */

	s.telinit <- s.Config.Level // go to default runlevel
	return nil                  // no errors
}

func (s *Server) signalcatch() {
	if !s.signal {
		return
	}
	quitchan := make(chan os.Signal, 1)
	signal.Notify(quitchan, os.Interrupt, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)
	select {
	case sig := <-quitchan:
		println("Diamond got signal:", sig.String()) // stderr
		s.ErrorLog.Println("Diamond got signal:", sig.String())
		s.Runlevel(0)
	}
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

// DefaultToolUpdate runs the command "git pull origin master" from current working directory.
//
// See "ToolUpdate"
func DefaultToolUpdate() (string, error) { // RPC: update
	cmd := exec.Command("git", "pull", "origin", "master")
	b, er := cmd.CombinedOutput()
	if er != nil {
		return string(b), er
	}
	return string(b), nil
}

// DefaultToolRebuild trys 'bin/build.sh', falls back on 'build.sh'
//
// See "ToolRebuild"
func DefaultToolRebuild() (string, error) { // RPC: rebuild
	buildfile := "bin/build.sh"
	buildargs := ""
	if _, err := os.Open(buildfile); err != nil {
		buildfile = "build.sh"
	}
	cmd := exec.Command(buildfile, buildargs)
	b, er := cmd.CombinedOutput()
	if er != nil {
		return string(b), er
	}
	return string(b), nil
}

// DefaultToolUpgrade runs 'ToolUpdate() && ToolRebuild() '
//
// See "ToolUpgrade"
func DefaultToolUpgrade() (string, error) { // RPC: upgrade
	s, e := ToolUpdate()
	if e != nil {
		return s, e
	}
	s2, e := ToolRebuild()
	if e != nil {
		return s + s2, e
	}
	return s + s2, e
}

func (s *Server) respawn() {
	s.ErrorLog.Printf("Respawning %s", time.Now())
	spawn.Spawn()
}
