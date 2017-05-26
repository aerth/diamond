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
	"log"
	"net/http"
	"testing"

	"github.com/tv42/httpunix"
)

func TestRunlevelsHandler(t *testing.T) {

	srv, _ := createTestServer(t)
	srv.Log.SetFlags(log.Lshortfile)
	testlisteners := [100]listener{
		{ltype: "tcp", laddr: "127.0.0.1:30000"},
		{ltype: "tcp", laddr: "127.0.0.1:30001"},
		{ltype: "unix", laddr: testsocket},
	}
	// 30003..30099 now have 99 listeners
	for i := 2; i < 100; i++ {
		testlisteners[i] = listener{
			ltype: "tcp",
			laddr: "127.0.0.1:300" + fmt.Sprintf("%0v", i),
		}
	}
	for _, v := range testlisteners {
		if v.ltype == "" || v.laddr == "" {
			continue
		}
		srv.Log.Println("AddListener", v.ltype, v.laddr)

		n, err := srv.AddListener(v.ltype, v.laddr)
		if err != nil {
			srv.Log.Println(n, "listeners", err)
			t.FailNow()
		}
	}
	srv.SetHandler(foohandler)
	err := srv.Runlevel(1)
	if err != nil {
		srv.Log.Println(err)
		t.FailNow()
	}
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
				srv.Log.Println(err)
				t.FailNow()
			}
			r := string(b)
			if r != "foo!\n" {
				log.Printf(`expected "foo!\n", got %q`, r)
				t.FailNow()
			}
			srv.Log.Println(r)
		}
	}

	srv.Runlevel(1)
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
			t.Log("failing!")
			t.FailNow()

		} else {
			srv.Log.Println("got expected error:", err)
		}
	}

	barhandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("12345"))
	})
	// switch handler what!
	e := srv.SetHandler(barhandler)
	if e != nil {
		srv.Log.Println("failing:", e)

		t.FailNow()
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
				srv.Log.Println(err)
				t.FailNow()
			}
			r := string(b)
			if r != "12345" {
				srv.Log.Printf(`expected "12345", got %q`, r)
				t.FailNow()
			}
			srv.Log.Println(r)
		}

	}
}
