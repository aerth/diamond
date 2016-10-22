package diamond

import (
	"context"
	"os/exec"
)

func upgGitPull() (string, error) { // RPC: update
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "pull", "origin", "master")
	b, er := cmd.CombinedOutput()
	if er != nil {
		return string(b), er
	}
	return string(b), nil
}
func upgMake() (string, error) { // RPC: rebuild
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "./build.sh", "server")
	b, er := cmd.CombinedOutput()
	if er != nil {
		return string(b), er
	}
	return string(b), nil
}
func upgrade() (string, error) { // RPC: upgrade
	s, e := upgGitPull()
	if e != nil {
		return s, e
	}
	s2, e := upgMake()
	if e != nil {
		return s + s2, e
	}
	return s + s2, e
}
