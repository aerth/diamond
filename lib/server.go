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

package diamond

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CHMODDIR by default is user read/write/searchable, group read/writable
var CHMODDIR os.FileMode = 0750

// CHMODFILE (control socket) by default is user/group read/write/exectuable
var CHMODFILE os.FileMode = 0770

// System listens on control socket, controlling listeners and runlevels
type System struct {

	// Config can be configured
	Config *Options

	// Log can be redirected
	Log             *log.Logger
	Server          *http.Server
	listeners       []*listener
	controlSocket   string // path to socket
	controlListener net.Listener
	runlevels       map[int]RunlevelFunc // map[int](func() error)
	level           int                  // current runlevel
	locklevel       sync.Mutex           // runlevel lock only for shifting runlevels
	done            chan int             // end
	httpmux         http.Handler         // has ServeHTTP(w,r) method
}

type listener struct {
	ltype    string
	laddr    string
	listener net.Listener
}

func (l listener) String() string {
	return l.laddr
}

// RunlevelFunc is any function with no arguments that returns an error
// It can be a method, such as `func (f foo) runlevel9000() error {}`
type RunlevelFunc func() error

// Options modify how the diamond system functions
type Options struct {
	// More verbose output
	Verbose bool

	// Able to be KICKed via control socket (same as command 'runlevel 0')
	Kickable bool

	// Will KICK if control socket exists at boot time, replacing socket
	Kicks bool
}

// NewServer returns a new server, and an error if the socket path is not valid
func NewServer(handler http.Handler, socket string) (*System, error) {
	s, e := New(socket)
	if e != nil {
		return nil, e
	}
	s.SetHandler(handler)
	return s, nil
}

// SetHandler for all future connections via http socket or tcp listeners
// This is only useful for web applications and can be safely ignored
func (s *System) SetHandler(h http.Handler) error {

	if l := s.level; l > 1 {
		return fmt.Errorf("need to be in runlevel 1, currently in runlevel %v", l)
	}

	s.Server.Handler = h
	return nil
}

// AddListener to the list of listeners, returning the
func (s *System) AddListener(ltype, laddr string) (n int, err error) {
	if ltype == "" || laddr == "" {
		return len(s.listeners), fmt.Errorf("Empty argument: %q %q", ltype, laddr)
	}
	if s.level > 1 {
		n = len(s.listeners)
		return n, fmt.Errorf("already listening on %v listeners, enter runlevel 1 first", n)
	}
	l := new(listener)
	l.ltype = ltype
	l.laddr = laddr
	s.listeners = append(s.listeners, l)
	n = len(s.listeners)
	return n, nil

}

func (s *System) nothinglisten() (err error) {
	var errors []error
	for i, v := range s.listeners {
		// create listener
		s.listeners[i].listener, err = net.Listen(v.ltype, v.laddr)
		if err != nil {
			errors = append(errors, fmt.Errorf("could not create listener: %v", err))
		}

		if len(errors) == 0 {
			return nil
		}

	}
	return fmt.Errorf(fmt.Sprintln(errors))
}

// Get a listener by index
func (s *System) GetListener(n int) net.Listener {
	if len(s.listeners)-1 < n {
		return nil
	}
	return s.listeners[n].listener
}

func (s *System) NListeners() int {
	return len(s.listeners)
}

// New diamond system, listening at specified socket.
func New(socket string) (*System, error) {
	// does the socket already exist?
	_, err := os.Stat(socket)
	if err == nil {
		// try to KICK. first, create client
		client, err := NewClient(socket)
		if err != nil {
			return nil, fmt.Errorf("socket already exists and client could not be created: %v", err)
		}

		// send the KICK command
		var resp string
		resp, err = client.Send("KICK")
		if err != nil {
			return nil, fmt.Errorf("socket already exists and server isnt responding, delete if you want (error %v)", err)
		}

		// if it works, we get OKAY
		if resp != "OKAY" {
			return nil, fmt.Errorf("socket already exists and server responeded with: %q", resp)
		}

		// response was OKAY, no errors.
		// this means we kicked the old server, and the socket *should* be removed.
		// lets force remove the socket and continue as usual ;)
		err = os.Remove(socket)
		if err != nil {
			return nil, fmt.Errorf("socket already exists and could not be removed: %q", err)
		}
	}

	srv := &System{
		Config:        &Options{},
		Log:           log.New(os.Stderr, "[diamond] ", 0),
		listeners:     nil,
		controlSocket: socket,
		runlevels:     make(map[int]RunlevelFunc),
		done:          make(chan int, 1),
	}
	srv.Server = &http.Server{
		ConnState: srv.connState,
	}
	// create and start listening on socket
	err = srv.listenControlSocket()
	if err != nil {
		return nil, err
	}

	return srv, nil
}

