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
	"net"
	"net/rpc"
)

// Client connects to a diamond.Server via unix socket
type Client struct {
	socket     string        // path to socket file
	ServerName string        // gets filled in with rpc, optional.
	Name       string        // optional, can be sent to identify the admin
	serveraddr *net.UnixAddr // gets parsed from path in NewClient(path)
}

// NewClient returns a  Client, to connect to the socket, which location is the only argument
//
// It returns an error only if the socket can not be resolved
func NewClient(socketpath string) (*Client, error) {
	addr, err := net.ResolveUnixAddr("unix", socketpath)
	if err != nil {
		return nil, err
	}
	return &Client{socket: socketpath, serveraddr: addr}, nil
}

// Send a command to the server's unix socket
//
// Commands are limited to help , update , upgrade, rebuild, telinit, CUSTOM
func (c *Client) Send(cmd string) (reply string, err error) {
	client, err := rpc.Dial("unix", c.serveraddr.String())
	if err != nil {
		return
	}
	err = client.Call("Packet.Command", cmd, &reply)
	if err != nil {
		return
	}

	return
}
