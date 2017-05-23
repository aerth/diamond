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
	"fmt"
	"strconv"
)

type Packet struct {
	parent *Server
}

func (p *Packet) Echo(arg string, reply *string) error {
	if arg == "" {
		return fmt.Errorf("empty argument")
	}
	*reply = arg
	return nil
}

func (p *Packet) Kick(arg string, reply *string) error {
	if !p.parent.Config.Kickable {
		*reply = "NOWAY"
		return fmt.Errorf("NOWAY")
	}
	*reply = "OKAY"
	p.parent.Runlevel(0)
	return nil
}

func (p *Packet) Runlevel(arg string, reply *string) error {
	n, err := strconv.Atoi(arg)
	if err != nil {
		*reply = "error"
		return err
	}
	err = p.parent.Runlevel(n)
	if err != nil {
		*reply = "error"
		return err
	}
	*reply = strconv.Itoa(p.parent.GetRunlevel())
	return nil
}
