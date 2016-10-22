package diamond

import (
		stoplisten "github.com/hydrogen18/stoppableListener"
	"net"
	"net/http"
	"time"
)

// ServeStatus serves Status report
func (s *Server) ServeStatus(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(s.Status()))
}

// Serve HTTP with one second read timeout (i wonder about large downloads)
// It is only called by runlevel3 function while s.lock is Locked.
func (s *Server) serveHTTP() {
	if s.listenerTCP == nil {
		if s.config.debug {
			s.ErrorLog.Printf("Not serving TCP, runlevel 3 is already dead")
		}
		s.lock.Unlock()
		return
	}

	if s.level != 3 {
		s.ErrorLog.Print("Not serving TCP, not runlevel 3!")
		s.lock.Unlock()
		return
	}

	if s.level != 3 {
		if s.config.debug {
			s.ErrorLog.Printf("Runlevel 3 already dead")

		}
		s.lock.Unlock()
		return
	}

	if s.config.debug {
		s.ErrorLog.Print("HTTP Processing")
	}

	if s.listenerTCP == nil {

		s.lock.Unlock()
		return
	}

	if s.listenerTCP.TCPListener == nil {
		s.lock.Unlock()
		return
	}

	if s.level != 3 {
		s.ErrorLog.Print("Silly")
		s.lock.Unlock()
		return
	}
	go func(listen *stoplisten.StoppableListener) {
		//time.Sleep(100 * time.Millisecond)
		if listen == nil {
			s.ErrorLog.Printf("Not listening (E1).")
			s.lock.Unlock()
			return
		}
		if listen.TCPListener == nil {
			s.ErrorLog.Printf("Not listening (E2).")
			s.lock.Unlock()
			return
		}

		// unlock RIGHT BEFORE SERVING or a telinit 1 could mess this all up
		s.ErrorLog.Printf("Listening on: %s", s.listenerTCP.Addr().String())
		s.lock.Unlock()
		e := s.Server.Serve(listen)
		if e != nil {
			s.ErrorLog.Printf("%s", e)

		} else {
			s.ErrorLog.Printf("Not listening (E3).")

		}

	}(s.listenerTCP)

}

// ConnState closes idle connections,
// so they don't hang open while switching to runlevel 1
func (s *Server) connState(c net.Conn, state http.ConnState) {
	s.counter.Lock()
	if s.numconn < 0 {
		s.numconn = 0
	}
	switch state {
	case http.StateActive: // increment counters
		s.numconn++
		s.allconn++
	case http.StateClosed:
		go func() { // make the active connections counter a little less boring
			time.Sleep(1 * time.Second)
			s.counter.Lock()
			s.numconn--
			s.counter.Unlock()
		}()
		c.Close() // dont wait around to close a connection

	case http.StateIdle:
		c.Close() // dont wait around for stale clients to close a connection
	default:
	}
	s.counter.Unlock()
}
