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

// Command a diamond server
package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aerth/clix"
	diamond "github.com/aerth/diamond/lib"
)

var (
	sock        = flag.String("s", "", "path to socket")
	refreshtime = flag.Duration("r", time.Minute*30, "refresh status duration")
	clientname  = "ADMIN" // use linker flag to change at compilation time
	socketpath  string    // use linker flag or CLI flag
)

const (
	cmdStatus   = "status"
	cmdRedeploy = "redeploy"
	stderr      = "stderr"
)

var (
	//msg, resp string // to display in the Entry screen
	//resperr   error
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	flag.Parse()
	if *sock == "" && socketpath == "" {
		flag.Usage()
		println("Need socket flag (diamond-admin -s /path/to/socket)")
		os.Exit(2)
	}
	if *sock != "" {
		socketpath = *sock
	}
	if len(flag.Args()) > 1 {
		client := buildClient()
		reply, err := client.Send(strings.Join(flag.Args()[1:], " "))
		if err != nil {
			println(err.Error())
		}
		if reply != "" {
			println(reply)
		}

		return
	}

	doCUI(socketpath)
}

func quit(mm *clix.MenuBar) {
	mm.GetScreen().Fini()
	println("Check for updates often! https://github.com/aerth/diamond")
}

func buildWindow() *clix.MenuBar {
	mm := clix.NewMenuBar(nil)
	mm.SetMessage("⋄ DIAMOND CMD v0.2")
	scrol := clix.NewScrollFrame("⋄")
	mm.AttachScroller(scrol)
	return mm
}

func buildMenu(mm *clix.MenuBar) {
	mm.NewItem("Check Server Status") // status
	mm.NewItem("Clear Buffer")        // save entire session to /tmp file
	mm.NewItem("Toggle` Buffer")        // save entire session to /tmp file
	mm.NewItem("Save Buffer")         // save entire session to /tmp file
	entry := clix.NewEntry(mm.GetScreen())
	mm.AddEntry("Other", entry) // manual command
	mm.NewItem("Quit Admin")
	mm2 := clix.NewMenuBar(mm.GetScreen())
	mm2.NewItem("help")
	mm2.NewItem("Single User Mode")    // telinit 1
	mm2.NewItem("Multi User Mode")     // telinit 3
	mm2.NewItem("Redeploy Server")     // redeploy
	mm2.NewItem("Clear Buffer")        // save entire session to /tmp file
	mm2.NewItem("Save Buffer")         // save entire session to /tmp file
	mm.AddSibling(mm2)

}

func handleKeyMouse(mm *clix.MenuBar) *clix.EventHandler {
	ev := clix.NewEventHandler()
	ev.AddMenuBar(mm)
	ev.Launch()
	mm.GetScreen().Show()
	return ev
}
func handleMenuInput(mm *clix.MenuBar, ev *clix.EventHandler) (cmd string) {
	select {
	case <-time.After(*refreshtime):
		cmd = cmdStatus
	case c := <-ev.Output:
		mm.GetScreen().Show()
		switch c.(string) {
		case "Quit Admin":
			cmd = "quit"
		case "Check Server Status":
			cmd = cmdStatus
		case "Single User Mode":
			cmd = "telinit 1"
		case "Multi User Mode":
			cmd = "telinit 3"
		case "Update Server":
			cmd = "update"
		case "Rebuild Server":
			cmd = "rebuild"
		case "Redeploy Server":
			cmd = cmdRedeploy
		case "Toggle Buffer":
			bbuf = mm.GetScroller().Buffer
			mm.GetScroller().Buffer = buf
		case "Clear Buffer":
			mm.GetScroller().Buffer.Truncate(0)
			mm.GetScreen().Clear()
			mm.GetScreen().Show()
			cmd = ""
		case "Save Buffer":
			tmpfile, e := ioutil.TempFile(os.TempDir(), "/diamond.log.")
			if e == nil {
				mm.GetScroller().Buffer.WriteTo(tmpfile)
				mm.GetScroller().Buffer.WriteString("Saved to:" + tmpfile.Name())
			}
			if e != nil {
				mm.GetScroller().Buffer.WriteString(e.Error())
			}
		default:
			cmd = c.(string)
		}
	}

	return cmd
}

func notrunning() {
	println("Server might not be running. Fix that first.")
	os.Exit(2)
}

func buildClient() *diamond.Client {
	client, e := diamond.NewClient(socketpath)
	if e != nil {
		println(e.Error())
		os.Exit(2)
	}
	client.Name = clientname
	r, e := client.Send("HELLO from " + client.Name)
	if e != nil {
		if strings.Contains(e.Error(), "no such file or directory") {
			notrunning()
			os.Exit(2)
		} else {
			println(e.Error())
			notrunning()
			os.Exit(2)
		}

	}

	if !strings.HasPrefix(r, "HELLO from ") {
		println("Can't connect to socket")
		os.Exit(2)
	}

	client.ServerName = strings.TrimPrefix(r, "HELLO from ")
	args := strings.Join(flag.Args(), " ")
	if args != "" && !strings.HasPrefix(args, "-") {
		println("Command:", args)
		rp, rerr := client.Send(args)
		if rp != "" {
			println(rp)
		}
		if rerr != nil {
			println(rerr.Error())
			os.Exit(222)
		}
		if rp == "" {
			println("empty reply from server")
			os.Exit(222)
		}
		os.Exit(0)
	}

	return client
}
var buf = new(bytes.Buffer)
var bbuf = new(bytes.Buffer)

func doCUI(socketpath string) {
	client := buildClient()
	mm := buildWindow()
	buildMenu(mm)
	var msg string
	var resperr error
	for {
		if msg == "" {
			msg = "Connected to: " + client.ServerName
		}
		if msg != "" {
			mm.GetScroller().Buffer.Truncate(0)
			mm.GetScroller().Buffer.WriteString(msg)
			mm.GetScroller().Buffer.WriteString("\n")
		}
		// Reset messages each loop after setting message
		mm.GetScroller().ScrollToEnd()
		ev := handleKeyMouse(mm)
		cmd := handleMenuInput(mm, ev)

		if cmd == "quit" {
			quit(mm)
			return
		}
		if cmd != "" {
		buf.WriteString("SENT: "+cmd)
		msg, resperr = client.Send(cmd)
		buf.WriteString("REPLY: "+msg+"\n")
			if resperr != nil {
				mm.GetScreen().Fini()
				println(resperr.Error())
				os.Exit(111)
			}
		}
		mm.GetScreen().Show()
	}

}
