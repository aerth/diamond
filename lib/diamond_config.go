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
	"net/http"
	"os"
	"time"
)

var (
	// CHMODDIR default permissions for directory create
	CHMODDIR os.FileMode = 0750

	// CHMODFILE default permissions for file create
	CHMODFILE os.FileMode = 0640
)

// ConfigFields as seen in s.Config
type ConfigFields struct {
	Name        string // user friendly name
	Addr        string // :8080 (Short for 0.0.0.0:8080) or 127.0.0.1:8080 (Only localhost)
	Socket      string // path of Socket file to create (/tmp/diamond.sock)
	SocketHTTP  string // if nonempty, listen on unix socket
	Level       int
	Debug       bool
	Kicks       bool // will kick to launch
	Kickable    bool // able to be kicked
	DoCycleTest bool // do 1-3-default cycle at launch

	// ssl options
	NoHTTP      bool // dont listen on HTTP
	UseTLS      bool // also listen on TLS
	RedirectTLS bool // open special handler on 80 that only redirects to 443

	RedirectHost string // which host to redirect to
	TLSAddr      string // TLS Addr required for TLS
	TLSCertFile  string // TLS Certificate file location required for TLS
	TLSKeyFile   string // TLS Key file location required for TLS
}

// Signals by default is true, if we get a signal (such as SIGINT), we switch to runlevel 0
//
// Use s.Signals(false) to catch them yourself (not tested)
//
// Must be called before Start()
//
// Not in config because it should always be true
//
func (s *Server) Signals(catch bool) {
	s.signal = catch
}

// SetMux replaces current handler with 'mux'
func (s *Server) SetMux(mux http.Handler) {
	srv := &http.Server{Handler: mux}
	srv.ReadTimeout = time.Duration(time.Second)
	srv.ConnState = s.connState
	srv.ErrorLog = s.ErrorLog
	s.Server = srv
}
