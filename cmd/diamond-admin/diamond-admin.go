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

// diamond-admin command for controlling a diamond daemon
package main

import (
	"bytes"
	"flag"
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
	if len(flag.Args()) > 0 { // custom CLI command, no menu
		client := buildClient()
		reply, err := client.Send(flag.Args()[0], strings.Join(flag.Args()[1:], " "))
		if err != nil {
			println(err.Error())
		}
		if reply != "" {
			println(reply)
		}

		return
	}

	// CLI menu
	doCUI(socketpath)
}

func quit(mm *clix.MenuBar) {
	mm.GetScreen().Fini()
	println("Check for updates often! https://github.com/aerth/diamond")
}

func buildWindow() *clix.MenuBar {
	mm := clix.NewMenuBar(nil)
	mm.SetMessage("⋄ DIAMOND CMD v0.3")
	scrol := clix.NewScrollFrame("⋄")
	mm.AttachScroller(scrol)
	return mm
}

func buildMenu(mm *clix.MenuBar) {
	entry := clix.NewEntry(mm.GetScreen())
	mm.AddEntry("Command", entry) // manual command
	mm.NewItem("Quit Admin")

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
			cmd = "runlevel 1"
		case "Multi User Mode":
			cmd = "runlevel 3"
		case "Redeploy Server":
			cmd = cmdRedeploy
		default: // custom command
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
	r, e := client.Send("HELLO", "from "+client.Name)
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
	return client
}

func doCUI(socketpath string) {
	var buf = new(bytes.Buffer)
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
			buf.WriteString("SENT: " + cmd)

			argv := strings.Split(cmd, " ")
			if len(argv) < 2 {
				msg, resperr = client.Send(argv[0])
			} else {
				msg, resperr = client.Send(argv[0], strings.Join(argv[1:], " "))
			}

			buf.WriteString("REPLY: " + msg + "\n")
			if resperr != nil {
				mm.GetScreen().Fini()
				println(resperr.Error())
				os.Exit(111)
			}
		}
		mm.GetScreen().Show()
	}

}
