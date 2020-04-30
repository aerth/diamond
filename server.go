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

// Diamond package provides runlevels to an application
//
// API is considered unstable until further notice
//
package diamond

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"time"
)

const (
	SINGLEUSER = 1
	MULTIUSER  = 3
)

// Server  ...
type Server struct {
	socket     net.Listener
	socketname string
	fns        []interface{}
	r          *rpc.Server
	quit       chan error
	listeners  []net.Listener // to suspend during lower runlevels
	runlevel   int
	HookLevel3 func() []net.Listener
	HookLevel4 func() []net.Listener
	cleanup    func() error
	httpPairs  []httpPair
}

type httpPair struct {
	Addr string
	H    http.Handler
}

// New creates a new Server, with a socket at socketpath, and starts listening.
//
// Optional ptrs are pointers to types (`new(t)`) that contain methods
// Each given of given ptrs must satisfy the criteria in the net/rpc package
// See https://godoc.org/net/rpc for these criteria.
func New(socketpath string, fnPointers ...interface{}) (*Server, error) {
	l, err := net.Listen("unix", socketpath)
	if err != nil {
		if strings.Contains(err.Error(), "bind: address already in use") {
			return nil, fmt.Errorf("%v\nDid a diamond server crash? You can delete the socket if you are sure that no other diamond servers are running.", err)
		}
		return nil, err
	}

	s := &Server{
		socket:     l,
		socketname: socketpath,
		fns:        fnPointers,
		quit:       make(chan error),
		cleanup: func() error {
			return os.Remove(socketpath)
		},
	}

	r := rpc.NewServer()
	var pack = &packet{s}
	if err := r.RegisterName("Diamond", pack); err != nil {
		log.Println("err registering rpc name:", err)
	}

	for i := range s.fns {
		if err := r.Register(s.fns[i]); err != nil {
			return nil, err
		}

		typ := reflect.TypeOf(s.fns[i])
		rcvr := reflect.ValueOf(s.fns[i])
		sname := reflect.Indirect(rcvr).Type().Name()
		log.Printf("Registered RPC type: %q", sname)
		for m := 0; m < typ.NumMethod(); m++ {
			method := typ.Method(m)
			mname := method.Name
			log.Printf("\t%s.%s()", sname, mname)
		}

	}
	s.r = r

	go func(s *Server) {
		for {
			conn, err := s.socket.Accept()
			if err != nil {
				log.Println("error:", err)
				continue
			}
			go s.handleConn(conn)
		}
	}(s)

	return s, nil
}

// AddListener listeners can only shutdown a port, not restart, returns total number of listeners (for shutdown)
func (s *Server) AddListener(l net.Listener) int {
	s.listeners = append(s.listeners, l)
	return len(s.listeners)
}

// AddHTTPHandler can restart, returns how many http handlers will be used (for shutdown and restarts)
func (s *Server) AddHTTPHandler(addr string, h http.Handler) int {
	s.httpPairs = append(s.httpPairs, httpPair{addr, h})
	return len(s.httpPairs)
}

// Wait can be called to wait for the program to finish and remove the socket file.
// It is not necessary to call Wait() if your program catches signals
// and cleans up the socket file on it's own.
func (s *Server) Wait() error {
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGHUP, syscall.SIGSTOP)
	var err error
	select {
	case err = <-s.quit:
		if err2 := os.Remove(s.socketname); err2 != nil {
			log.Println("error removing socket:", err2)
		}
	case sig := <-sigs:
		log.Println("recv sig:", sig.String())
		if err2 := os.Remove(s.socketname); err2 != nil {
			log.Println("error removing socket:", err2)
		}
	}
	return err
}
func (s *Server) handleConn(conn net.Conn) {
	// do auth?
	s.r.ServeConn(conn)
	conn.Close()
}

