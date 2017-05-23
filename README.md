# DIAMOND ⋄

### Runlevels for your web application

#### Latest: [tiny branch](https://github.com/aerth/diamond/tree/tiny)

 :zap: ```telinit 4```

[![GoDoc](https://godoc.org/github.com/aerth/diamond/lib?status.svg)](https://godoc.org/github.com/aerth/diamond/lib)
[![Build Status](https://travis-ci.org/aerth/diamond.svg?branch=master)](https://travis-ci.org/aerth/diamond)

![Screenshot diamond-admin CUI](https://github.com/aerth/diamond/blob/master/docs/diamond-screenshot.png?raw=true)

You can open admin interface by using no arguments:
```
diamond-admin -s diamond.sock
```

Or use in scripts:
```
diamond-admin -s diamond.sock telinit 3
Command: telinit 3
DONE: telinit 3
```

## Using the library

Diamond requires Go 1.6.4 or newer

```
package main
import "github.com/aerth/diamond/lib"
import "net/http"
func main(){
router := http.FileServer(http.Dir("."))
d := diamond.NewServer(router)
d.Start()
println(<-d.Done)
}

```

Read more:

[aerth.github.io/diamond](https://aerth.github.io/diamond/)

[github.com/aerth/diamond](https://github.com/aerth/diamond/)

#### CAUTION

API may change without notice

#### Contributing

Submit new issue or pull request
