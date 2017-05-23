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
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const CHMODDIR = 0750
const CHMODFILE = 0750

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
}

type RunlevelFunc func() error

type Options struct {
	Verbose  bool
	Kickable bool
	Kicks    bool
}

func NewServer(socket string) (*Server, error) {
	_, err := os.Stat(socket)
	if err == nil {
		return nil, fmt.Errorf("socket already exists, delete if you want")
	}

	srv := &Server{
		Config:        &Options{},
		Log:           log.New(os.Stderr, "[diamond] ", 0),
		listeners:     nil,
		controlSocket: socket,
		runlevels:     make(map[int]RunlevelFunc),
		done:          make(chan int, 1),
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
	if err = rcpServer.RegisterName("Packet", pack); err != nil {
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
