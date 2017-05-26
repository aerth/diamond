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
	"os"
	"testing"
)

func TestNewServer(t *testing.T) {
	socket, err := ioutil.TempFile("", "testsocket") // unique filename
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	os.Remove(socket.Name())
	defer os.Remove(socket.Name()) // isnt created yet
	_, err = New(socket.Name())
	if err != nil {
		t.Logf("tried to create socket %q, got error: %v", socket.Name(), err)
		t.FailNow()
	}
}

func createTestServer(t *testing.T) (*System, string) {
	socket, err := ioutil.TempFile("", "testsocket") // unique filename
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	os.Remove(socket.Name())
	srv, err := New(socket.Name())
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
