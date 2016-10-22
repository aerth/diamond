package diamond

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// socketListen sets the Server's listenerSocket or returns an error
func (s *Server) socketListen(path string) error {
	//os.Remove(path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("diamond: Could not create service path")
	}
	var err error
	s.listenerSocket, err = net.Listen("unix", path)
	if err != nil {
		return fmt.Errorf(
			"diamond: Could not listen on unix domain socket %q: %v",
			path, err,
		)
	}
	return nil
}

// rpcpacket to be sent by socket, has Command method for clients to use.
type rpcpacket struct {
	// Name   string
	// Body   string
	parent *Server // unexported so that only Command can access
}

// Command to process (RPC via UNIX socket)
func (p *rpcpacket) Command(args string, reply *string) error {
	if p.parent.config.debug {
		p.parent.logf("RECV: %s", args)
	}
	switch {
	case strings.HasPrefix(args, "telinit "):
		if args == "telinit 1" {
			p.parent.telinit <- 1
		} else if args == "telinit 3" {
			p.parent.telinit <- 3
		} else if args == "telinit 0" {
			p.parent.telinit <- 0
		}

		*reply = "DONE"
	case strings.HasPrefix(args, "HELLO from "):
		p.parent.ErrorLog.Println(args)
		*reply = "HELLO from " + p.parent.config.name
	case args == "status":
		*reply = p.parent.Status()
	case args == "update":
		str, e := upgGitPull()
		if e != nil {
			str += "\nERROR: " + e.Error()
		}
		*reply = str
	case args == "upgrade":
		str, e := upgrade()
		if e != nil {
			str += "\nERROR: " + e.Error()
		}
		*reply = str
	case args == "rebuild":
		str, e := upgMake()
		if e != nil {
			str += "\nERROR: " + e.Error()
		}
		*reply = str
	case args == "redeploy":
		*reply = "Redeploying ⋄"
		p.parent.respawn()
		p.parent.telinit <- 0
	case args == "reconfig":
		*reply = "Reconfiguring ⋄"
		p.parent.configured = false
		conf, e := readconf(p.parent.configpath)
		if e != nil {
			*reply = e.Error()
		}
		p.parent.doconfig(conf)
		cur := p.parent.level
		if cur != 1 {
			p.parent.telinit <- 1 // close http listener
		}
		p.parent.telinit <- cur // reinit current runlevel

	case args == "KICK":
		if p.parent.config.kickable {
			*reply = "OKAY"
			p.parent.log("Got KICKed")
			p.parent.telinit <- 0
		} else {
			*reply = "NO WAY"
		}
	}

	if p.parent.config.debug {
		p.parent.logf("REPL: %s", *reply)
	}
	return nil
}

// socketAccept one connection on unix socket, send to Packet processor
func (s *Server) socketAccept() error {
	conn, err := s.listenerSocket.Accept()
	if err != nil {
		if strings.Contains(err.Error(), "use of closed network connection") {
			return fmt.Errorf("Not listening on UNIX socket.")
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
		rcpServer.ServeConn(conn)
		conn.Close()
	}()

	return nil
}

// Listen on s.config.socket for admin connections
func admin(ch chan int, s *Server) {
Okay:
	if s.config.socket == "" {
		exit("Blank socket")
		return
	}
	addr, _ := net.ResolveUnixAddr("unix", s.config.socket)
	r, e := net.DialUnix("unix", nil, addr)
	if e != nil {
		if !strings.Contains(e.Error(), "no such") {
			s.ErrorLog.Printf("** WARNING ** Socket exists: %q", s.config.socket)
			s.ErrorLog.Printf("You may safely delete it if there are no running Diamond processes.")
			if s.config.debug {
				s.ErrorLog.Printf("%s", e)
			}
			os.Exit(2)
		}
	} else {
		if s.config.debug {
			s.ErrorLog.Print("Socket exists:" + r.RemoteAddr().String())
		}
		// There is a running Diamond instance.
		if s.config.kicks { // We are kicking
			out := s.kickDiamond()
			if out == "OKAY" || out == "unexpected EOF" {
				s.ErrorLog.Print("Kicked a diamond")
				time.Sleep(100 * time.Millisecond)
				goto Okay
			}
			exit("There is already a running server with socket: " +
				s.config.socket + ". We tried kicking, but it replied with: " + out)
		}
		exit("There is already a running server with socket: " +
			s.config.socket +
			". If you want to replace it, use {\"Kick\": true} in config.")

	}
	e = s.socketListen(s.config.socket)
	if e != nil {
		log.Println("eek:", e)
		panic(e)
	}
	s.socketed = true
	ch <- 1
	// We are listening on a UNIX socket
	for {
		e = s.socketAccept()
		if e != nil {
			if strings.Contains(e.Error(),
				"use of closed network connection") && s.level == 0 {
				return
			}
			if s.config.debug {
				s.ErrorLog.Printf("[socket] %s", e.Error())
			}
			return
		}
		if s.config.debug {
			s.ErrorLog.Printf("Socket Connection: %s", time.Now().Format(time.Kitchen))
		}

	}
}
