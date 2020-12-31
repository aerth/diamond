/*
* The MIT License (MIT)
*
* Copyright (c) 2016,2017,2020  aerth <aerth@riseup.net>
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

// Package diamond provides runlevels to a web application
//
// API is considered unstable until further notice
//
package diamond

import (
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var log = 0

const (
	// SINGLEUSER mode where we listen only on unix socket
	SINGLEUSER = 1

	// MULTIUSER mode where we listen on network
	MULTIUSER = 3
)

// Server  ...
type Server struct {
	socket     net.Listener
	socketname string
	fns        []interface{}
	r          *rpc.Server
	quit       chan error
	listeners  []net.Listener // to suspend during lower runlevels
	runlevel   *atomic.Value
	rlock      *sync.Mutex

	// HookLevel0 gets called during shift to runlevel 0
	HookLevel0    func() []net.Listener
	HookLevel1    func() []net.Listener
	HookLevel2    func() []net.Listener
	HookLevel3    func() []net.Listener
	HookLevel4    func() []net.Listener
	ServerOptions *http.Server

	// removes socket file
	cleanup func() error

	// http addr:handler pairs
	httpPairs []httpPair

	// standard stdloger
	log *stdlog.Logger
}

type httpPair struct {
	Addr string
	H    http.Handler
}

// Log exports our logger for customization
func (s Server) Log() *stdlog.Logger {
	return s.log
}

// New creates a new Server, with a socket at socketpath, and starts listening.
//
// Optional fnPointers are pointers to types (`new(t)`) that contain methods
// Each given of given ptrs must satisfy the criteria in the net/rpc package
// See https://godoc.org/net/rpc for these criteria.
func New(socketpath string, fnPointers ...interface{}) (*Server, error) {
	l, err := net.Listen("unix", socketpath)
	if err != nil {
		if strings.Contains(err.Error(), "bind: address already in use") {
			return nil, fmt.Errorf("error: %v Did a diamond server crash? You can delete the socket if you are sure that no other diamond servers are running", err)
		}
		return nil, err
	}

	s := &Server{
		socket:     l,
		socketname: socketpath,
		fns:        fnPointers, // keep these around lol
		quit:       make(chan error),
		cleanup: func() error {
			return os.Remove(socketpath)
		},
		log:           stdlog.New(os.Stderr, "[diamond] ", stdlog.LstdFlags),
		runlevel:      new(atomic.Value),
		rlock:         new(sync.Mutex),
		ServerOptions: &http.Server{},
	}
	s.runlevel.Store(0)

	r := rpc.NewServer()
	var pack = &packet{s}
	if err := r.RegisterName("Diamond", pack); err != nil {
		s.log.Println("err registering rpc name:", err)
	}

	// given rpc methods
	for i := range s.fns {
		if err := r.Register(s.fns[i]); err != nil {
			return nil, err
		}

		typ := reflect.TypeOf(s.fns[i])
		rcvr := reflect.ValueOf(s.fns[i])
		sname := reflect.Indirect(rcvr).Type().Name()
		// print type name
		s.log.Printf("Registered RPC type: %q", sname)
		for m := 0; m < typ.NumMethod(); m++ {
			method := typ.Method(m)
			mname := method.Name
			// print func name with type
			s.log.Printf("\t%s.%s()", sname, mname)
		}

	}
	s.r = r
	s.runlevel.Store(1)

	// start listening on the unix socket in a new goroutine
	go func(s *Server) {
		for {
			conn, err := s.socket.Accept()
			if err != nil {
				s.log.Println("error:", err)
				continue
			}
			// handle each new connection in a new goroutine
			go s.handleConn(conn)
		}
	}(s)

	return s, nil
}

// AddListener listeners can only shutdown a port, not restart *yet*,
// returns total number of listeners (for shutdown)
func (s *Server) AddListener(l net.Listener) int {
	s.listeners = append(s.listeners, l)
	return len(s.listeners)
}

// AddHTTPHandler can restart, returns how many http handlers will be used (for shutdown and restarts)
func (s *Server) AddHTTPHandler(addr string, h http.Handler) int {
	s.httpPairs = append(s.httpPairs, httpPair{addr, h})
	return len(s.httpPairs)
}

// Wait for SIGINT, SIGHUP, or runlevel 0.
// When we receive sig, we shift down each gear from current level to zero.
func (s *Server) Wait() error {
	sigs := make(chan os.Signal)

	// sigint and sighup, cant handle sigstop anyways
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGHUP)
	var err error

	// here we press the ctrl+c 3 times
	for sigSmash := 0; sigSmash < 3; sigSmash++ {
		select {
		case err = <-s.quit: // quit request via runlevel 0
			// (via the parent app or via our unix socket)
			if err2 := os.Remove(s.socketname); err2 != nil {
				s.log.Println("error removing socket:", err2)
			}
			break
		case sig := <-sigs: // quit via signal
			cur := s.runlevel.Load().(int)
			s.log.Printf("recv sig: %q, shifting down from runlevel %d", sig.String(), cur)
			for i := cur - 1; i >= 0; i-- {
				if err := s.Runlevel(i); err != nil {
					s.log.Printf("encountered an error during shift to runlevel %d: %v", i, err)
				}
			}
			if err2 := os.Remove(s.socketname); err2 != nil && !os.IsNotExist(err) {
				s.log.Println("error removing socket:", err2)
			}
		}
	}
	return err
}
func (s *Server) handleConn(conn net.Conn) {
	// TODO do auth?
	s.r.ServeConn(conn)
	if err := conn.Close(); err != nil {
		s.log.Printf("error closing unix socket connection: %v", err)
	}
}

// Runlevel changes gears into the selected runlevel.
func (s *Server) Runlevel(level int) error {
	s.rlock.Lock() // runlevel 0 will not unlock
	s.rlock.Unlock()
	// TODO: custom levels past 4
	if 0 > level || level > 4 {
		return fmt.Errorf("invalid level: %d, try 0, 1, 2, 3, 4", level)
	}

	// get current level
	var cur = s.runlevel.Load().(int)
	if cur == level {
		s.log.Printf("warning: already in level %d, will continue...", level)
	}
	s.log.Printf("Entering runlevel %d from %d...", level, cur)
	if cur < 3 {
		wait := closeListeners(s)
		wait()
	}
	switch level {
	case 0:
		s.log.Println("Removing diamond socket...")
		if err := s.cleanup(); err != nil {
			s.log.Println("error cleaning up:", err)
		}
		if s.HookLevel0 != nil {
			s.log.Println("Shutting down gracefully...")
			s.listeners = s.HookLevel0() // could close databases properly etc.
		}
		// to skip this, just have your program not return from HookLevel0.
		<-time.After(time.Second / 2) // lets give a little bit of time
		s.log.Println("")             // done
		os.Exit(0)
	case 1:
		if s.HookLevel1 != nil {
			s.listeners = s.HookLevel1()
		}
	case 2:
		if s.HookLevel2 != nil {
			s.listeners = s.HookLevel2()
		}
	case 3:
		if s.HookLevel3 == nil && len(s.httpPairs) == 0 {
			return fmt.Errorf("cant runlevel 3 with no listeners and no HookLevel3()")
		}
		var listeners []net.Listener
		if s.HookLevel3 != nil {
			listeners = s.HookLevel3()
		}
		// create the http servers
		for i := range s.httpPairs {
			l, err := net.Listen("tcp", s.httpPairs[i].Addr)
			if err != nil {
				s.log.Println("error listening:", err)
				continue
			}
			handler := &http.Server{
				Handler:        s.httpPairs[i].H,
				ReadTimeout:    10 * time.Second,
				WriteTimeout:   10 * time.Second,
				MaxHeaderBytes: 1 << 20,
				IdleTimeout:    time.Second,
				Addr:           "", // this one unused because we create our own listener

				// TODO: other fields if need
				BaseContext:       s.ServerOptions.BaseContext,
				ConnContext:       s.ServerOptions.ConnContext,
				ConnState:         s.ServerOptions.ConnState,
				ErrorLog:          s.log,
				ReadHeaderTimeout: 10 * time.Second,
				TLSConfig:         s.ServerOptions.TLSConfig,
				TLSNextProto:      s.ServerOptions.TLSNextProto,
			}

			// start the http server in a new goroutine
			go func(l net.Listener, srv *http.Server) {
				name := l.Addr().String()
				closeErr := srv.Serve(l)
				if !strings.HasSuffix(closeErr.Error(), "use of closed network connection") {
					s.log.Printf("error while closing server: %q", closeErr.Error())
				} else {
					s.log.Printf("closed listener: %q", name)
				}
			}(l, handler)

			listeners = append(listeners, l)
		}

		// keep these tcp listeners around
		if len(listeners) > 0 {
			s.listeners = append(s.listeners, listeners...)
		}
		s.log.Printf("auto http listeners: %d, total known listeners: %d", len(listeners), len(s.listeners))
	case 4:
		s.log.Println("Entering runlevel 4...")
		if s.HookLevel4 != nil {
			s.listeners = s.HookLevel4()
		}
	}
	s.runlevel.Store(level)
	return nil
}

type packet struct {
	parent *Server
}

func (p *packet) HELLO(arg string, reply *string) error {
	p.parent.log.Printf("HELLO: %q", arg)
	*reply = "HELLO from Diamond Socket"
	return nil
}

func (p *packet) RUNLEVEL(level string, reply *string) error {
	p.parent.log.Printf("Request to shift runlevel: %q", level)
	if len(level) != 1 {
		*reply = "need runlevel to switch to (digit)"
		return nil
	}
	if fmt.Sprintf("%d", p.parent.runlevel) == level {
		*reply = "already"
		return nil
	}

	s := p.parent
	switch level {
	case "0":
		if err := p.parent.Runlevel(0); err != nil {
			s.log.Println(err)
		}
		return nil
	case "1":
		if err := p.parent.Runlevel(1); err != nil {
			s.log.Println(err)
		}
		*reply = fmt.Sprintf("level %d", p.parent.runlevel)
		return nil
	case "2":
		if err := p.parent.Runlevel(2); err != nil {
			s.log.Println(err)
		}
		*reply = fmt.Sprintf("level %d", p.parent.runlevel)
		return nil
	case "3":
		if err := p.parent.Runlevel(3); err != nil {
			s.log.Println(err)
		}
		*reply = fmt.Sprintf("level %d", p.parent.runlevel)
		return nil
	case "4":
		if err := p.parent.Runlevel(4); err != nil {
			s.log.Println(err)
		}
		*reply = fmt.Sprintf("level %d", p.parent.runlevel)
		return nil
	default:
		s.log.Println("invalid arg:", level)
		return nil
	}
}

func closeListeners(s *Server) func() {
	if len(s.listeners) == 0 {
		return func() {}
	}
	s.log.Printf("closing listeners")
	wg := sync.WaitGroup{}
	for i := range s.listeners {
		wg.Add(1)
		if err := s.listeners[i].Close(); err != nil &&
			!strings.HasSuffix(err.Error(), "use of closed network connection") {
			s.log.Printf("error closing listener %d: %v", i, err)
		}
		wg.Done()
	}
	return wg.Wait
}
