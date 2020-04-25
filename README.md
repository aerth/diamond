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
