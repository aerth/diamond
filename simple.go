// The most simple Diamond ⋄ server
package main

import "time"

import diamond "github.com/aerth/diamond/lib" // ⋄
/*
This Diamond only serves 404 pages!
*/
func main() {
	// Create new diamond.Server
	d := diamond.NewServer(nil)
	d.Config.Name = "Diamond Demo ⋄"
	d.SetConfigPath("config.json")
	d.Config.Level = 1 // in three seconds we will switch gears
	println(d.Config.Name)
	n, _ := d.SaveConfig()
	println("saved", n, "bytes to config.json")
	d.ConfigPath("config.json")

	// start in d.Config.Level
	err := d.Start()
	if err != nil {
		println(err.Error())
	}

	// redefine HookLevel0
	quitchan := make(chan string, 1)
	diamond.HookLevel0 = func() {
		quitchan <- "goodbye!"
	}

	// wait three seconds, switch gears
	go func() {
		<-time.After(3 * time.Second)
		println("[demo] Switching gears to 3")
		d.Runlevel(3)
		<-time.After(3 * time.Second)
		println("[demo] Switching gears to 1")
		d.Runlevel(1)
		<-time.After(3 * time.Second)
		println("[demo] Switching gears to 3")
		d.Runlevel(3)
	}()

	// wait for quitchan
	select {
	case cya := <-quitchan:
		println(cya)
	}

}

var ccc = []byte(`{
            "Name":"Diamonds! ⋄",
            "Level":3,
            "Addr":":8777",
            "Socket":"/tmp/diamond.socket",
            "Kicks": true,
            "Kickable": true,
    }`)
