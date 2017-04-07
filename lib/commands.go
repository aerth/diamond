// +build go1.8

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
	"crypto/tls"
	"fmt"
	"net"
	"net/rpc"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HookLevels are called at the end of each runlevel
var (
	HookLevel0  = func() {}
	DoneMessage = "Reached target shutdown"
	HookLevel1  = func() {}
	HookLevel2  = func() {}
	HookLevel3  = func() {}
	HookLevel4  = func() {}
)

// socketListen sets the Server's listenerSocket or returns an error
func (s *Server) socketListen(path string) error {
	//os.Remove(path)
	if err := os.MkdirAll(filepath.Dir(path), CHMODDIR); err != nil {
		return fmt.Errorf("diamond: Could not create service path")
	}
	var err error
	s.listenerSocket, err = net.Listen("unix", path)
	if err != nil {
		s.ErrorLog.Println("wow", err)
		return fmt.Errorf(
			"diamond: Could not listen on unix domain socket %q: %v",
			path, err,
		)
	}
	err = os.Chmod(path, CHMODFILE)
	if err != nil {
		s.ErrorLog.Println(err)
	}

	return nil
}

// rpcpacket to be sent by socket, has Command method for clients to use.
type rpcpacket struct {
	parent *Server // unexported so that only Command can access
}

// CustomCommander can be reassigned
func (s *Server) CustomCommander(duck func(args string, reply *string) error) {
	s.customCommander = duck
}

// Command to process (RPC via UNIX socket)
func (p *rpcpacket) Command(args string, reply *string) error {
	p.parent.ErrorLog.Printf("ADMIN: %v", args)
	switch {

	// telinit
	case strings.HasPrefix(args, "telinit "):
		if args == "telinit 1" {
			p.parent.telinit <- 1
		} else if args == "telinit 3" {
			p.parent.telinit <- 3
		} else if args == "telinit 4" {
			p.parent.telinit <- 4
		} else if args == "telinit 0" {
			p.parent.telinit <- 0
		}
		*reply = "DONE"

	// hello
	case strings.HasPrefix(args, "HELLO from "):
		p.parent.ErrorLog.Println(args)
		*reply = "HELLO from " + p.parent.Config.Name

	// CUSTOM prefix
	case strings.HasPrefix(args, "CUSTOM "):
		if p.parent.customCommander == nil {
			p.parent.ErrorLog.Println("CUSTOM SOCKETS DISABLED:", args)
			*reply = fmt.Sprintf("not defined: %q", args)
			break
		}
		p.parent.ErrorLog.Println("CUSTOM SOCKET COMMAND:", args)
		args = strings.TrimPrefix(args, "CUSTOM ")
		err := p.parent.customCommander(args, reply)
		if err != nil {
			*reply = fmt.Sprint(*reply, err.Error())
		}

	// status
	case args == "status":
		*reply = p.parent.Status()

	// update
	case args == "update":
		if ToolUpdate == nil {
			*reply = "updating is not enabled"
		} else {
			str, e := ToolUpdate()
			if e != nil {
				str += "\nERROR: " + e.Error()
			}
			*reply = str
		}

	// rebuild
	case args == "rebuild":
		p.parent.ErrorLog.Println("90")
		if ToolRebuild == nil {
			*reply = "rebuilding is not enabled"
		} else {
			p.parent.ErrorLog.Println("B+")
			str, e := ToolRebuild()
			if e != nil {
				str += "\nERROR: " + e.Error()
			}
			p.parent.ErrorLog.Println("A+")
			*reply = str
			p.parent.ErrorLog.Printf("Str %q", str)
		}

	// redeploy
	case args == "redeploy":
		*reply = "Redeploying ⋄"
		p.parent.respawn()
		p.parent.telinit <- 0

		// help
	case args == "help":
		*reply = "Commands: help telinit update* rebuild*"

	// KICK
	case args == "KICK":
		if p.parent.Config.Kickable {
			*reply = "OKAY"
			p.parent.ErrorLog.Println("Got KICKed")
			go func() {
				p.parent.telinit <- 0
			}()
		} else {
			*reply = "NO WAY"
		}

	// undefined command
	default:
		p.parent.ErrorLog.Println("New Socket Command:", args)
	}

	if p.parent.Config.Debug {
		p.parent.ErrorLog.Printf(" CMD: %s", args)
		p.parent.ErrorLog.Printf("REPL: %s", *reply)
	}
	if *reply == "" {
		*reply = fmt.Sprintf("Command not found: %s", args)
	}
	return nil
}

