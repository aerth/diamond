package diamond

import (
	"net"
	"net/http"
	"strings"
	"time"
	//	stoplisten "github.com/hydrogen18/stoppableListener"
)

var RedirectHost = "localhost"

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
		if s.Config.RedirectTLS && s.Config.UseTLS && s.listenerTLS != nil {
<<<<<<< HEAD
			srv := &http.Server{Handler: http.HandlerFunc(s.redirector(s.Config.TLSAddr))}
=======
			srv := &http.Server{Handler: http.HandlerFunc(redirector(s.Config.TLSAddr))}
>>>>>>> a82680d6e2c1a7d63b6d0f1e4b87390c435eb33a
			srv.ReadTimeout = time.Duration(time.Second)
			srv.ConnState = s.connState
			srv.ErrorLog = s.ErrorLog
			s.ErrorLog.Println("Redirecting", s.listenerTCP.Addr().String(), "to", s.listenerTLS.Addr().String())
			go srv.Serve(s.listenerTCP)
		} else {
			s.ErrorLog.Println("Listening:", s.listenerTCP.Addr().String())
			chosen = append(chosen, s.listenerTCP)
		}
	}

	if s.Config.UseTLS {
		s.ErrorLog.Println("Listening TLS:", s.listenerTLS.Addr().String())
		chosen = append(chosen, s.listenerTLS)

	}

	s.lock.Unlock()

	// serve loop in goroutine
	for _, listener := range chosen {
		go func(listener net.Listener) {
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

<<<<<<< HEAD
func (s *Server) redirector(destination string) func(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(destination, "443") {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://"+s.Config.RedirectHost+r.URL.Path, 302)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+s.Config.RedirectHost+destination+r.URL.Path, 302)
=======
func redirector(destination string) func(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(destination, "443") {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://"+RedirectHost+r.URL.Path, 302)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+RedirectHost+destination+r.URL.Path, 302)
>>>>>>> a82680d6e2c1a7d63b6d0f1e4b87390c435eb33a
	}
}

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
