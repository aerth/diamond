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
)

func Example() {
	// package main
	// import diamond "github.com/aerth/diamond/lib"
	// func main(){
	//
	// Create new diamond server with predefined http.Handler named mux
	s := diamond.NewServer(mux)

	// Or create new diamond server and assign mux later
	// As long as SetMux is called before Start, mux will be used as handler when in runlevel 3
	s = diamond.NewServer()
	mux = http.DefaultServeMux
	s.SetMux(mux)

	// Default Server Config
	s.Config.Addr = "127.0.0.1:8777"
	s.Config.Kickable = true
	s.Config.Kicks = true
	s.Config.Name = "Diamond ⋄ " + version
	s.Config.Socket = "./diamond.socket"
	s.Config.DoCycleTest = false
	s.Config.Level = 3
	s.Config.Debug = false
	s.Config.NoHTTP = false
	s.Config.RedirectHost = ""
	s.Config.RedirectTLS = false
	s.Config.TLSAddr = ""
	s.Config.TLSCertFile = ""
	s.Config.TLSKeyFile = ""
	s.Config.UseTLS = false

	// Listen on socket, shift to default runlevel
	if err := s.Start(); err != nil {
		println(err.Error())
	}

	// Runlevel 0 shuts down properly, sending a message to s.Done chan string
	bye := <-s.Done
	println(bye)
	// }
}


// just for example to look good
type dx struct{}

var diamond dx
var mux http.Handler

func (d dx) NewServer(mux ...http.Handler) *Server {
	return NewServer(mux...)
}