// socketAccept one connection on unix socket, send to Packet processor
func (s *Server) socketAccept() error {
	conn, err := s.listenerSocket.Accept()
	if err != nil {
		if strings.Contains(err.Error(), "use of closed network connection") {
			return fmt.Errorf("closed properly")
		}

		return fmt.Errorf("diamond: Could not accept connection: %v",
			err)

	}
	rcpServer := rpc.NewServer()
	var pack = new(rpcpacket)
	// needed to access Server methods such as Status() by *packet.Command()
	pack.parent = s
	if err = rcpServer.RegisterName("Packet", pack); err != nil {
		return fmt.Errorf("diamond: %s",
			err.Error())
	}
	go func() {
		if conn != nil {
			s.ErrorLog.Println("Got conn:", conn.LocalAddr().String())
		}
		rcpServer.ServeConn(conn)
		conn.Close()
	}()

	return nil
}

// Listen on s.Config.Socket for admin connections
func admin(ch chan int, s *Server) {
	var try = 0
Okay:
	if s.Config.Socket == "" {
		s.ErrorLog.Println("please specify where to create socket")
		s.Runlevel(0)
		return
	}
	if s.Config.Debug {
		s.ErrorLog.Println("SOCKET", s.Config.Socket)
	}
	addr, _ := net.ResolveUnixAddr("unix", s.Config.Socket)
	r, e := net.DialUnix("unix", nil, addr)
	if e != nil {
		if strings.Contains(e.Error(), "no such") {
			try++
			if try > 2 {
				s.ErrorLog.Println("FATAL", e)
				s.Runlevel(0)
				return
			}
		}

		if !strings.Contains(e.Error(), "no such") {
			try++
			if try < 2 {
				if e.Error() != "dial unix "+s.Config.Socket+": connect: connection refused" {
					s.ErrorLog.Printf("%s", e)
				}
				s.ErrorLog.Println("replacing stale socket")
				os.Remove(s.Config.Socket)
				goto Okay
			}

			s.ErrorLog.Printf("%s", e)
			s.ErrorLog.Printf("** WARNING ** Socket exists: %q", s.Config.Socket)
			s.ErrorLog.Printf("You may safely delete it if there are no running Diamond processes.")

			s.Runlevel(0)
			return
		}
	} else {

		s.ErrorLog.Print("Socket exists:" + r.RemoteAddr().String())

		// There is a running Diamond instance.
		if s.Config.Kicks { // We are kicking
			out := s.Kick()
			if out == "OKAY" || out == "unexpected EOF" {
				s.ErrorLog.Print("Kicked a diamond")
				<-time.After(100 * time.Millisecond)
				goto Okay
			}
			s.ErrorLog.Println("There is already a running server with socket: " +
				s.Config.Socket + ". We tried kicking, but it replied with: " + out)
			s.Runlevel(0)
			return
		}
		s.ErrorLog.Println("There is already a running server with socket: " +
			s.Config.Socket +
			". If you want to replace it, use {\"Kick\": true} in Config.")
		return

	}

	e = s.socketListen(s.Config.Socket)
	if e != nil {
		s.ErrorLog.Println("eek:", e)
		s.Runlevel(0)
		return
	}
	s.socketed = true
	ch <- 1
	// We are listening on a UNIX socket
	for {
		e = s.socketAccept()
		if e != nil {
			s.ErrorLog.Printf("SOCKET %s", e.Error())
			return
		}
		if s.Config.Debug {
			s.ErrorLog.Printf("Socket Connection: %s", time.Now().Format(time.Kitchen))
		}

	}
}

func (s *Server) telcom() {
	//s.ErrorLog.Println("\n\n\nTELCOM up\n\n\n")
	go s.signalcatch()
	for {
		select {
		case newlevel := <-s.telinit:
			s.levellock.Lock()
			s.levellock.Unlock()

			switch newlevel {
			case -1:
				//s.ErrorLog.Println("\n\n\nTELCOM down\n\n\n")
				return
			case 0:
				s.ErrorLog.Printf("ENTERING RUNLEVEL 0")
				s.runlevel0()
				go func() {
					s.telinit <- -1 // kill this goroutine
				}()
			case 1:
				s.ErrorLog.Printf("ENTERING RUNLEVEL 1")
				s.runlevel1()

			case 3:
				s.ErrorLog.Printf("ENTERING RUNLEVEL 3")
				s.runlevel3()
				<-time.After(300 * time.Millisecond)
			case 4:
				s.ErrorLog.Printf("ENTERING RUNLEVEL 4")
				s.Runlevel4()
				<-time.After(300 * time.Millisecond)
			default:
				s.ErrorLog.Printf("BAD RUNLEVEL: %v", newlevel)
			}
		}
	}
}

