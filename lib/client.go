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
	"strings"
)

// Client connects to a diamond.Server via unix socket
type Client struct {
	socket     string        // path to socket file
	ServerName string        // gets filled in with rpc, optional.
	Name       string        // optional, can be sent to identify the admin
	serveraddr *net.UnixAddr // gets parsed from path in NewClient(path)
}

// NewClient returns an initialized Client, returning an error only if the socket can not be resolved
// It is possible that the socket does not exist yet
func NewClient(socketpath string) (*Client, error) {
	addr, err := net.ResolveUnixAddr("unix", socketpath)
	if err != nil {
		return nil, err
	}
	return &Client{socket: socketpath, serveraddr: addr}, nil
}

// Send command and optional arguments and return the reply and any errors
// Commands available to the client are exported methods of Packet type:
//   * returning error
//   * first argument of string ("args")
//   * second argument of pointer to string (used as "reply")
func (c *Client) Send(cmd string, args ...string) (reply string, err error) {
	client, err := rpc.Dial("unix", c.serveraddr.String())
	if err != nil {
		return "", err
	}
	err = client.Call("Diamond."+strings.Title(cmd), strings.Join(args, " "), &reply)
	if err != nil {
		return "", err
	}
	return reply, nil
}

// GetSocket returns the filename of socket used for connections
func (c *Client) GetSocket() string {
	return c.socket
}
