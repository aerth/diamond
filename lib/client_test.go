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
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestClient(t *testing.T) {

	println("Creating Socket")
	socket, err := ioutil.TempFile("", "testsocket") // unique filename
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	os.Remove(socket.Name())
	defer os.Remove(socket.Name()) // isnt created yet
	println("Creating Server")
	srv, err := New(socket.Name())
	if err != nil {
		t.Logf("tried to create socket %q, got error: %v", socket.Name(), err)
		t.FailNow()
	}
	_ = srv // i know

	println("Creating Client")
	client, err := NewClient(socket.Name())
	if err != nil {
		t.Logf("tried to create client, got error: %v", err)
		t.FailNow()
	}

	var cmdreply string
	c := make(chan string, 1)
	go func() {
		println("Sending ECHO")
		reply, err := client.Send("echo", "hello world")
		if err != nil {
			t.Logf("tried to send command, got error: %v", err)
			t.FailNow()
		}
		c <- reply
	}()

	select {
	case <-time.After(3 * time.Second):
		t.Log("Timeout waiting for reply")
		t.FailNow()
	case reply := <-c:
		cmdreply = reply
	}

	println("Got reply:", cmdreply)

	if cmdreply != "hello world" {
		t.Logf(`expected "hello world", got: %q`, cmdreply)
		t.FailNow()
	}
}

func TestClientRunlevel(t *testing.T) {
	srv, socket := createTestServer(t)
	defer os.Remove(socket)

	srv.SetRunlevel(1, func() error { fmt.Println("hi"); return nil })

	println("Creating Client")
	client, err := NewClient(socket)
	if err != nil {
		t.Logf("tried to create client, got error: %v", err)
		t.FailNow()
	}

	var cmdreply string
	c := make(chan string, 1)
	go func() {
		println("Sending RUNLEVEL 1 request")
		reply, err := client.Send("runlevel", "1")
		if err != nil {
			t.Logf("tried to send command, got error: %v", err)
			t.FailNow()
		}
		c <- reply
	}()

	select {
	case <-time.After(3 * time.Second):
		t.Log("Timeout waiting for reply")
		t.FailNow()
	case reply := <-c:
		cmdreply = reply
	}

	println("Got reply:", cmdreply)

	if cmdreply != "1" {
		t.Logf(`expected "1", got: %q`, cmdreply)
		t.FailNow()
	}
}

func TestClientKick(t *testing.T) {
	srv, socket := createTestServer(t)
	srv.Config.Kickable = true
	defer os.Remove(socket)

	srv.SetRunlevel(1, func() error { fmt.Println("hi"); return nil })

	println("Creating Client")
	client, err := NewClient(socket)
	if err != nil {
		t.Logf("tried to create client, got error: %v", err)
		t.FailNow()
	}

	var cmdreply string
	c := make(chan string, 1)
	go func() {
		println("Sending KICK")
		reply, err := client.Send("KICK")
		if err != nil {
			t.Logf("tried to send command, got error: %v", err)
			t.FailNow()
		}
		c <- reply
	}()

	select {
	case <-time.After(3 * time.Second):
		t.Log("Timeout waiting for reply")
		t.FailNow()
	case reply := <-c:
		cmdreply = reply
	}

	if cmdreply != "OKAY" {
		t.Log("Wanted OKAY, got:", cmdreply)
		t.FailNow()
	}
	println("kicked!")

}
