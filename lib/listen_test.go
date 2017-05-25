package diamond

import (
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/tv42/httpunix"
)

var testclient http.Client
var testsocket = "httpsocket.tmp"

func init() {
	// set up test client (http, unix)
	socket := testsocket
	u := &httpunix.Transport{
		DialTimeout:           100 * time.Millisecond,
		RequestTimeout:        1 * time.Second,
		ResponseHeaderTimeout: 1 * time.Second,
	}
	u.RegisterLocation(socket, socket)
	t := &http.Transport{}
	t.RegisterProtocol(httpunix.Scheme, u)
	testclient = http.Client{
		Transport: t,
	}

}

var foohandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("foo!\n"))
})

func TestOpenCloseListeners(t *testing.T) {
	srv, _ := createTestServer(t)
	testlisteners := []listener{
		listener{ltype: "tcp", laddr: "127.0.0.1:30001"},
		listener{ltype: "tcp", laddr: "127.0.0.1:30002"},
		listener{ltype: "unix", laddr: testsocket},
	}
	for _, v := range testlisteners {
		n, err := srv.AddListener(v.ltype, v.laddr)
		if err != nil {
			t.Log(n, "listeners", err)
			t.FailNow()
		}
		t.Log(n, "listeners")
	}
	srv.SetHandler(foohandler)
	srv.Runlevel(1)
	// test that we cant connect
	for _, v := range testlisteners {
		u := "http://" + v.laddr + "/"
		if v.ltype == "unix" {
			u = httpunix.Scheme + "://" + v.laddr + "/"
		}
		resp, err := testclient.Get(u)
		if err == nil {
			t.Fail()
			defer resp.Body.Close()
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Log(err)
			}
			println(string(b))
			t.Log("wanted error, got response")
			t.FailNow()
		}
		//	t.Log("got expected error:", err)
	}
	srv.Runlevel(3)
	// test we can connect
	for _, v := range testlisteners {

		u := "http://" + v.laddr + "/"
		if v.ltype == "unix" {
			u = httpunix.Scheme + "://" + v.laddr + "/"
		}
		println("testing", v.ltype, u)
		resp, err := testclient.Get(u)
		if err != nil {
			log.Println(err)
			t.FailNow()
		} else {

			defer resp.Body.Close()
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
				t.FailNow()
			}
			r := string(b)
			if r != "foo!\n" {
				log.Printf(`expected "foo!\n", got %q`, r)
				t.FailNow()
			}
			println(r)
		}
	}

	srv.Runlevel(1)
	// test we cant connect again after downshift
	for _, v := range testlisteners {
		u := "http://" + v.laddr + "/"
		if v.ltype == "unix" {
			u = httpunix.Scheme + "://" + v.laddr + "/"
		}
		resp, err := testclient.Get(u)

		if err == nil {
			log.Println(err)
			t.Fail()
			defer resp.Body.Close()
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
			}
			t.Log("wanted error, got", string(b))
			t.FailNow()
		}
		t.Log("got expected error:", err)
	}

}
