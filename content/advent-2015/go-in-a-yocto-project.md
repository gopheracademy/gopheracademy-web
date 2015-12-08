+++
author = ["Marcelo Magallon"]
date = "2015-12-13T00:00:00-06:00"
linktitle = "Go in a Yocto project"
series = ["Advent 2015"]
title = "Integrating Go in a Yocto-based project"
+++

From its website, [Yocto](https://www.yoctoproject.org/), part of the
[Linux Foundation Collaborative
Projects](http://collabprojects.linuxfoundation.org/), is <cite>an open
source collaboration project that provides templates, tools and methods
to help you create custom Linux-based systems for embedded products
regardless of the hardware architecture</cite>. Given Go's [wonderful
support for cross
compilation](http://dave.cheney.net/2015/08/22/cross-compilation-with-go-1-5),
the two are like a match made in heaven.

While not a requirement, Yocto-based projects have a strong preference
for building everything from source, including the toolchain. One of the
biggest changes introduced with the Go 1.5 release was the move of the
entire toolchain to Go, which poses an interesting bootstraping problem:
you need a Go compiler to build a Go compiler. The developers solved
this by introducing [a clear
requirement](https://docs.google.com/document/d/1OaatvGhEAq7VseQ9kkavxKNAfepWy2yhPUBs96FGV28/edit)
for the toolchain to be buildable, for as long as possible, using Go
1.4.

On the other hand, Yocto makes a distinction between programs meant to
run on the host and programs meant to run on the target system. Yocto
recipes can produce binaries that are required to run in the host during
a later phase of the build process, called "native packages". A
cross-compiler is a program that runs on the host system and produces
binaries for the target system.  Together with the bootstraping
solution, this means in order to support Go, you need at least two
packages:

* A bootstrap Go 1.4 compiler (`go-bootstrap`), which is a native
  package used to build the compiler for the desired version.
* A Go 1.5 (or later) cross compiler (`go-cross`), which is a native
  package used to build binaries for the target. This provides the
  entire Go toolchain.

There's a third type of program that's of interest: one that runs on the
host but is not meant to produce binaries for the target. An example is
a program that processes some input and generates output that's needed
later in the build process, like the Protocol Buffer compiler producing
`.go` files out of `.proto` files. To cover this case, a third package
can be used (`go-native`), which compiles Go programs for the host
system.

Since Go comes with its own and widely used build system (`go build` and
friends), it makes sense to provide convenience functions to make the
build process of Go projects easier. In Yocto-speak, this means
providing a _bitbake class_.

All of this is provided by the
[oe-meta-go](https://github.com/mem/oe-meta-go.git) layer. Like the name
imples, a layer in Yocto is a component that's layered on top of other
components and can be used to provide additional capabilities and
functionality for the system.

One important characteristic of the Yocto build system is that recipes
specify precisely what they need and what they provide. For example, the
simple "hello world" program in Go does not require anything beyond the
`go-cross` package. On the other hand, something like
[Caddy](https://caddyserver.com/) has many dependencies. In Go we are
used to the simplicity of `go get`, which fetches code from multiple
repositories at once. Yocto on the other hand prefers an approach at the
other end of the spectrum, where each code repository is fetched
individually. This allows the system integrator, among many other
things, to provide patches for each individual package; check the
license for each package; verify that the license conditions are
compatible with each other; provide customized build and installation
steps for each package; have fine-grained control over the versions of
the packages used in the system, down to the commit level. For these
reasons, instead of the usual `go get github.com/mholt/caddy`, the
approach taken in the `oe-meta-go` layer is to provide a recipe for each
individual package.

As an illustration, Caddy's recipe looks like this:

```
DESCRIPTION = "Fast, cross-platform HTTP/2 web server with automatic HTTPS"

GO_IMPORT = "github.com/mholt/caddy"

inherit go

SRC_URI = "git://${GO_IMPORT};protocol=https;destsuffix=${PN}-${PV}/src/${GO_IMPORT}"
SRCREV = "${AUTOREV}"
LICENSE = "Apache-2.0"
LIC_FILES_CHKSUM = "file://src/${GO_IMPORT}/LICENSE.txt;md5=e3fc50a88d0a364313df4b21ef20c29e"

FILES_${PN} += "${GOBIN_FINAL}/*"

DEPENDS += "\
	github.com-BurntSushi-toml \
	github.com-dustin-go-humanize \
	github.com-flynn-go-shlex \
	github.com-gorilla-websocket \
	github.com-hashicorp-go-syslog \
	github.com-jimstudt-http-authentication \
	github.com-russross-blackfriday \
	github.com-shurcooL-sanitized-anchor-name \
	github.com-square-go-jose \
	github.com-xenolf-lego \
	golang.org-x-crypto \
	golang.org-x-net \
	gopkg.in-natefinch-lumberjack.v2 \
	gopkg.in-yaml.v2 \
"
```

Some remarks about this recipe:

* The `GO_IMPORT` variable is specific to the `go` class. It specifies
  the import path for the package in question.
* The line `inherit go` causes this recipe to use the "go" class as its
  base. That class provides default values for several variables, as
  well as functions to compile and install the package.
* `SRC_URI` specifies the location of the source code, a Github
  repository in this case. The additional information in `SRC_URI` tells
  the build how to access the repository (over HTTPS) and where to put
  the downloaded files. The specific structure shown here makes it
  possible to set `GOPATH` to the download location, and have `go
  install` just work.
* `FILES_${PN}` is Yocto's way of specifying which files are expected to
  be installed along with which package (`${PN}` is a variable holding
  the main package's name).
* The `DEPENDS` variable specifies which other recipes must be built
  before this one, and you can probably guess that each of those is
  providing a single Go package.

If you take a look at [all the other recipes provided as an
example](https://github.com/mem/oe-meta-go/tree/master/recipes-caddy/caddy),
you'll see that they are all very similar. Here the uniformity and
simplicity that's characteristic of Go still shines thru.

I've shown that, after taking care of some integration details,
providing Yocto recipes for Go packages is a simple process.  [Yocto's
documentation can be
daunting](http://www.yoctoproject.org/docs/1.8/mega-manual/mega-manual.html)
but I hope that this brief introduction has spiked your curiosity enough
to go over the quick start and try your hand at a recipe for your own
packages.

Enjoy!
