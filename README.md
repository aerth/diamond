# DIAMOND ⋄

[![GoDoc](https://godoc.org/github.com/aerth/diamond/lib?status.svg)](https://godoc.org/github.com/aerth/diamond/lib)

Turn your HTTP web application into a machine.

Like a transmission, a diamond server has "gears". They are called "runlevels". You can define functions that are called when shifting to a runlevel. These are called:

  * ```diamond.HookLevel0``` runs when 0 is reached, before sending to quitchan
  * ```diamond.HookLevel1``` runs when 1 is reached
  * ```diamond.HookLevel3``` runs when 3 is reached

![Screenshot of both diamond-admin and diamond server](https://github.com/aerth/diamond/blob/master/example/diamond-screenshot.png?raw=true)

You can pass commands as arguments to diamond-admin

```
./diamond-admin -s diamond.sock telinit 3
./diamond-admin -s diamond.sock telinit 1
./diamond-admin -s diamond.sock telinit 0
```

include CUSTOM prefix for custom commands (CustomCommander)
./diamond-admin -s diamond.sock telinit CUSTOM commandname arguments

The server can boot without listening, then the admin can shift gears into the
        'public HTTP mode', (runlevel 3) and back to 'single user mode' (runlevel 1).

While not listening, *another server* (possible another **diamond**) can occupy that port.
Configuration allows a "default runlevel", either 1 or 3.

```
var quitchan = make(chan string, 1)
diamond.HookLevel0 = func() {
  quitchan <- "runlevel 0 attained"
}
```

Diamond allows the administrator (via UNIX socket)
to do more than just start and stop the process.

```
diamond-admin -s $SOCKETNAME
```


Now your application has runlevels!

```d.Loop() blocks until runlevel 0 has been reached```

Once connected to the UNIX socket, the administrator can:

  * Switch runtime levels
  * Redeploy (respawn the binary)

In runlevel 1,
        only the local administrator may access the server using the UNIX socket.

In runlevel 3,
        we open a TCP listener and serve HTTP for the public.

This project is split into three sections.

**Library, Admin, Example Server**

## 1. diamond library

```
d := diamond.NewServer(nil)
d.ErrorLog.SetOutput(mylogfile)
d.SetMux(myrouter)
d.Start()

```

Stays on, Receives COMMANDS from the client,
which is owned by the same UNIX user.
The server has three modes of operation, which
are called "runlevels". They are, in order:

  * 0 = halt, stopping everything (not os.Exit!)
  * 1 = single user mode, only allowing RPC via socket
  * 3 = multiuser mode, allowing HTTP/HTTPS

When entering "runlevel 1", the server opens a
tcp socket, which is how the administrator can make changes.
It stays open until runlevel 0 is entered, in which the server is stopped completely.

Entering runlevel 1 from a higher level closes all TCP connection immediately.

Entering "runlevel 3" opens up a TCP port for
HTTP traffic.

Public may access the web application through ip:port,
or the Diamond may be placed behind a reverse proxy.

The administrator accesses the daemon through
the supplied command line program (via UNIX socket).

### Built in upgrade mechanisms
Admin commands `update`, `upgrade`, `rebuild`, `redeploy` deal with upgrading
the actual running server.

#### Behind the hood:

**update** runs git pull origin master

**rebuild** runs './build.sh server'

**upgrade** runs both, using something like 'update && rebuild', only rebuilding if update was successful.

**redeploy** command spawns another instance and then switches to runlevel 0, leaving.


## 2. Client / Command (diamond-admin)

Adminstrative user must be the same unix user as the server.

Commands:

  * telinit 0-4 - Tell the server to initialize runlevel X
  * restart - Return to default runlevel
  * stop - Short for telinit 0
  * load - Load a configuration file (see CONFIG FILE)
  * backup - Return an compressed archive of the database to be used with import
  * import - Import a backup that was generated with the backup command
  * update - Fetch latest source, build, install. does not restart.
  * upgrade - Fetch latest source, build, install. does not restart.
  * redeploy - telinit 0 && relaunch binary (could be upgraded binary, same path.)
  * reconfig - reload preset json file (use s.ConfigPath() to set)



## 3. Example server

When using go get, you will notice a small example server gets built.

This is the most simple Diamond server.

More examples are located in the ./example directory.

## CONFIG FILE

Must be JSON encoded. Required fields are Addr and Socket. All other fields are optional.

If not specified, diamond library will search for a config.json file in the working directory.

With s.Config(b []byte) you can supply raw bytes,
or s.ConfigPath(path string) you can supply a custom config path.

        Name     string // user friendly name of server
        Addr     string // :8080 (Short for 0.0.0.0:8080) or 127.0.0.1:8080 (Only localhost)
        Socket   string // path of socket file to create (Example: /tmp/diamond.sock)
        Level    int // Default Runlevel
        Debug    bool // More verbose logs
        Kicks    bool   // will kick another diamond to launch
        Kickable bool   // able to be kicked by other diamonds
        Log      string // directory to write logs (Can be a single dot)


### example config.json with minimal recommended fields:
        ```
        {
                "Name":"Diamonds! ⋄",
                "Level":1,
                "Addr":":8777",
                "Socket":"/tmp/diamond.socket",
                "Kicks": true,
                "Kickable": true,
        }

        ```


## build.sh

Change directory to $GOPATH/src/github.com/aerth/diamond before using bin/build.sh

```
bin/build.sh admin
bin/build.sh server
bin/build.sh all # builds both
```

## Reporting issues, Contributing

Please submit a pull request if:

  * I forgot something
  * You have an example for usage with a different http routing library
  * You have a bug fix
  * You have simplified the library, removed lines of code, etc

Please submit a new github issue if:

  * You have encountered a bug while using diamond
  * You have a question about an undocumented feature

## Author

Copyright 2016-2017  aerth <aerth@riseup.net>

Contributions are welcome

MIT LICENSE