func (s *System) SetRunlevel(level int, fn RunlevelFunc) {
	s.locklevel.Lock()
	defer s.locklevel.Unlock()
	s.runlevels[level] = fn
}

func (s *System) GetRunlevel() int {
	s.locklevel.Lock()
	defer s.locklevel.Unlock()
	return s.level
}

// Runlevel switches gears, into the specified level.
// func main() typically should os.Exit(0) some time after s.Wait()
func (s *System) Runlevel(level int) (err error) {
	s.locklevel.Lock()
	defer s.locklevel.Unlock()
	if s.level == 0 && level != 1 {
		if e := s.closelisteners(); e != nil {
			return e
		}
	}
	if s.level == level {
		return fmt.Errorf("already in runlevel %v", level)
	}
	if fn, ok := s.runlevels[level]; ok {
		err = fn()
		if err != nil {
			return fmt.Errorf("still in runlevel %v, could not switch to %v (%v)", s.level, level, err)
		}

		s.level = level
	}
	switch level {
	default:
	case 0:
		// remove listener sockets if exists
		for _, v := range s.listeners {
			if v.ltype == "unix" {
				s.Log.Println("removing http socket:", v.laddr)
				if e := os.Remove(v.laddr); e != nil {
					s.Log.Printf("error removing socket: %v", e)
				}
			}
		}

		// remove control socket file
		s.Log.Println("removing socket")
		err = os.Remove(s.controlSocket)
		if err != nil {
			s.Log.Println(err)
			s.done <- 111
			return err
		}
		s.done <- 0
		return nil
	case 1:
		// close all connection (TCP or http unix socket)
		return s.closelisteners()
	case 3:
		// open all listeners (TCP or http unix socket)
		return s.openlisteners()
	}

	if s.level == level {
		return nil
	}

	return fmt.Errorf(`runlevel "%v" seems not to exist`, level)
}

func (s *System) listenControlSocket() error {
	path := s.controlSocket
	if err := os.MkdirAll(filepath.Dir(path), CHMODDIR); err != nil {
		return fmt.Errorf("diamond: Could not create service path")
	}
	var err error
	s.controlListener, err = net.Listen("unix", path)
	if err != nil {
		return fmt.Errorf("diamond: Could not listen on unix domain socket %q: %v", path, err)
	}
	err = os.Chmod(path, CHMODFILE)
	if err != nil {
		return fmt.Errorf("diamond: Could not change permissions on socket file: %v", err)
	}

	// start listening
	go func() {
		for {
			e := s.socketAccept()
			if e != nil {
				s.Log.Printf("FATAL ADMIN SOCKET %s", e.Error())
				continue
			}
			if s.Config.Verbose {
				s.Log.Printf("Admin Socket Connection Completed: %s", time.Now().Format(time.Kitchen))
			}

		}
	}()
	return nil
}

// socketAccept one connection on unix socket, offering public methods on the 'packet' type
func (s *System) socketAccept() error {
	conn, err := s.controlListener.Accept()
	if err != nil {
		return fmt.Errorf("diamond: Could not accept connection: %v", err)
	}
	s.Log.Println("Received connection", time.Now().Format(time.Kitchen))
	rcpServer := rpc.NewServer()
	var pack = new(packet)
	pack.parent = s
	if err = rcpServer.RegisterName("Diamond", pack); err != nil {
		return fmt.Errorf("diamond: %s",
			err.Error())
	}
	go func() {
		if conn != nil {
			s.Log.Println("Got conn:", conn.LocalAddr().String())
		}
		rcpServer.ServeConn(conn)
		conn.Close()
	}()

	return nil
}

// Wait until runlevel 0 is finished (after running custom RunlevelFunc 0 and socket is removed)
func (s *System) Wait() int {
	return <-s.done
}
