package diamond

import (
	"context"
	"os/exec"
	"os"
)

// ToolGitPull ...
var ToolGitPull = func()( output string, err error ){ return }

// ToolRebuild ...
var ToolRebuild = func()( output string, err error ){ return }

// ToolRedeploy ...
var ToolRedeploy = func()( output string, err error ){ return }

func defaultgitpull() (string, error) { // RPC: update
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "pull", "origin", "master")
	b, er := cmd.CombinedOutput()
	if er != nil {
		return string(b), er
	}
	return string(b), nil
}
func defaultrebuild() (string, error) { // RPC: rebuild
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

func defaultupgrade() (string, error) { // RPC: upgrade
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
