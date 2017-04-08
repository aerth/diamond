//+build !noexec

/*
* The MIT License (MIT)
*
* Copyright (c) 2016,2017  aerth <aerth@riseup.net>
*
* Permission is hereby granted, free of charge, to any person obtaining a copy
* of this software and associated documentation files (the "Software"), to deal
* in the Software without restriction, including without limitation the rights
* to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
* copies of the Software, and to permit persons to whom the Software is
* furnished to do so, subject to the following conditions:
*
* The above copyright notice and this permission notice shall be included in all
* copies or substantial portions of the Software.
*
* THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
* IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
* FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
* AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
* LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
* OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
* SOFTWARE.
 */

package diamond

import (
	"os"
	"os/exec"
)

// DefaultToolUpdate runs the command "git pull origin master" from current working directory.
//
// See "ToolUpdate"
func DefaultToolUpdate() (string, error) { // RPC: update
	cmd := exec.Command("git", "pull", "origin", "master")
	b, er := cmd.CombinedOutput()
	if er != nil {
		return string(b), er
	}
	return string(b), nil
}

// DefaultToolRebuild trys 'bin/build.sh', falls back on 'build.sh'
//
// See "ToolRebuild"
func DefaultToolRebuild() (string, error) { // RPC: rebuild
	buildfile := "./bin/build.sh"
	buildargs := ""
	if _, err := os.Open(buildfile); err != nil {
		buildfile = "./build.sh"
	}
	cmd := exec.Command(buildfile, buildargs)
	b, er := cmd.CombinedOutput()
	if er != nil {
		return string(b), er
	}
	return string(b), nil
}

// DefaultToolUpgrade runs 'ToolUpdate() && ToolRebuild() '
//
// See "ToolUpgrade"
func DefaultToolUpgrade() (string, error) { // RPC: upgrade
	s, e := ToolUpdate()
	if e != nil {
		return s, e
	}
	s2, e := ToolRebuild()
	if e != nil {
		return s + s2, e
	}
	return s + s2, e
}
