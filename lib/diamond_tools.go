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
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var (

	// Debug = true for verbose connection logging
	Debug bool
	// could be improved? let me know!
	preferredCipherSuites = []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	}
)

// how long it takes to count down the connection (for status and CountConnectionsActive)
var durationactive = time.Second * 2

func (s *Server) redirector(destination string) func(w http.ResponseWriter, r *http.Request) {
	if s.Config.RedirectHost == "" {
		s.ErrorLog.Println("RedirectHost is empty. Trouble ahead.")
		return func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, r.URL.Path, 302)
		}
	}
	if strings.Contains(destination, "443") {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://"+s.Config.RedirectHost+r.URL.Path, 302)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+s.Config.RedirectHost+destination+r.URL.Path, 302)
	}
}

// Serve HTTP with one second read timeout (i wonder about large downloads)
func (s *Server) serveHTTP() {
	// handle switched runlevel (?)
	if s.level != 3 {
		s.ErrorLog.Print("Not serving HTTP, not runlevel 3!")
		return
	}

	// handle dead listener
	if !s.Config.NoHTTP && s.listenerTCP == nil {
		s.ErrorLog.Printf("Not serving HTTP, runlevel 3 is already dead (E2)")
		return
	}

	// handle dead tls listener
	if s.Config.UseTLS && s.listenerTLS == nil {
		s.ErrorLog.Printf("Not serving HTTPS, runlevel 3 is already dead (E2)")
		return
	}

	// everything looks good, create listener

	var chosen []net.Listener

	// http listener
	if !s.Config.NoHTTP {

		// RedirectTLS
		if s.Config.RedirectTLS && s.Config.UseTLS && s.listenerTLS != nil {
			srv := &http.Server{Handler: http.HandlerFunc(s.redirector(s.Config.TLSAddr))}
			srv.ReadTimeout = time.Duration(time.Second)
			srv.ConnState = s.connState
			srv.ErrorLog = s.ErrorLog
			s.ErrorLog.Println("REDIRECTING", s.listenerTCP.Addr().String(), "->", s.listenerTLS.Addr().String())
			go srv.Serve(s.listenerTCP) // listen and serve redirector
		} else {
			s.ErrorLog.Println("Listening:", s.listenerTCP.Addr().String())
			chosen = append(chosen, s.listenerTCP)
		}
	}

	// https / tls listener
	if s.Config.UseTLS {
		s.ErrorLog.Println("Listening TLS:", s.listenerTLS.Addr().String())
		chosen = append(chosen, s.listenerTLS)
	}

	if s.Config.SocketHTTP != "" {
		go func() {
			s.ErrorLog.Println("Listening Unix:,", s.Config.SocketHTTP)
			defer s.ErrorLog.Println("Stopped Unix listener:", s.Config.SocketHTTP)
			address := s.Config.SocketHTTP
			defer os.Remove(address)
		ServeUnix:
			// Look up address
			socketAddress, err := net.ResolveUnixAddr("unix", address)
			if err != nil {
				s.ErrorLog.Println(err)
				return
			}
			ulistener, err := net.ListenUnix("unix", socketAddress)
			if err != nil {
				if strings.Contains(err.Error(), "already in use") {
					os.Remove(address)
					goto ServeUnix
				}
				s.ErrorLog.Println(err)
				return
			}
			e := s.Server.Serve(ulistener)
			if e != nil {
				s.ErrorLog.Println(err)
			}
		}()
	}

	// serve loop in goroutine
	for _, listener := range chosen {
		go func(listener net.Listener) {
			if listener == nil {
				s.ErrorLog.Printf("Not listening (E1).")
				return
			}

			name := listener.Addr().String()
			e := s.Server.Serve(listener)
			if e != nil && !strings.Contains(e.Error(), "use of closed network connection") && s.Config.Debug {
				s.ErrorLog.Printf("%s", e)
			}
			s.ErrorLog.Println("Listener stopped:", name)
		}(listener)

	}

	// done
}

// ConnState closes idle connections, while counting  active connections
// so they don't hang open while switching to runlevel 1
func (s *Server) connState(c net.Conn, state http.ConnState) {
	if Debug {
		s.ErrorLog.Println(state, c.LocalAddr(), c.RemoteAddr())
	}
	switch state {
	case http.StateActive: // increment counters
		go s.counters.Up("total", "active")
	case http.StateClosed:
		go func() { // make the active connections counter a little less boring
			<-time.After(durationactive)
			s.counters.Down("active")
		}()
		c.Close() // dont wait around to close a connection
	case http.StateIdle:
		c.Close() // dont wait around for stale clients to close a connection
	case http.StateNew:
	default:
		s.ErrorLog.Println("Got new state:", state.String())
	}
}
