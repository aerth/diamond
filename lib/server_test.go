package diamond

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	socket, err := ioutil.TempFile("", "testsocket") // unique filename
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	os.Remove(socket.Name())
	defer os.Remove(socket.Name()) // isnt created yet
	_, err = NewServer(socket.Name())
	if err != nil {
		t.Logf("tried to create socket %q, got error: %v", socket.Name(), err)
		t.FailNow()
	}
}

func createTestServer(t *testing.T) (*Server, string) {
	socket, err := ioutil.TempFile("", "testsocket") // unique filename
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	os.Remove(socket.Name())
	srv, err := NewServer(socket.Name())
	if err != nil {
		t.Logf("tried to create socket %q, got error: %v", socket.Name(), err)
		t.FailNow()
		return nil, socket.Name()
	}
	return srv, socket.Name()
}

func TestRunlevelBasic(t *testing.T) {
	srv, socket := createTestServer(t)
	defer os.Remove(socket)
	// set runlevel
	srv.SetRunlevel(1, func() error { println("runlevel works"); return nil })

	// enter runlevel
	if err := srv.Runlevel(1); err != nil {
		t.Logf("error while trying to enter runlevel 1: %v", err)
		t.FailNow()
	}

}

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
	srv, err := NewServer(socket.Name())
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

func TestChangeRunlevel(t *testing.T) {
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
