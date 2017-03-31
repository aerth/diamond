package diamond

import (
	"net"
	"net/http"
	"time"

//	stoplisten "github.com/hydrogen18/stoppableListener"
)

// Serve HTTP with one second read timeout (i wonder about large downloads)
// It is only called by runlevel3 function while s.lock is Locked.
func (s *Server) serveHTTP() {
	// handle switched runlevel (?)
	if s.level != 3 {
		s.ErrorLog.Print("Not serving HTTP, not runlevel 3!")
		s.lock.Unlock()
		return
	}

	// handle dead listener
	if !s.Config.NoHTTP && s.listenerTCP == nil {
		s.ErrorLog.Printf("Not serving HTTP, runlevel 3 is already dead (E2)")
		s.lock.Unlock()
		return
	}

	// handle dead tls listener
	if s.Config.UseTLS && s.listenerTLS == nil {
		s.ErrorLog.Printf("Not serving HTTPS, runlevel 3 is already dead (E2)")
		s.lock.Unlock()
		return
	}


	var chosen []net.Listener
	if !s.Config.NoHTTP {
		s.ErrorLog.Println("Listening:", s.listenerTCP.Addr().String())
		chosen = append(chosen, s.listenerTCP)

	}

	if s.Config.UseTLS {
		s.ErrorLog.Println("Listening TLS:", s.listenerTLS.Addr().String())
		chosen = append(chosen, s.listenerTLS)

	}

	s.lock.Unlock()

	// serve loop in goroutine
 	for _, listener := range chosen {
				go func(listener net.Listener){
					if listener == nil {
						s.ErrorLog.Printf("Not listening (E1).")
						return
					}

					name := listener.Addr().String()
					e := s.Server.Serve(listener)
					if e != nil {
						s.ErrorLog.Printf("%s", e)
					}
					s.ErrorLog.Println("Listener stopped:", name)
			}(listener)

}

	// done
}












// // Serve HTTP with one second read timeout (i wonder about large downloads)
// // It is only called by runlevel3 function while s.lock is Locked.
// func (s *Server) serveHTTP() {
// 	if s.listenerTCP == nil {
// 		if s.Config.Debug {
// 			s.ErrorLog.Printf("Not serving TCP, runlevel 3 is already dead")
// 		}
// 		s.lock.Unlock()
// 		return
// 	}
//
// 	if s.level != 3 {
// 		s.ErrorLog.Print("Not serving TCP, not runlevel 3!")
// 		s.lock.Unlock()
// 		return
// 	}
//
// 	if s.level != 3 {
// 		if s.Config.Debug {
// 			s.ErrorLog.Printf("Runlevel 3 already dead")
//
// 		}
// 		s.lock.Unlock()
// 		return
// 	}
//
// 	if s.listenerTCP.TCPListener == nil {
// 		s.lock.Unlock()
// 		return
// 	}
//
// 	if s.level != 3 {
// 		s.ErrorLog.Print("Silly")
// 		s.lock.Unlock()
// 		return
// 	}
// 	if s.Config.Debug {
// 		s.ErrorLog.Println("Listening:", s.listenerTCP.Addr().String())
// 	}
// 	go func(listen *stoplisten.StoppableListener) {
// 		//<- time.After(100 * time.Millisecond)
// 		if listen == nil {
// 			s.ErrorLog.Printf("Not listening (E1).")
// 			s.lock.Unlock()
// 			return
// 		}
// 		if listen.TCPListener == nil {
// 			s.ErrorLog.Printf("Not listening (E2).")
// 			s.lock.Unlock()
// 			return
// 		}
//
// 		// unlock RIGHT BEFORE SERVING or a telinit 1 could mess this all up
// 		s.lock.Unlock()
// 		name := listen.Addr().String()
// 		e := s.Server.Serve(listen)
// 		if e != nil {
// 			s.ErrorLog.Printf("%s", e)
// 		}
// 		s.ErrorLog.Println("Listener stopped:", name)
// 	}(s.listenerTCP)
//
// }

// ConnState closes idle connections, while counting  active connections
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
			<-time.After(5 * time.Second)
			s.counter.Lock()
			s.numconn--
			s.counter.Unlock()
		}()
		c.Close() // dont wait around to close a connection

	case http.StateIdle:
		c.Close() // dont wait around for stale clients to close a connection
	case http.StateNew:
	default:
		s.ErrorLog.Println("Got new state:", state.String())
	}
	s.counter.Unlock()
}

// ServeStatus serves Status report
func (s *Server) ServeStatus(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(s.Status()))
}
