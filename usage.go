// usage of diamond library
package main

import "time"
import "net/http"
import "log"
import "os"

import diamond "github.com/aerth/diamond/lib" // ⋄

var logfile = os.Stderr

func init() {
	// add default update, upgrade, redeploy
	// without these reassignments, the functions are nil and
	// trying them (via unix socket) will be denied
	diamond.ToolUpdate = diamond.DefaultToolUpdate
	diamond.ToolRebuild = diamond.DefaultToolRebuild
	diamond.ToolUpgrade = diamond.DefaultToolUpgrade

	// demo using logfile as arg[1] to experiment with stderr vs logfile
	if len(os.Args) > 1 {
		switch os.Args[1] {
		default: //
		case "-h", "--help", "help":
			println(`` +
				`Diamond ⋄ Demo

USAGE
` + os.Args[0] + ` [<logfile>]


EXAMPLES
Output to d.log:
` + os.Args[0] + ` d.log

Output to /dev/stderr:
` + os.Args[0] + `
`)
			os.Exit(2)
		}
		file, err := os.OpenFile(os.Args[1], os.O_CREATE|os.O_APPEND|os.O_WRONLY, diamond.CHMODFILE)
		if err != nil {
			// print error, but ignore chosen filename, use stderr
			println(err.Error())
		} else {
			logfile = file
		}
	}
}

func main() {
	// Create new diamond.Server
	d := diamond.NewServer()

	// Deferred functions get called during runlevel 0 (before shutdown)
	fn := func() {
		d.ErrorLog.Println("hello defer in main")
	}

	// Defer accepts a pointer to a function, not a function
	d.Defer(&fn)

	// Set http.Handler before Start()
	d.SetMux(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(d.Status()))
	}))

	// Name for diamond-admin replies
	d.Config.Name = "Diamond Demo ⋄"

	// ErrorLog can be redirected
	d.ErrorLog = log.New(logfile, "⋄ ", log.Ltime)
	println("[demo] logging to", logfile.Name())
	println("[demo]", d.Config.Name)

	// ConfigFields can be changed before Start()
	d.Config.Addr = ":8777"             // can be empty if SocketHTTP is non-empty
	d.Config.Socket = "./diamond.sock"  // must be non-empty
	d.Config.SocketHTTP = "./http.sock" // can be empty
	d.Config.Level = 1                  // in three seconds we will switch gears (demo)
	d.Config.Debug = true

	// Start() returns fatal error or nil
	err := d.Start()
	if err != nil {
		println("[demo]", err.Error())
		os.Exit(111)
	}

	// Server is now in configured initial runlevel ( Above, we set d.Config.Level = 1 )
	// Admin socket is created, and ready for commands.

	// HookLevel0 gets called LAST after deferred funcs.
	println("[demo]", "adding hook for runlevel 0")

	quitchan := make(chan string, 1)

	diamond.HookLevel0 = func() {
		quitchan <- "HookLevel0 runs after all deferred functions, before final os.Exit(0)"
	}

	// For demo, wait three seconds, switch gears.
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

	// Now is a great time to try out diamond-admin command!
	println("\n\n[demo]", "Now open 'diamond-admin -s "+d.Config.Socket+"\n\n")

	// Lets add a deferred function! Last in, first out!
	fn2 := func() {
		d.ErrorLog.Println("Deferred functions are executed in 'last in, first out' order!")
	}

	// Functions can change
	fn = func() {
		d.ErrorLog.Println("Deferred functions can be modified after the fact, because they are pointers")
	}
	d.Defer(&fn2)

	// Could use d.Done chan, but we have customized quitchan for this demo
	for {
		select {
		case <-time.After(100 * time.Second):
			println("[demo]", "Status:\n", d.Status())
		case cya := <-quitchan:
			println("[demo]", cya)
			return
		}
	}

	// Recommended way of using d.Done chan
	println(<-d.Done)
	println("THANKS FOR TRYING DIAMOND DEMO")
}
