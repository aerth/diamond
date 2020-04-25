# DIAMOND ⋄

### Runlevels for your web application

:zap: ```telinit 3```

[![GoDoc](https://godoc.org/github.com/aerth/diamond/lib?status.svg)](https://godoc.org/github.com/aerth/diamond/lib)
[![Build Status](https://travis-ci.org/aerth/diamond.svg?branch=master)](https://travis-ci.org/aerth/diamond)

![Screenshot diamond-admin CUI](https://github.com/aerth/diamond/blob/master/docs/diamond-screenshot.png?raw=true)


# Using diamond-admin

You can open admin interface by using no arguments:

```
diamond-admin -s diamond.sock
```

### Start all listeners and http servers

```
diamond-admin -s diamond.sock RUNLEVEL 3
```

### Stop all listeners, cut http connections

```
diamond-admin -s diamond.sock RUNLEVEL 1
```

## Using the library

Diamond requires a recent version of Go

```

// New creates a new admin socket and starts listening for commands
s, err := diamond.New("/tmp/diamond.socket")
if err != nil {
    log.Fatalln(err)
}

// Add variety of http handlers and their addr to listen on
// They won't start listening right away, so they could be
// occupied by other servers
s.AddHTTPHandler(":8080", http.HandlerFunc(myHandler))
s.AddHTTPHandler(":8081", http.HandlerFunc(handler2))
s.AddHTTPHandler(":8082", handler3)

// start in multiuser mode, serving http
// without calling Runlevel(3) you must
// connect via socket and issue the RUNLEVEL 3 command
s.Runlevel(3) 

// serve forever
log.Fatalln(s.Wait())
```

See the [examples](example)

Read more:

[aerth.github.io/diamond](https://aerth.github.io/diamond/)

[github.com/aerth/diamond](https://github.com/aerth/diamond/)

#### CAUTION

API may change without notice! (it already has two times!)

#### Contributing

Submit new issue or pull request

### Old version

```
import "gopkg.in/aerth/diamond.v1"
```
