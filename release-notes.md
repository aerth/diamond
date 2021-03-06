# diamond v0.8

i have decided to rewrite the diamond library, way leaner.

the new diamond library does the following:

  * opens unix socket for master control
  * respects the "KICK" command
  * runlevels that can be customized

thats all. the diamond-admin command still works as expected.

thanks for using DIAMOND!

submit bugs, feature requests, and pull requests at github.com/aerth/diamond

- aerth

# diamond v0.6

further gutting

# diamond v0.5

lighter

tls (https)

better

# diamond v0.4

experimental

# diamond v0.3

Report bugs/issues via github issues: https://github.com/aerth/diamond/issues

## Big Changes:

  * on boot, old socket will be deleted if it is thought to be orphaned

  * server handling TERM signal, shutting down properly (runlevel 0, calling Hooks)


## New Features:
### Hook functions

You can define custom 'hook' functions to be called when a runlevel is entered

  * var diamond.Hooklevel0 func(){}
  * var diamond.Hooklevel1 func(){}
  * var diamond.Hooklevel3 func(){}

Would ```func(ch chan interface{}) error {}``` offer safer hook functions?

### diamond.DoneMessage and s.Quit chan

  * DoneMessage will get sent to s.Quit chan string

  * Define: 'diamond.DoneMessage' to customize message

  * So an application can end with:

```
select {
  case msg := <- s.Quit:
    println(msg)
    os.Exit(0) // may be omitted
}
```

### CountConnections

```func (s *Server) CountConnections() int64 {}```

## Notes

  * How to TLS with Diamond Server?
  * Should rewrite now that Go http server has shutdown ability? Probably
  * diamond-admin command still laggy after 2 commands, restart diamond-admin frequently as workaround

------------------

# diamond v0.1

Breaking Changes:

  * s.Config is now a type (ConfigFields) and its fields are exported.
      type ConfigFields struct {
      	Name        string // user friendly name
      	Addr        string // :8080 (Short for 0.0.0.0:8080) or 127.0.0.1:8080 (Only localhost)
      	Socket      string // path of Socket file to create (/tmp/diamond.sock)
      	Level       int
      	Debug       bool
      	Kicks       bool   // will kick to launch
      	Kickable    bool   // able to be kicked
      	DoCycleTest bool   // do 1-3-default cycle at launch
      	Log         string // directory to write logs
      }

  * The function to load a configuration from bytes is now s.Configure()
      func (s *Server) Configure(b []byte) error {}

Notes:
  * To load configuration from file, still use this method (as seen in simple.go)
      err := s.ConfigPath("config.json")
  * Easily replace "s.Config(" with "s.Configure(" if you are loading from bytes.
  *