func (s *Server) Runlevel(level int) error {
	if level != 3 && level != 0 {
		panic("can only set Runlevel 3 from go (for now)")
	}
	if level == 0 {
		for i := range s.listeners {
			if err := s.listeners[i].Close(); err != nil {
				log.Printf("error closing listener %d: %v", i, err)
			}
		}
		if err := s.cleanup(); err != nil {
			log.Println("error cleaning up:", err)
		}
		s.runlevel = 0
		return nil
	}
	if s.HookLevel3 == nil && len(s.httpPairs) == 0 {
		panic("cant runlevel 3 with no listeners and no HookLevel3()")
	}
	var listeners []net.Listener
	if s.HookLevel3 != nil {
		listeners = s.HookLevel3()
	}
	for i := range s.httpPairs {
		l, err := net.Listen("tcp", s.httpPairs[i].Addr)
		if err != nil {
			log.Println("error listening:", err)
			continue
		}
		listeners = append(listeners, l)
		handler := &http.Server{
			Handler:        s.httpPairs[i].H,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
			IdleTimeout:    time.Second,
		}
		go func(l net.Listener, srv *http.Server) {
			log.Println(srv.Serve(l))
		}(l, handler)
	}
	s.listeners = append(s.listeners, listeners...)
	log.Printf("new listeners: %d, total listeners: %d", len(listeners), len(s.listeners))
	s.runlevel = 3
	return nil
}

type packet struct {
	parent *Server
}

func (p *packet) HELLO(arg string, reply *string) error {
	log.Printf("HELLO: %q", arg)
	*reply = "HELLO from Diamond Socket"
	return nil
}

func (p *packet) RUNLEVEL(level string, reply *string) error {
	if len(level) != 1 {
		*reply = "need runlevel to switch to (digit)"
		return nil
	}
	if fmt.Sprintf("%d", p.parent.runlevel) == level {
		*reply = "already"
		return nil
	}

	str := "Switching to runlevel " + level

	switch level {
	case "0":
		for i := range p.parent.listeners {
			if err := p.parent.listeners[i].Close(); err != nil {
				log.Printf("error closing listener %d: %v", i, err)
			}
		}
		if err := p.parent.cleanup(); err != nil {
			log.Println(str, err)
		}
		p.parent.runlevel = 0
		os.Exit(0)
	case "1":
		for i := range p.parent.listeners {
			if err := p.parent.listeners[i].Close(); err != nil {
				log.Printf("error closing listener %d: %v", i, err)
			}
		}
		p.parent.listeners = nil
		p.parent.runlevel = 1
		*reply = fmt.Sprintf("level %d", p.parent.runlevel)
		return nil
	case "2", "3":
		if p.parent.HookLevel3 == nil && len(p.parent.httpPairs) == 0 {
			*reply = "no level 3"
			return nil
		}
		var listeners []net.Listener
		if p.parent.HookLevel3 != nil {
			listeners = p.parent.HookLevel3()
		}
		for i := range p.parent.httpPairs {
			l, err := net.Listen("tcp", p.parent.httpPairs[i].Addr)
			if err != nil {
				log.Println("error listening:", err)
				continue
			}
			listeners = append(listeners, l)
			handler := &http.Server{
				Handler:        p.parent.httpPairs[i].H,
				ReadTimeout:    10 * time.Second,
				WriteTimeout:   10 * time.Second,
				MaxHeaderBytes: 1 << 20,
				IdleTimeout:    time.Second,
			}
			go func(l net.Listener, srv *http.Server) {
				log.Println(srv.Serve(l))
			}(l, handler)
		}
		p.parent.listeners = append(p.parent.listeners, listeners...)
		log.Printf("new listeners: %d, total listeners: %d", len(listeners), len(p.parent.listeners))
		p.parent.runlevel = 3
		*reply = fmt.Sprintf("level %d", p.parent.runlevel)
		return nil
	case "4":
		if p.parent.HookLevel4 == nil {
			*reply = "no level 4"
			return nil
		}
		p.parent.listeners = append(p.parent.listeners, p.parent.HookLevel4()...)
		p.parent.runlevel = 4
		*reply = fmt.Sprintf("level %d", p.parent.runlevel)
		return nil
	default:
		log.Println("invalid arg:", level)
		return nil
	}

	return nil

}
