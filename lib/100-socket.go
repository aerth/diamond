package diamond

import (
	"fmt"
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

// Command to process (RPC via UNIX socket)
func (p *rpcpacket) Command(args string, reply *string) error {
	if p.parent.Config.Debug {
		p.parent.logf("RECV: %s", args)
	}
	switch {
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
	case strings.HasPrefix(args, "HELLO from "):
		p.parent.ErrorLog.Println(args)
		*reply = "HELLO from " + p.parent.Config.Name
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
	case args == "reConfig":
		*reply = "ReConfiguring ⋄"
		conf, e := readconf(p.parent.configpath)
		if e != nil {
			*reply = e.Error()
		}
		p.parent.Config = conf
		// err := p.parent.ReloadConfig()
		// if err != nil {
		// 	*reply = err.Error()
		// }
		cur := p.parent.level
		if cur != 1 {
			p.parent.telinit <- 1 // close http listener
		}
		p.parent.telinit <- cur // reinit current runlevel

	case args == "KICK":
		if p.parent.Config.Kickable {
			*reply = "OKAY"
			p.parent.log("Got KICKed")
			p.parent.telinit <- 0
		} else {
			*reply = "NO WAY"
		}
	}

	if p.parent.Config.Debug {
		p.parent.logf("REPL: %s", *reply)
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
