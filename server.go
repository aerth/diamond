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
	"net/rpc"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
)

// Server  ...
type Server struct {
	socket     net.Listener
	socketname string
	fns        []interface{}
	r          *rpc.Server
	quit       chan error
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

type packet struct {
	parent *Server
}

func (p *packet) HELLO(arg string, reply *string) error {
	log.Printf("HELLO: %q", arg)
	*reply = "HELLO from Diamond Socket"
	return nil
}