func socketExists(path string) bool {
	_, e := os.Open(path)
	if e != nil {
		if strings.Contains(e.Error(), "no such") {
			return false
		}
	}
	return true
}

// tear down and exit
func (s *Server) runlevel0() {
	s.runlevel6()
	s.level = 0
	defer s.ErrorLog.Println("RUNLEVEL 0 REACHED")
	defer func() { go func() { s.Done <- DoneMessage }() }()
	defer HookLevel0()
	if s.listenerSocket == nil {
		s.ErrorLog.Printf("Socket disappeared before we could close it.")
		return
	}
	e := s.listenerSocket.Close()
	if e != nil {
		s.ErrorLog.Printf("%s", e)
	}
}

// single user mode
func (s *Server) runlevel1() {
	defer s.ErrorLog.Println("RUNLEVEL 1 REACHED")
	s.levellock.Lock()
	s.runlevel6() // stop listeners
	s.level = 1
	HookLevel1()
	s.levellock.Unlock()
}

// multiuser mode
func (s *Server) runlevel3() {
	if s.level == 3 {
		s.ErrorLog.Printf("Already in runlevel 3, switch to runlevel 1 first.")
		return
	}
	s.levellock.Lock()
	defer func() {
		s.levellock.Unlock()
	}()

	// start listening on s.Config.Addr (config or -http flag)
	l, err := net.Listen("tcp", s.Config.Addr)
	if err != nil {
		s.ErrorLog.Printf("** WARNING **: %s\n", err)
		s.ErrorLog.Printf("REVERTING to runlevel: %v\n", s.level)
		return
	}

	defer s.ErrorLog.Println("RUNLEVEL 3 REACHED")
	s.listenerTCP = l

	if s.Config.UseTLS {
		// start listening on s.Config.TLSAddr (config or -http flag)
		cer, err := tls.LoadX509KeyPair(s.Config.TLSCertFile, s.Config.TLSKeyFile)
		if err != nil {
			s.ErrorLog.Printf("** TLS WARNING **: %s\n", err)
			s.ErrorLog.Printf("Reverting to runlevel: %v\n", s.level)
			s.Runlevel(s.level)
			return
		}

		config := &tls.Config{
			Certificates:             []tls.Certificate{cer},
			CipherSuites:             preferredCipherSuites,
			PreferServerCipherSuites: true,
		}

		config.BuildNameToCertificate()
		for i := range config.NameToCertificate {
			_, err = url.Parse(i)
			if err == nil {
				s.Config.RedirectHost = i
			}
		}

		s.ErrorLog.Printf("Found %v TLS Certificates: %q\n", len(config.NameToCertificate), s.Config.RedirectHost)

		tlsl, err := tls.Listen("tcp", s.Config.TLSAddr, config)
		if err != nil {
			s.ErrorLog.Printf("** TLS WARNING **: %s\n", err)
			s.ErrorLog.Printf("Reverting to runlevel: %v\n", s.level)
			s.Runlevel(s.level)
			return
		}

		s.listenerTLS = tlsl
	}

	//	s.handlerTCP = s.mux

	s.level = 3
	HookLevel3()
	s.serveHTTP()
	return
}

/*

004-levels.go

*/

// close TCP listeners
// should not be called by anything but other runlevel methods. (runlevels 0,3)

func (s *Server) runlevel6() {
	// s.levellock is locked
	s.level = 6

	// disallow new multiuser connections

	if s.listenerTCP != nil {
		s.ErrorLog.Printf("Closing TCP listener: %s", s.Config.Addr)
		e := s.listenerTCP.Close()
		if e != nil {
			s.ErrorLog.Println(e)
		}
	}

	if s.listenerTCP != nil {
		s.listenerTCP = nil
	}

	if s.listenerTCP != nil {
		s.ErrorLog.Println("Cant close TLS Listener:", s.listenerTCP.Addr().String())
		s.listenerTLS = nil
	}

	if s.listenerTLS != nil {
		s.ErrorLog.Printf("Closing TLS listener: %s", s.Config.TLSAddr)
		e := s.listenerTLS.Close()
		if e != nil {
			s.ErrorLog.Println(e)
		}
	}

	if s.listenerTLS != nil {
		s.ErrorLog.Println("Cant close TLS Listener:", s.listenerTLS.Addr().String())
		s.listenerTLS = nil
	}

}

// Runlevel4 runs HookLevel4, gets called programatically or after 'telinit <- 4' admin command
func (s *Server) Runlevel4() {
	s.ErrorLog.Println("Entering Custom Runlevel")
	HookLevel4()
}
