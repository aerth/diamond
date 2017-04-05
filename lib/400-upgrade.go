package diamond

import (
	"context"
	"os/exec"
	"os"
	"time"
	"github.com/aerth/spawn"
)

// ToolGitPull ...
var ToolGitPull = func()( output string, err error ){ return }

// ToolRebuild ...
var ToolRebuild = func()( output string, err error ){ return }

// ToolUpgrade ...
var ToolUpgrade = func()( output string, err error ){ return }

func DefaultToolGitpull() (string, error) { // RPC: update
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "pull", "origin", "master")
	b, er := cmd.CombinedOutput()
	if er != nil {
		return string(b), er
	}
	return string(b), nil
}
func DefaultToolRebuild() (string, error) { // RPC: rebuild
	buildfile := "bin/build.sh"
	buildargs := ""
	ctx := context.Background()
	if _, err := os.Open(buildfile); err != nil {
		buildfile = "build.sh"
	}
	cmd := exec.CommandContext(ctx, buildfile, buildargs)
	b, er := cmd.CombinedOutput()
	if er != nil {
		return string(b), er
	}
	return string(b), nil
}

func DefaultToolUpgrade() (string, error) { // RPC: upgrade
	s, e := ToolGitPull()
	if e != nil {
		return s, e
	}
	s2, e := ToolRebuild()
	if e != nil {
		return s + s2, e
	}
	return s + s2, e
}

func (s *Server) respawn() {
	s.ErrorLog.Printf("Respawning %s", time.Now())
	spawn.Spawn()
}

// Kick ! Another Diamond is occupying our socket. Let's kick it!
func (s *Server) Kick() string {
	client, e := NewClient(s.Config.Socket)
	if e != nil {
		return e.Error()
	}
	reply, e := client.Send("KICK")
	if e != nil {
		return reply+e.Error()
	}
	return reply

}
