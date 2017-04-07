// The most simple Diamond ⋄ server
package main

import "time"
import "net/http"
import "log"
import diamond "github.com/aerth/diamond/lib" // ⋄

// add default update, upgrade, redeploy
func init() {
	diamond.ToolUpdate = diamond.DefaultToolUpdate
	diamond.ToolRebuild = diamond.DefaultToolRebuild
	diamond.ToolUpgrade = diamond.DefaultToolUpgrade
}
func main() {
	// Create new diamond.Server
	d := diamond.NewServer(nil)
	d.SetMux(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(d.Status()))
	}))
	d.Config.Name = "Diamond Demo ⋄"
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
		case <-time.After(10 * time.Second):
			println("[demo]", "Status:\n", d.Status())
		case cya := <-quitchan:
			println("[demo]", cya)
			return
		}
	}

}
