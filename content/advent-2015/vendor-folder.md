+++
author = ["Daniel Theophanes"]
date = "2015-12-28T00:00:00-08:00"
linktitle = "vendor folder"
series = ["Advent 2015"]
title = "Understanding and using the vendor folder"
+++

With the release of Go 1.5, there is a new way the go tool can discover go
packages. This method is off by default and the surrounding tools, such as
`goimports`, do not understand that folder layout. However in Go 1.6 this method
will be on for everyone and other tools will have support for it as well. This
new package discovery method is the `vendor` folder.

Before we look into the solution and semantics of the vendor folder, let's explore
the problem that prompted it.

## The prompting problem

Go programs are often comprised of packages from many different sources.
Each one of these sources are pulled in from the `GOPATH` or from the standard
library. However, only their project was subject their own source control.
Projects that cared about not breaking when their dependent packages changed
or went away did one of the following:

 1. Copied the dependent packages into the project source tree and rewrote imports
    that referenced it.
 2. Copied the dependent packages into the project source tree and modified
    the `GOPATH` variable to include a project specific sub-tree.
 3. Wrote the repository revision down in a file, then updated the existing
    `GOPATH` packages to be that revision.
	
Although different projects did slight variations of this, these were
the major trends present in all.

There was one large problem and several smaller ones. The largest problem
was that each of these were different. The individual problems included;

 * Many people did not like modifying the import paths or being required to
   include dependent packages in their repository.
 * Modifying `GOPATH` implied using a bare `go build` would not be sufficient.
   Wrappers for the `go` command emerged, each slightly different.
 * Modifying packages in the normal `GOPATH` required each project
   to have a unique `GOPATH`.

## A solution to these problems

With Go 1.5 a new method to discover go packages was released.
Nothing has been changed or added
to the go language or the go compilers. Packages must still reside in `GOPATH`.
However, if a package or a parent folder of a package contains folder named
`vendor` it will be searched for dependencies using the `vendor` folder as an
import path root. While `vendor` folders can be nested, in most cases it is
not advised or needed.
Any package in the `vendor` folder will be found *before* the standard library.

To enable this in Go 1.5 set the environment variable `GO15VENDOREXPERIMENT=1`.
In Go 1.6 this will be on by default without an environment variable.

## An example

This simple package lives in `$GOPATH/src/github.com/kardianos/spider`.
```
$ cat main.go
package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/cascadia"
	"github.com/tdewolff/parse/css"
	"golang.org/x/net/html"
)
...
```

```
$ tree
.
├── css_test.go
├── main.go
└── vendor
    ├── github.com
    │   ├── andybalholm
    │   │   └── cascadia
    │   │       ├── LICENSE
    │   │       ├── parser.go
    │   │       ├── README.md
    │   │       └── selector.go
    │   └── tdewolff
    │       ├── buffer
    │       │   ├── buffer.go
    │       │   ├── lexer.go
    │       │   ├── LICENSE.md
    │       │   ├── reader.go
    │       │   ├── README.md
    │       │   ├── shifter.go
    │       │   └── writer.go
    │       └── parse
    │           ├── common.go
    │           ├── css
    │           │   ├── hash.go
    │           │   ├── lex.go
    │           │   ├── parse.go
    │           │   ├── README.md
    │           │   └── util.go
    │           ├── LICENSE.md
    │           ├── README.md
    │           └── util.go
    ├── golang.org
    │   └── x
    │       └── net
    │           └── html
    │               ├── atom
    │               │   ├── atom.go
    │               │   ├── gen.go
    │               │   └── table.go
    │               ├── const.go
    │               ├── doc.go
    │               ├── doctype.go
    │               ├── entity.go
    │               ├── escape.go
    │               ├── foreign.go
    │               ├── node.go
    │               ├── parse.go
    │               ├── render.go
    │               └── token.go
    └── vendor.json
```

## How to use the vendor folder

Advice on how to use the vendor folder is varied. Some will shutter at
the thought of including dependencies in the project repository. Others
hold it is unthinkable to *not* include the dependencies in the project
repository. Some will just want to include a manifest or lock file and
fetch the dependencies before building.

Regardless of what *you* choose to do with it, the vendor folder enables you to
do more.

## Tools that use this.

There are [many tools](https://github.com/golang/go/wiki/PackageManagementTools#go15vendorexperiment) that use the vendor folder today. There are even more
tools that have support for the vendor folder in a feature branch, so the future
is bright for the `vendor` folder.

I am the author of [govendor](https://github.com/kardianos/govendor) and
[vendor-spec](https://github.com/kardianos/vendor-spec). The primary goal
for `govendor` was to prove and use a common vendor specification file.
The secondary goals were to make a tool that worked at the package level
and could provide quick insight into the status of vendor packages. today
`govendor list` and `govendor list -v` are some of my favorite commands.
If you are coming from `godep`, you just need to run `govendor migrate`
to start using the vendor folder today.
