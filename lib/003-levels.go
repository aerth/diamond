package diamond

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	stoplisten "github.com/hydrogen18/stoppableListener"
)

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
	s.ErrorLog.Printf("Shifted to runlevel 0")

	if s.listenerSocket == nil {
		s.ErrorLog.Printf("Goodbye!")
		os.Exit(0)
	}

	e := s.listenerSocket.Close()
	if e != nil {
		s.ErrorLog.Printf("%s", e)
	}
	// exit code 0
	s.ErrorLog.Printf("Goodbye!")
	if s.config.log != "stdout" {
		fmt.Println("Goodbye!")
	}
	os.Exit(0)
}

// single user mode
func (s *Server) runlevel1() {
	s.lock.Lock()
	s.runlevel6() // stop listener
	s.level = 1
	time.Sleep(1 * time.Second)
	s.lock.Unlock() 

}

// multiuser mode
func (s *Server) runlevel3() {
	if s.level == 3 {
		if s.config.debug {
			s.ErrorLog.Printf("Already in runlevel 3, switch to runlevel 1 first.")
		}
		return
	}
	s.lock.Lock()

	// not using defer unlock because httpserver will unlock properly.

	if s.config.debug {
		s.ErrorLog.Printf("Entering runlevel 3")
	}

	// start listening on s.config.addr (config or -http flag)
	l, err := net.Listen("tcp", s.config.addr)
	if err != nil {
		s.ErrorLog.Printf("** WARNING **: %s", err)
		s.lock.Unlock()
		return

	}

	// create a new stoppable net.Listener
	sl, err := stoplisten.New(l)
	if err != nil {
		s.ErrorLog.Printf("Can't runlevel3: %s", err)
		s.lock.Unlock()
		return
	}

	s.listenerTCP = sl

	//	s.handlerTCP = s.mux

	s.level = 3

	s.serveHTTP()

}

// restart into single user mode.
// should not be called by anything but other runlevel methods.

func (s *Server) runlevel6() {
	s.level = 6

	// disallow new multiuser connections

	if s.listenerTCP != nil {
		s.ErrorLog.Printf("Closing TCP listener: %s", s.config.addr)
		e := s.listenerTCP.Close()
		if e != nil {
			panic(e)
		}
		s.listenerTCP.Stop()

		//fmt.Println("loopin?")

	}

	if s.listenerTCP != nil {
		s.listenerTCP.TCPListener = nil
		s.listenerTCP = nil
	}

}
