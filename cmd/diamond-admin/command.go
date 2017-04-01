// Command a diamond server
package main

import (
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
)

const (
	cmdStatus = "status"
)

var (
	msg, resp  string // to display in the Entry screen
	resperr    error
	socketpath string //
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
	try := 0
Start:
	try++
	client := diamond.NewClient(socketpath)
	client.Name = "ADMIN"
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

	client.Server = strings.TrimPrefix(r, "HELLO from ")
	args := strings.Join(flag.Args(), " ")
	if args != "" {

		if strings.Contains(args, "help") || strings.HasPrefix(args, "-") {
			flag.Usage()
			os.Exit(2)
		}
		println("Command:", args)
		rp, rerr := client.Send(args)
		if rp != "" {
			println(rp)
		}
		if rerr != nil {
			println(rerr)
		}

		os.Exit(0)
	}

	// clix library: create the menubar once,
	// using a single window for the entire for loop
	mm := clix.NewMenuBar(nil)
	mm.SetMessage("⋄ DIAMOND")

	scrol := clix.NewScrollFrame("⋄")
	for {
		if resp != "" {
			msg = resp
		}
		if msg == "" {
			if try != 1 {
				msg = "Reconnected to: " + client.Server
			} else {
				msg = "Connected to: " + client.Server
			}
		}

		b := scrol.Buffer.Bytes()
		scrol.Buffer.Reset()
		if msg != "" {
			scrol.Buffer.WriteString(msg)
			scrol.Buffer.WriteString("\n\n\n\n")
			scrol.Buffer.Write(b)
		}
		mm.AttachScroller(scrol)
		if resperr != nil {
			if strings.Contains(resperr.Error(), "no such file or directory") {
				mm.GetScreen().Fini()
				println("Server is down.")
				os.Exit(0)
			}
			if strings.Contains(resperr.Error(), "unexpected EOF") {
				<-time.After(200 * time.Millisecond)
				mm.GetScreen().Fini()

				goto Start
			}
		}
		// Reset messages each loop after setting message
		msg = ""
		resperr = nil
		resp = ""

		mm.NewItem("Check Server Status")
		mm.NewItem("Halt Server")
		mm.NewItem("Single User Mode")
		mm.NewItem("Multi User Mode")
		mm.NewItem("Upgrade Server")
		mm.NewItem("Rebuild Server")
		mm.NewItem("Redeploy Server")
		mm.NewItem("Save Log Buffer")

		entry := clix.NewEntry(mm.GetScreen())
		mm.AddEntry("Other", entry)

		mm.NewItem("Quit Admin")

		ev := clix.NewEventHandler()
		ev.AddMenuBar(mm)
		ev.Launch()
		mm.GetScreen().Show()
		var cmd string

		select {
		case <-time.After(*refreshtime):

			cmd = cmdStatus

		case c := <-ev.Output:
			mm.GetScreen().Show()
			switch c.(string) {
			case "Quit Admin":
				cmd = "quit"
			case "restart":
				cmd = "telinit 1"
			case "Check Server Status":
				cmd = cmdStatus

			case "Single User Mode":
				cmd = "telinit 1"
			case "Multi User Mode":
				cmd = "telinit 3"
			case "Custom Multi User Mode":
				cmd = "telinit 4"
			case "Upgrade Server":
				cmd = "upgrade"
			case "Rebuild Server":
				cmd = "rebuild"
			case "Redeploy Server":
				cmd = "redeploy"
			case "Save Log Buffer":
				tmpfile, e := ioutil.TempFile(os.TempDir(), "/diamond.log.")
				if e == nil {
					scrol.Buffer.WriteTo(tmpfile)
					resp = "Saved to:" + tmpfile.Name()
				}
				if e != nil {
					resp = e.Error()
				}
				continue
			default:
				cmd = c.(string)
			}
		}
		tmp := make(chan int)
		mm.GetScreen().Clear()
		clix.Type(mm.GetScreen(), 1, 1, 1, "This may take a while...")
		mm.GetScreen().Show()
		var x = 1
		t1 := time.Now()

		go func() {
		Waiting:
			for {
				select {
				case <-time.After(100 * time.Millisecond):
					if time.Now().Sub(t1) > time.Second*4 {
						msg = "Timeout occured."
						continue
					}
					x++
					clix.Type(mm.GetScreen(), len("This may take a while..."), 1, 1, strings.Repeat(".", x))
					mm.GetScreen().Show()
					<-time.After(100 * time.Millisecond)
					mm.GetScreen().Show()
				case <-tmp:
					mm.GetScreen().Clear()
					mm.Present(true)
					break Waiting
				}
			}
			return
		}()

		// do it
		switch {
		case cmd == "", cmd == "quit":
			mm.GetScreen().Fini()
			os.Exit(0)
		case cmd == "Halt Server":
			msg = "Are you sure? Please select OTHER and type: telinit 0" // to prevent accidental halt
			continue
		case cmd == "restart":
			cmd = "redeploy"
			fallthrough
		case cmd == "stop":
			cmd = "telinit 0"
			fallthrough
		case strings.HasPrefix(cmd, "telinit"):
			msg = "Runlevel: " + cmd
			resp, resperr = client.Send(cmd)
			if resp == "DONE" {
				resp, resperr = client.Send(cmdStatus)
			}
		case strings.HasPrefix(cmd, "load"):
			msg = "Load: " + cmd
			resp, resperr = client.Send(cmd)
		case strings.HasPrefix(cmd, "import"):
			msg = "Import: " + cmd
			resp, resperr = client.Send(cmd)

		case cmd == "backup", cmd == "upgrade", cmd == "rebuild", cmd == cmdStatus:
			resp, resperr = client.Send(cmd)

		case cmd == "redeploy":
			resp, resperr = client.Send(cmd)
			if strings.Contains(resp, "Redeploying") {
				<-time.After(200 * time.Millisecond)
				resp, resperr = client.Send(cmdStatus)
			}
		// not a known command, lets try RPC anyways
		default:
			msg = "Trying new command: " + cmd
			resp, resperr = client.Send(cmd)
		}
		tmp <- 1

	}
}

func notrunning() {
	println("Server might not be running. Fix that first.")
	os.Exit(2)
}
