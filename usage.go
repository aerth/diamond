// usage of diamond library
package main

import "time"
import "net/http"
import "log"
import "os"

import diamond "github.com/aerth/diamond/lib" // ⋄

var logfile *os.File

func init() {
	// add default update, upgrade, redeploy
	// without these reassignments, the functions are nil and
	// trying them (via unix socket) will reply such as ""
	diamond.ToolUpdate = diamond.DefaultToolUpdate
	diamond.ToolRebuild = diamond.DefaultToolRebuild
	diamond.ToolUpgrade = diamond.DefaultToolUpgrade
	logfile = os.Stderr

	// logfile as arg[1]
	if len(os.Args) > 1 {
		switch os.Args[1] {
		default://
		case "-h", "--help":
			println(``+
`Diamond ⋄ Demo

USAGE
`+os.Args[0]+` [<logfile>]


EXAMPLES
Output to d.log:
`+os.Args[0]+` d.log

Output to /dev/stderr:
`+os.Args[0]+`
`)
			os.Exit(2)
		}
		file, err := os.OpenFile(os.Args[1], os.O_CREATE|os.O_APPEND|os.O_WRONLY, diamond.CHMODFILE)
		if err != nil {
			println(err.Error())
			} else {
			logfile = file
		}
	}
}
func main() {
	// Create new diamond.Server
	d := diamond.NewServer()
	d.SetMux(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(d.Status()))
	}))
	d.Config.Name = "Diamond Demo ⋄"
	d.ErrorLog.SetOutput(logfile)
	println("[demo] logging to", logfile.Name())
	println("[demo]", d.Config.Name)
	d.Config.Addr = ":8777"
	d.Config.Socket = "./diamond.sock"
	d.Config.Level = 1 // in three seconds we will switch gears
	d.Config.Debug = true
	d.ErrorLog.SetFlags(log.Lshortfile)

	err := d.Start()
	if err != nil {
		println("[demo]", err.Error())
	}

	// redefine HookLevel0
	println("[demo]", "adding hook for runlevel 0")
	quitchan := make(chan string, 1)
	diamond.HookLevel0 = func() {
		quitchan <- "goodbye!"
	}

	// wait three seconds, switch gears
	go func() {
		<-time.After(3 * time.Second)
		println("[demo]", "Switching gears to 3")
		d.Runlevel(3)
		<-time.After(3 * time.Second)
		println("[demo]", "Switching gears to 1")
		d.Runlevel(1)
		<-time.After(3 * time.Second)
		println("[demo]", "Switching gears to 3")
		d.Runlevel(3)
	}()
	println("[demo]", "Now open 'diamond-admin -s ./diamond.sock'")
	// wait for quitchan
	for {
		select {
		case <-time.After(100 * time.Second):
			println("[demo]", "Status:\n", d.Status())
		case cya := <-quitchan:
			println("[demo]", cya)
			return
		}
	}

}
