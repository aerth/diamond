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
	"strings"
	"testing"
	"time"
)

import "log"

var s = NewServer()
var testAddr = "127.0.0.1:30000"
var testClient *Client
var testSocket = "./delete-me"

func init() {
	println("initializing test client")
	s.Signals(false)
	log.SetFlags(log.Lshortfile)
	var err error
	testClient, err = NewClient(testSocket)
	if err != nil {
		panic(err)
	}
	s.ErrorLog.SetFlags(log.Lshortfile)
	s.ErrorLog.SetPrefix("Diamond [test]: ")
}

func up() error {
	reply, err := testClient.Send("status")
	if reply != "" {
		s.ErrorLog.Printf("Status Reply:\n" + reply)
	}
	return err
}

func stat() (string, error) {
	return testClient.Send("status")
}

func TestNewServer(t *testing.T) {
	s.Config.Socket = testSocket
	s.Config.Addr = testAddr
	println("starting first server")
	s.Config.Name = "NewServer"
	s.ErrorLog.SetPrefix("NewServer" + ": ")
	err := s.Start()
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
		return
	}

	// check if admin command 'status' works
	if err := up(); err != nil {
		t.Log("Wanted server to be up, got down", err)
		t.FailNow()
		return
	}

	s.Runlevel(1)
	s.Runlevel(0)

	//<- time.After(0*time.Millisecond)
	// check if quit
	if nonzero := s.Level(); nonzero != 0 {
		t.Log("Wanted 0, got:", nonzero)
		t.FailNow()
		return
	}

	if err := up(); err == nil {
		t.Log("Expected not to be up, got up")
		t.FailNow()
		return
	}

	t.Log("NewServer", "Passed")

}

func TestRunlevel1(t *testing.T) {
	s.Config.Level = 1
	s.Config.Name = "Runlevel 1"
	s.ErrorLog.SetPrefix("Runlevel 1" + ": ")
	s.Config.Addr = testAddr
	s.Config.Socket = testSocket
	err := s.Start()
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
	// quit properly
	defer func() {
		s.Runlevel(0)
		<-time.After(time.Second)
	}()

	reply, err := testClient.Send("status")
	if err != nil {
		println("Error:", err.Error())
		t.FailNow()
		return
	}

	if reply != "" {
		println("Reply:", reply)
	}

}
func TestRunlevel3(t *testing.T) {
	s.Config.Name = "Runlevel 1"
	s.Config.Level = 3
	t.Log(s.Config.Level)
	s.ErrorLog.SetPrefix("Runlevel 1" + ": ")
	s.Config.Addr = testAddr
	s.Config.Socket = testSocket
	err := s.Start()
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
		return
	}
	// quit properly
	defer func() {
		s.Runlevel(0)
		<-time.After(time.Second)
	}()

	<-time.After(time.Second)
	reply, err := testClient.Send("status")
	if err != nil {
		println("Error:", err.Error())
		t.Fail()
	}

	if reply != "" {
		println("Reply:", reply)
	}
	if !strings.Contains(reply, "Current Runlevel: 3") {
		println("Wanted: Current Runlevel: 3, could not find")
		t.FailNow()
	}
}

type teststruct struct {
	Test, PassPrefix, PassSuffix string
}

func TestClientCommands(t *testing.T) {
	s.Config.Level = 1
	s.Config.Name = "Diamond ⋄ "+version
	s.Config.Addr = testAddr
	s.ErrorLog.SetPrefix("TestSocket: ")
	s.Config.Socket = testSocket
	s.Config.Kickable = false
	err := s.Start()
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
	// quit properly
	defer func() {
		s.Runlevel(0)
		<-time.After(time.Second)
	}()

	s.Runlevel(1)
	<-time.After(time.Second)
	cases := []teststruct{
		{"status", "Server Name: Diamond ⋄ " + version + "\n", ""},
		//{"status", "Server Name: Diamond ⋄ "+version+"\nDiamond Version: "+version+"\nDefault Runlevel: 1\nCurrent Runlevel: 1\nSocket: ./delete-me\nAddr: 127.0.0.1:30000 (Not Listening)", "Recent Connections: 0\nTotal Connections: 0\n"},
		{"help", "Commands:", ""},
		{"KICK", "NO WAY", ""}, // KICK is disabled in this test, tested in TestKick(t)
		{"CUSTOM help", "not defined:", ""},
		{"rebuild", "rebuilding is not enabled", ""}, // rebuild is not defined yet, tested in TestDefineRebuild
		{"update", "updating is not enabled", ""},    // update is not defined yet, tested in TestDefineUpdate
		//	{"redeploy", "Redeploying ⋄", ""}, // redeploy is tested more thoroughly ahead
	}
	var reply string
	for i, v := range cases {
		t.Logf("Command #%v %v", i, v.Test)
		reply, err = testClient.Send(v.Test)
		if err != nil {
			t.Log("Error:", err)
			t.FailNow()
		}

		if strings.HasPrefix(reply, v.PassPrefix) {
			if v.PassSuffix != "" {
				t.Logf("found prefix: %q", v.PassPrefix)
			}
		} else {
			t.Logf("expected prefix: \n%q, got: \n%q", v.PassPrefix, reply[:len(v.PassPrefix)])
			t.FailNow()
		}

		if strings.HasSuffix(reply, v.PassSuffix) {
			if v.PassSuffix != "" {
				t.Logf("found suffix: %q", v.PassSuffix)
			}
		} else {
			t.Logf("expected \n%q, got \n%q", v.PassSuffix, reply[:len(v.PassSuffix)])
			t.FailNow()
		}
	}

}

var uptime time.Duration

func TestKick(t *testing.T) {
	println("starting test kick")
	s.Config.Socket = testSocket
	s.Config.Name = "Runlevel 1"
	s.Config.Kickable = true
	s.ErrorLog.SetPrefix("Runlevel 1" + ": ")
	err := s.Start()
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
	// quit properly
	defer func() {
		s.Runlevel(0)
		<-time.After(time.Second)
	}()

	<-time.After(time.Second)
	if err = up(); err != nil {
		t.Log("Wanted server to be up, got down", err)
		t.FailNow()
		return
	}

	println("Testing KICK!")
	reply, err := testClient.Send("KICK")
	if reply != "OKAY" {
		t.Logf("Wanted \"OKAY\", got %q\n", reply)
		t.FailNow()
		return
	}
	if err != nil {
		t.Log("error:", err.Error())
		t.FailNow()
		return
	}

	t1 := time.Now()
	for i := 0; i < 10; i++ {

		<-time.After(300 * time.Millisecond)
		reply, err := stat()
		if err != nil {
			if err.Error() != "dial unix ./delete-me: connect: no such file or directory" {
				t.Log("Test Kick", "failed:", err)
				t.Fail()
			}
			break
		}
		if reply != "" {
			t.Logf("Attempt %v, wanted server to be down, got: \n%s", i, reply)
			t.Fail()
		}
		t.Log(i)
	}
	t.Logf("Took %s to go down", time.Now().Sub(t1))
	<-time.After(time.Second)
	return

}
func TestUptime(t *testing.T) {
	uptime = s.Uptime()
	t.Logf("Testing took: %s", uptime.String())
}

func NoTestRedeploy(t *testing.T) {

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
	<-time.After(time.Second)
}
