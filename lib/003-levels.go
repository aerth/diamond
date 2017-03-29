package diamond

import (
	"net"
	"os"
	"strings"
	"time"

	stoplisten "github.com/hydrogen18/stoppableListener"
)

// HookLevels are called at the end of each runlevel
var (
	HookLevel0 = func() {}
	HookLevel1 = func() {}
	HookLevel2 = func() {}
	HookLevel3 = func() {}
	HookLevel4 = func() {}
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
	defer HookLevel0()
	if s.listenerSocket == nil {
		s.ErrorLog.Printf("Socket disappeared")
		return
	}
	e := s.listenerSocket.Close()
	if e != nil {
		s.ErrorLog.Printf("%s", e)
	}
}

// single user mode
func (s *Server) runlevel1() {
	s.lock.Lock()
	s.runlevel6() // stop listener
	s.level = 1
	time.Sleep(1 * time.Second)
	HookLevel1()
	s.lock.Unlock()

}

// multiuser mode
func (s *Server) runlevel3() {
	if s.level == 3 {
		if s.Config.Debug {
			s.ErrorLog.Printf("Already in runlevel 3, switch to runlevel 1 first.")
		}
		return
	}
	s.lock.Lock()

	// not using defer unlock because httpserver will unlock properly.

	if s.Config.Debug {
		s.ErrorLog.Printf("Entering runlevel 3")
	}

	// start listening on s.Config.Addr (config or -http flag)
	l, err := net.Listen("tcp", s.Config.Addr)
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
	HookLevel3()
	s.serveHTTP()

}

/*

004-levels.go

*/

// restart into single user mode.
// should not be called by anything but other runlevel methods.

func (s *Server) runlevel6() {
	s.level = 6

	// disallow new multiuser connections

	if s.listenerTCP != nil {
		s.ErrorLog.Printf("Closing TCP listener: %s", s.Config.Addr)
		e := s.listenerTCP.Close()
		if e != nil {
			panic(e)
		}
		s.listenerTCP.Stop()

	}

	if s.listenerTCP != nil {
		s.listenerTCP.TCPListener = nil
		s.listenerTCP = nil
	}

}
