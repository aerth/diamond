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
	srv.Log.SetFlags(log.Lshortfile)
	testlisteners := []listener{
		listener{ltype: "tcp", laddr: "127.0.0.1:30001"},
		listener{ltype: "tcp", laddr: "127.0.0.1:30002"},
		listener{ltype: "unix", laddr: testsocket},
	}
	for _, v := range testlisteners {
		srv.Log.Println("AddListener", v.ltype, v.laddr)

		n, err := srv.AddListener(v.ltype, v.laddr)
		if err != nil {
			srv.Log.Println(n, "listeners", err)
			t.FailNow()
		}
		t.Log(n, "listeners")
	}
	srv.SetHandler(foohandler)
	srv.Runlevel(1)
	srv.Log.Println("test that we cant connect")
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
			srv.Log.Println("wanted error, got response")
			t.FailNow()
		}

		srv.Log.Println("got expected error:", err)
	}
	srv.Runlevel(3)
	// test we can connect
	srv.Log.Println("test that we can connect")
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
	<-time.After(time.Second)
	// test we cant tconnect again after downshift
	srv.Log.Println("test that we cant connect again**")
	for _, v := range testlisteners {
		u := "http://" + v.laddr + "/"
		if v.ltype == "unix" {
			u = httpunix.Scheme + "://" + v.laddr + "/"
		}
		srv.Log.Printf("testing: %q %q", v.laddr, v.ltype)
		resp, err := testclient.Get(u)
		if err == nil {
			defer resp.Body.Close()
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				srv.Log.Println(err)
			}
			srv.Log.Println("wanted connection-type error, got http response", string(b))
			srv.Log.Println(v.laddr, v.ltype, u)
			t.FailNow()

		} else {
			srv.Log.Println("got expected error:", err)
		}
	}

}
