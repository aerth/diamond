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
	"strings"
	"sync"
	"time"
)

// CHMODDIR by default is user read/write/searchable, group read/writable
var CHMODDIR os.FileMode = 0750

// CHMODFILE (control socket) by default is user/group read/write/exectuable
var CHMODFILE os.FileMode = 0770

// Server listens on control socket, controlling listeners and runlevels
type Server struct {
	Config          *Options
	Log             *log.Logger
	listeners       []net.Listener
	controlSocket   string // path to socket
	controlListener net.Listener
	runlevels       map[int]RunlevelFunc
	level           int
	locklevel       sync.Mutex
	done            chan int
	httpmux         http.Handler
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
func NewServer(handler http.Handler, socket string) (*Server, error) {
	s, e := New(socket)
	if e != nil {
		return nil, e
	}
	s.SetHandler(handler)
	return s, nil
}

// SetHandler for all future connections via http socket or tcp listeners
// This is only useful for web applications and can be safely ignored
func (s *Server) SetHandler(h http.Handler) {
	s.locklevel.Lock()
	defer s.locklevel.Unlock()
	s.httpmux = h
}

// AddListener to the list of listeners, returning the
func (s *Server) AddListener(l net.Listener) (n int, err error) {
	s.locklevel.Lock()
	defer s.locklevel.Unlock()
	if s.level > 1 {
		n = len(s.listeners)
		return n, fmt.Errorf("already listening on %v listeners", n)
	}
	s.listeners = append(s.listeners, l)
	n = len(s.listeners)
	return n, nil
}

// New diamond system, listening at specified socket.
func New(socket string) (*Server, error) {
	_, err := os.Stat(socket)
	if err == nil {
		client, err := NewClient(socket)
		if err != nil {
			return nil, fmt.Errorf("socket already exists and client could not be created: %v", err)
		}
		var resp string
		resp, err = client.Send("KICK")
		if err != nil {
			return nil, fmt.Errorf("socket already exists and server isnt responding, delete if you want (error %v)", err)
		}
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

	srv := &Server{
		Config:        &Options{},
		Log:           log.New(os.Stderr, "[diamond] ", 0),
		listeners:     nil,
		controlSocket: socket,
		runlevels:     make(map[int]RunlevelFunc),
		done:          make(chan int, 1),
		httpmux:       http.NewServeMux(),
	}

	// create and start listening on socket
	err = srv.listenControlSocket()
	if err != nil {
		return nil, err
	}

	return srv, nil
}

func (s *Server) SetRunlevel(level int, fn RunlevelFunc) {
	s.locklevel.Lock()
	defer s.locklevel.Unlock()
	s.runlevels[level] = fn
}

func (s *Server) GetRunlevel() int {
	s.locklevel.Lock()
	defer s.locklevel.Unlock()
	return s.level
}

func (s *Server) Runlevel(level int) error {
	s.locklevel.Lock()
	defer s.locklevel.Unlock()

	switch level {
	default:
	case 0:
		// remove socket file
		if e := os.Remove(s.controlSocket); e != nil {
			s.Log.Println(e)
		}
	}

	if fn, ok := s.runlevels[level]; ok {
		err := fn()
		if err != nil {
			return err
		}
		s.level = level
		return nil

	}

	return fmt.Errorf(`runlevel "%v" does not exist`, level)
}

func (s *Server) listenControlSocket() error {
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
				s.Log.Printf("ADMIN SOCKET %s", e.Error())
				return
			}
			if s.Config.Verbose {
				s.Log.Printf("Admin Socket Connection: %s", time.Now().Format(time.Kitchen))
			}

		}
	}()
	return nil
}

// socketAccept one connection on unix socket, send to Packet processor
func (s *Server) socketAccept() error {
	conn, err := s.controlListener.Accept()
	if err != nil {
		if strings.Contains(err.Error(), "use of closed network connection") {
			return fmt.Errorf("closed properly")
		}
		return fmt.Errorf("diamond: Could not accept connection: %v", err)
	}
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

func (s *Server) Wait() {
	<-s.done
}
