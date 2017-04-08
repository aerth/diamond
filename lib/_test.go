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
	"strings"
	"testing"
	"time"
)

import "log"

// just for example
type dx struct{}

var diamond dx
var mux http.Handler

func (d dx) NewServer(mux ...http.Handler) *Server {
	return NewServer(mux...)
}

var s = NewServer()
var testAddr = "127.0.0.1:30000"
var testClient *Client
var testSocket = "./delete-me"

func init() {

	log.SetFlags(log.Lshortfile)
	var err error
	testClient, err = NewClient(testSocket)
	if err != nil {
		panic(err)
	}

	s.ErrorLog.SetFlags(log.Lshortfile)
}

func up() error {
	reply, err := testClient.Send("status")
	if reply != "" {
		log.Println("Status Reply:\n" + reply)
	}
	return err
}

func TestNewServer(t *testing.T) {
	s.Config.Socket = testSocket
	s.Config.Addr = testAddr
	err := s.Start()
	if err != nil {
		log.Println(err.Error())
		t.FailNow()
	}

	if err := up(); err != nil {
		log.Println("Wanted server to be up, got down", err)
		t.FailNow()
	}

	// quit properly
	s.Runlevel(0)

	// check if quit
	if nonzero := s.Level(); nonzero != 0 {
		log.Println(nonzero)
		t.FailNow()
	}

	if err := up(); err == nil {
		log.Println("Expected not to be up, got up")
		t.FailNow()
	}

}

func TestKick(t *testing.T) {
	s.Config.Socket = testSocket
	s.Config.Name = "TestKick"
	s.ErrorLog.SetPrefix("TestKick: ")
	err := s.Start()
	if err != nil {
		log.Println(err.Error())
		t.FailNow()
	}

	if err = up(); err != nil {
		log.Println("Wanted server to be up, got down", err)
		t.FailNow()
	}

	reply, err := testClient.Send("KICK")
	if reply != "OKAY" {
		log.Printf("Wanted \"OKAY\", got %q\n", reply)
		t.FailNow()
	}
	if err != nil {
		log.Println("error:", err.Error())
		t.FailNow()
	}

	t1 := time.Now()
	for i, err := 0, up(); err != nil && i < 10; i++ {
		<-time.After(300 * time.Millisecond)
		log.Printf("Attempt %v, wanted server to be down, got up.", i)
		t.Fail()
	}
	log.Printf("Took %s to go down", time.Now().Sub(t1))
	<-time.After(time.Second)

}

func TestRunlevel1(t *testing.T) {
	s.Config.Level = 1
	s.Config.Name = "TestRunlevel1"
	s.ErrorLog.SetPrefix("TestRunlevel1: ")
	s.Config.Addr = testAddr
	s.Config.Socket = testSocket
	err := s.Start()
	if err != nil {
		log.Println(err.Error())
		t.FailNow()
	}
	reply, err := testClient.Send("status")
	if err != nil {
		println("Error:", err.Error())
	}

	if reply != "" {
		println("Reply:", reply)
	}
	s.Runlevel(0)
}
func TestRunlevel3(t *testing.T) {

	s.Config.Name = "TestRunlevel3"
	s.ErrorLog.SetPrefix("TestRunlevel3: ")
	s.Config.Level = 3
	s.Config.Addr = testAddr
	s.Config.Socket = testSocket
	err := s.Start()
	if err != nil {
		log.Println(err.Error())
		t.FailNow()
	}
	reply, err := testClient.Send("status")
	if err != nil {
		println("Error:", err.Error())
	}

	if reply != "" {
		println("Reply:", reply)
	}
	if !strings.Contains(reply, "Current level: 3") {
		println("Wanted: Current level: 3, could not find")
		t.FailNow()
	}
	s.Runlevel(0)
	<-time.After(time.Second)
}

type teststruct struct {
	Test, PassPrefix, PassSuffix string
}

func TestClientCommands(t *testing.T) {
	s.Config.Level = 1
	s.Config.Name = "Diamond ⋄ 0.5"
	s.Config.Addr = testAddr
	s.ErrorLog.SetPrefix("TestSocket: ")
	s.Config.Socket = testSocket
	s.Config.Kickable = false
	err := s.Start()
	if err != nil {
		log.Println(err.Error())
		t.FailNow()
	}
	s.Runlevel(1)
	cases := []teststruct{
		{"status", "Server Name: Diamond ⋄ 0.5\nDiamond Version: 0.5\nDefault Runlevel: 1\nCurrent Runlevel: 1\nSocket: ./delete-me\nAddr: 127.0.0.1:30000 (Not Listening)", "Recent Connections: 0\nTotal Connections: 0\n"},
		{"help", "Commands:", ""},
		{"KICK", "NO WAY", ""}, // KICK is disabled in this test, tested in TestKick(t)
		{"CUSTOM help", "not defined:", ""},
		{"rebuild", "rebuilding is not enabled", ""}, // rebuild is not defined yet, tested in TestDefineRebuild
		{"update", "updating is not enabled", ""},    // update is not defined yet, tested in TestDefineUpdate
		//	{"redeploy", "Redeploying ⋄", ""}, // redeploy is tested more thoroughly ahead
	}
	var reply string
	for i, v := range cases {
		log.Printf("Command #%v %v", i, v.Test)
		reply, err = testClient.Send(v.Test)
		if err != nil {
			log.Println("Error:", err)
			t.FailNow()
		}

		if v.PassPrefix != "" && strings.HasPrefix(reply, v.PassPrefix) {
			log.Printf("found prefix: %q", v.PassPrefix)
		} else {
			log.Printf("expected prefix: \n%q, got: \n%q", v.PassPrefix, reply)
			t.FailNow()
		}

		if v.PassSuffix != "" && strings.HasSuffix(reply, v.PassSuffix) {
			log.Printf("found suffix: %q", v.PassSuffix)
		} else {
			log.Printf("expected \n%q, got \n%q", v.PassSuffix, reply)
			t.FailNow()
		}

	}
	log.Println("Completed")
}

var uptime time.Duration

func TestUptime(t *testing.T) {
	uptime = s.Uptime()
	log.Printf("Testing took: %s", uptime.String())
}

func TestRedeploy(t *testing.T) {

	reply, err := testClient.Send("redeploy")
	if err != nil {
		println(err)
		println(reply)
		t.FailNow()
		return
	}

	if reply != "" {
		println(reply)
	}

	if s.Uptime() > uptime {
		println("Expected lower uptime after redeploying, got: %s > %s", s.Uptime(), uptime)
		t.FailNow()
	}
}

func TestTeardown(t *testing.T) {
	s.Runlevel(0)
}

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
