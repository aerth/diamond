package diamond

import (
	"context"
	"os/exec"
)

func upgGitRemote(branch, repo string) (string, error) { // RPC: update
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "remote", "add", branch, repo)
	b, er := cmd.CombinedOutput()
	if er != nil {
		return string(b), er
	}
	return string(b), nil
}

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
	cmd := exec.CommandContext(ctx, "bin/build.sh", "server")
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
