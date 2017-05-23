package main

import (
	"os"

	diamond "github.com/aerth/diamond/lib"
)

func runlevel0() error {
	println("Goodbye!")
	return nil
}

func runlevel1() error {
	println("demo runlevel 1")
	return nil
}

func runlevel3() error {
	println("demo runlevel 3")
	return nil
}

func main() {
	srv, err := diamond.NewServer("diamond.socket")
	if err != nil {
		println(err.Error())
		os.Exit(111)
	}
	srv.Config.Verbose = true
	srv.SetRunlevel(0, runlevel0)
	srv.SetRunlevel(1, runlevel1)
	srv.SetRunlevel(3, runlevel3)

	err = srv.Runlevel(1)
	if err != nil {
		println(err.Error())
		os.Exit(111)
	}

	srv.Wait() // wait until runlevel 0

}
