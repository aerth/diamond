package diamond

import (
	"fmt"
	"net"
	"net/http"
)

func (s *Server) closelisteners() error {
	var errors = []error{}
	for i := range s.listeners {

		// close listener if exists
		if s.listeners[i].listener != nil {
			s.Log.Println("closing listener:", s.listeners[i].String())
			err := s.listeners[i].listener.Close()
			if err != nil {
				// log error while its easy
				s.Log.Println("error closing", s.listeners[i].String()+":", err)
				errors = append(errors, err)
			}
		}
	}
	if len(errors) == 0 {
		return nil
	}

	// any errors is an error
	return fmt.Errorf("%v errors, check log for details.", len(errors))
}
func (s *Server) openlisteners() error {
	var errors = []error{}

	// open listener
	for i := range s.listeners {
		s.Log.Printf("opening %q listener on %q", s.listeners[i].ltype, s.listeners[i].laddr)
		switch s.listeners[i].ltype {
		default:
			panic("wtf")
		// tls
		case "tls":
			panic("no tls, sorry")
		// tcp or unix socket
		case "tcp", "unix":
			l, err := net.Listen(s.listeners[i].ltype, s.listeners[i].laddr)
			if err != nil {
				panic(err)
				s.Log.Printf("error opening %s (%s): %v", s.listeners[i].laddr, s.listeners[i].ltype, err)
				errors = append(errors, err)
			} else {
				s.listeners[i].listener = l
				s.Log.Printf("now able to listen (%s) on %s", s.listeners[i].ltype, s.listeners[i].laddr)
				if s.httpmux != nil {
					go func() {
						s.Log.Printf("serving http on %s", s.listeners[i].laddr)
						err := http.Serve(l, s.httpmux)
						if err != nil {
							println(err.Error())
						}
					}()
				}
			}

		}
	}
	// any number of errors is an error
	if len(errors) != 0 {
		return fmt.Errorf("%v errors, check log for details.", len(errors))
	}
	return nil
}
