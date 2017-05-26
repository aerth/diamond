package diamond

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

func (s *System) closelisteners() error {
	var errors = make(chan error, len(s.listeners))
	var nl int = len(s.listeners)
	for i := 0; i < nl; i++ {
		s.Log.Println("closing listener:", s.listeners[i].String())
		go func(name string, listener net.Listener) {
			if listener == nil {
				errors <- nil
				return
			}
			err := listener.Close()
			if err != nil {
				if estring := err.Error(); !strings.Contains(estring, "use of closed") {
					errors <- err
					return
				}
			}
			errors <- nil

		}(s.listeners[i].String(), s.listeners[i].listener)

	}
	var nerr = 0
	for i := 0; i < nl; i++ {
		select {
		case err := <-errors:
			if err == nil {
				continue
			}

			s.Log.Println(err)
			nerr++
		}
	}

	if nerr == 0 || s.Config.Force {
		return nil
	}

	// any errors is an error
	return fmt.Errorf("%v errors, check log for details.", len(errors))
}
func (s *System) openlisteners() error {
	var errors = []error{}

	// open listener
	for i := range s.listeners {
		s.Log.Printf("opening %q listener on %q", s.listeners[i].ltype, s.listeners[i].laddr)
		str := s.listeners[i].ltype
		switch str {
		default:
			s.Log.Println(str)
			panic("Listener type incorrect: tcp, unix, tls, got:" + s.listeners[i].ltype)
		// tls
		case "tls":
			panic("no tls, sorry")
		// tcp or unix socket
		case "tcp", "unix":
			l, err := net.Listen(s.listeners[i].ltype, s.listeners[i].laddr)
			if err != nil {
				s.Log.Printf("error opening %s (%s): %v", s.listeners[i].laddr, s.listeners[i].ltype, err)
				errors = append(errors, err)
			} else {
				s.listeners[i].listener = l
				s.Log.Printf("now able to listen (%s) on %s", s.listeners[i].ltype, s.listeners[i].laddr)
				s.Log.Printf("serving http on %s", s.listeners[i].laddr)
				go func(li net.Listener, laddr string) {

					s.Server.Serve(li)
					if err != nil {
						s.Log.Printf("no longer serving http on %s: %s", laddr, err.Error())
					}
				}(l, s.listeners[i].laddr)
			}

		}
	}

	if len(errors) == 0 || s.Config.Force {
		return nil
	}
	// any number of errors is an error
	return fmt.Errorf("%v errors, check log for details.", len(errors))
}

// ConnState closes idle connections, while counting  active connections
// so they don't hang open while switching to runlevel 1
func (s *System) connState(c net.Conn, state http.ConnState) {
	if s.Config.Verbose {
		s.Log.Println(state, c.LocalAddr(), c.RemoteAddr())
	}
	switch state {
	case http.StateActive: // increment counters
		//go s.counters.Up("total", "active")
	case http.StateClosed:
		//go func() { // make the active connections counter a little less boring
		//	<-time.After(durationactive)
		//	s.counters.Down("active")
		//}()
		e := c.Close() // dont wait around to close a connection
		if e != nil {
			s.Log.Println(e)
		}
	case http.StateIdle:
		e := c.Close() // dont wait around for stale clients to close a connection
		if e != nil {
			s.Log.Println(e)
		}
	case http.StateNew:
	default:
		s.Log.Println("Got new alien state:", state.String())
	}
}
