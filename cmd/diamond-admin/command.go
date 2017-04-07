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
	clientname  = "ADMIN" // use linker flag to change at compilation time
	socketpath  string    // use linker flag or CLI flag
)

const (
	cmdStatus = "status"
	stderr    = "stderr"
)

var (
	msg, resp string // to display in the Entry screen
	resperr   error
)

func init() {
	log.SetFlags(log.Lshortfile)
}

// what a great main function
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
			println(rerr)
			os.Exit(222)
		}
		if rp == "" {
			println("empty reply from server")
			os.Exit(222)
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
				msg = "Reconnected to: " + client.ServerName
			} else {
				msg = "Connected to: " + client.ServerName
			}
		}

		if msg != "" {
			b := scrol.Buffer.Bytes()
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

		mm.NewItem("Check Server Status") // status
		mm.NewItem("Halt Server")         // tells user to use telinit 0
		mm.NewItem("Single User Mode")    // telinit 1
		mm.NewItem("Multi User Mode")     // telinit 3
		mm.NewItem("Redeploy Server")     // redeploy
		mm.NewItem("Update Server")       // update (must be defined)
		mm.NewItem("Rebuild Server")      // redeploy (must be defined)
		mm.NewItem("Clear Buffer")        // save entire session to /tmp file
		mm.NewItem("Save Buffer")         // save entire session to /tmp file

		entry := clix.NewEntry(mm.GetScreen())
		mm.AddEntry("Other", entry) // manual command

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
			case "Update Server":
				cmd = "update"
			case "Rebuild Server":
				cmd = "rebuild"
			case "Redeploy Server":
				cmd = "redeploy"
			case "Clear Buffer":
				scrol.Buffer.Truncate(0)
				continue
			case "Save Buffer":
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
		/*

			Commands:

				Enter Runlevel: telinit [#]
				Restart Executable: redeploy



		*/
		switch {
		case cmd == "", cmd == "quit":
			mm.GetScreen().Fini()
			os.Exit(0)
		case cmd == "Halt Server":
			msg = "Are you sure? Please select OTHER and type: telinit 0" // to prevent accidental halt
			continue
		case cmd == "stop":
			cmd = "telinit 0"
			fallthrough
		case strings.HasPrefix(cmd, "telinit"):
			msg = "Runlevel: " + cmd
			resp, resperr = client.Send(cmd)
			if resp == "DONE" {
				resp, resperr = client.Send(cmdStatus)
			}
		case cmd == "restart":
			cmd = "redeploy"
			fallthrough
		case cmd == "redeploy":
			resp, resperr = client.Send(cmd)
			if strings.Contains(resp, "Redeploying") {
				<-time.After(200 * time.Millisecond)
				resp, resperr = client.Send(cmdStatus)
			}

		// not a known command, lets try RPC anyways
		default:
			msg = "Alien: " + cmd
			resp, resperr = client.Send(cmd)
		}
		tmp <- 1

	}
}

func notrunning() {
	println("Server might not be running. Fix that first.")
	os.Exit(2)
}
