+++
author = [ "Kevin Burke" ]
date = "2017-12-02T00:00:00"
linktitle = "Automating Go development with Make"
title = "Automating Go development with Make"
series = ["Advent 2017"]

+++

[Make][make] is an old tool that you can use today to help get everyone on your team
on the same page, and make it easy for new contributors to your project to get
started. In some cases Make can help you avoid unnecessary work! Let's see how
you can integrate Make into your development workflow.

For this example, we are going to pretend our application uses [protocol
buffers][protobuf] to send data back and forth. Protocol buffers, or protobufs
for short, are a data serialization format that let you declare the same
information in a lot fewer bytes than JSON. Protobufs are the inspiration for
the [`encoding/gob` package][gob].

To use protobufs, first we declare a `.proto` file containing all of the
objects we might send over the wire. ([Here's what a real protobuf file looks
like][maintner-proto].) Our application is pretty simple.

##### app.proto

```proto
syntax = "proto3";

package app;

message User {
    int64 id = 1;
    string email = 2;
    string name = 3;
}
```

To use this in an application, we generate a language-specific protobuf file,
using the protobuf compiler `protoc`. To compile the Go protobuf file, we need
`protoc` and the Go specific extension, `protoc-gen-go`. (We'll walk through how
to get those in a bit). Then we run:

```
protoc --go_out=. app.proto
```

This will generate a Go file called app.pb.go that's about a hundred lines
long, with some helpers for accessing properties on the struct. Here's a sample:

##### app.pb.go

```go
type User struct {
	Id    int64  `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	Email string `protobuf:"bytes,2,opt,name=email" json:"email,omitempty"`
	Name  string `protobuf:"bytes,3,opt,name=name" json:"name,omitempty"`
}

func (m *User) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}
```

In a large application, the `protoc` command is likely to be a lot longer. When
something is long and potentially hard to remember, members of your team may
remember and run different invocations of the command. This can lead to
problems, so you introduce a Makefile with a single target, `compile`, that runs
the above command.

The advantage of using a Makefile (versus `rake`, `grunt`, `robo`, or any
other automation tools) is that Make comes preinstalled on pretty much every
Unix machine, so you don't need any dependencies to run it. You can use Make
no matter what language your code is written in, so your organization can
standardize on `make test` to run tests and `make serve` to start a development
server on *every* project at your company, regardless of the language it's
written in. `make` is also very fast; `make` can print the version string (or
e.g. start running commands) in about ten milliseconds, where Grunt or other
tools written in interpreted languages can take about 300ms before they start
doing anything. If your test suite only takes 10 milliseconds to run, you don't
want your build tool increasing the time it takes to run tests by 30x.

##### Makefile

```make
.PHONY: compile

compile:
	protoc --go_out=. app.proto
```

So you add a Makefile, tell everyone on your team to forget about the long
`protoc` invocation and just run `make compile` instead. Everyone standardizes
on the same command, you can update the run command and ensure everyone is
instantly using it, and things are good.

But we can do better! At the heart of `make` is a dependency graph: `make`
assembles targets from various inputs. The key insight is that *if the inputs
haven't changed, you don't have to regenerate the output.* In our example, if
we run `make compile` but app.proto hasn't changed, we don't need to regenerate
`app.pb.go`, since we will (or should, at least) get the same result.

We have to rearrange our Makefile a bit. The left side of the colon is the
*target*, the file being built. To the right of the colon, we list the various
inputs that the target depends upon.

```make
.PHONY: compile

app.pb.go: app.proto
	protoc --go_out=. app.proto

# This is a "phony" target - an alias for the above command, so "make compile"
# still works.
compile: app.pb.go
```

If Make determines the target file is newer than the inputs, it will decide
there is nothing to do.

```bash
$ make app.pb.go
make: 'app.pb.go' is up to date.
```

This is especially useful when you need to compile a lot of things to get your
application running, as it lets you skip a lot of unnecessary work.

But we can go further! The compile target also depends on `protoc` and on the
`protoc-gen-go` helper. We can add these as dependencies as well. Since we only
need them to exist on the filesystem, we don't care if they are newer or older
than the target, we add a pipe character in the declaration, and list them
afterwards.

```make
.PHONY: compile
PROTOC_GEN_GO := $(GOPATH)/bin/protoc-gen-go

# If $GOPATH/bin/protoc-gen-go does not exist, we'll run this command to install
# it.
$(PROTOC_GEN_GO):
	go get -u github.com/golang/protobuf/protoc-gen-go

app.pb.go: app.proto | $(PROTOC_GEN_GO)
	protoc --go_out=. app.proto

# This is a "phony" target - an alias for the above command, so "make compile"
# still works.
compile: app.pb.go
```

Now, the first time someone on a new computer runs `make compile`, they'll get
this:

```bash
$ make compile
go get -u github.com/golang/protobuf/protoc-gen-go
protoc --go_out=. app.proto
```

And the second time they edit app.proto and run `make compile`, they'll skip the
install step:

```bash
$ make compile
protoc --go_out=. app.proto
```

That's a really useful technique for automating installation of binaries that
are necessary to run various build tasks.

We can use roughly the same technique to install the protoc binary. This is
trickier since the command might change based on the machine we're running on,
so we need to call `uname` to get the machine type, and then branch the install
command based on the machine.

If this seems like too much work, and it might be, you can just have your
Make target exit with an error if protoc is not installed, or echo a message
explaining to people how and where to get protoc.

```make
.PHONY: compile
PROTOC_GEN_GO := $(GOPATH)/bin/protoc-gen-go
PROTOC := $(shell which protoc)
# If protoc isn't on the path, set it to a target that's never up to date, so
# the install command always runs.
ifeq ($(PROTOC),)
    PROTOC = must-rebuild
endif

# Figure out which machine we're running on.
UNAME := $(shell uname)

$(PROTOC):
# Run the right installation command for the operating system.
ifeq ($(UNAME), Darwin)
	brew install protobuf
endif
ifeq ($(UNAME), Linux)
	sudo apt-get install protobuf-compiler
endif
# You can add instructions for other operating systems here, or use different
# branching logic as appropriate.

# If $GOPATH/bin/protoc-gen-go does not exist, we'll run this command to install
# it.
$(PROTOC_GEN_GO):
	go get -u github.com/golang/protobuf/protoc-gen-go

app.pb.go: app.proto | $(PROTOC_GEN_GO) $(PROTOC)
	protoc --go_out=. app.proto

# This is a "phony" target - an alias for the above command, so "make compile"
# still works.
compile: app.pb.go
```

The other place I've found dependency management like this to be really useful
is in restarting a development server. You may have a few different targets - Go
files, assets that must be compiled into a binary, and Protobuf files, and want
to make sure they're all up to date before starting the server. However, if
they haven't changed, you don't want to do the additional work.

```
compile: app.pb.go # see above

assets/bindata.go: $(shell find static/ templates/)
	# https://github.com/kevinburke/go-bindata
	go-bindata ./static ./templates

assets: assets/bindata.go

# This command will run any time a .go file in the current directory is newer
# than the compiled binary.
$(GOPATH)/bin/myserver: $(shell find . -name '*.go)
	go install ./cmd/myserver

serve: $(GOPATH)/bin/myserver compile assets
```

Specifying the server's dependencies as upstream inputs should make your server
restarts much faster, since you skip work you don't need to do!

### Notes

How does `make` determine whether the target needs to be updated? It checks the
file mtime ([ModTime in a os.FileInfo][mod-time]) of the target. If the target
is newer than the mtimes of the inputs, there's no need to run the command.

Using mtimes for this check has flaws. For one, I can change the mtime of a file
without actually modifying it, which would lead `make` to do unnecessary work.
The other is that clocks can drift on your machine, which would lead mtimes to
suggest work should be done when it doesn't really have to be done.

A better approach is to hash the contents of the inputs and the target, and only
rerun the `make` target body if the hashes don't line up. This is the approach
taken by [Bazel][bazel], a tool written by Google that builds on a lot of what
Make does. Bazel has a steep learning curve and can have slow startup times for
smaller projects; if you're not using any sort of dependency-tracking tool, just
adding a Makefile can get you about 80% of the benefits that Bazel does.

It's also - in a piece of excellent news - the approach built in to test running
and build compilation in Go 1.10. Go 1.10 uses [content-based hashing][hashing]
to figure out when it doesn't need to recompile a package. It uses the same
approach for tests - if you ran the tests and they passed, and the input files
haven't changed at all, you don't need to rerun the tests. So when Go 1.10 gets
released, you don't need the `$(GOPATH)/bin/myserver` target in the Makefile
above - you can *always* run `go install ./...` and it will exit immediately if
there is no work to do!

### Conclusion

You can use Make to automate the process of installing build tools that are
ancillary to your build process, which should help open source projects or
larger teams where contributors don't necessarily need or want to know how to
install those tools.

There are probably steps in your build process that you are running
unnecessarily. Use Make (and Go 1.10) to avoid doing work you don't need to, and
get back that extra time!

To view a repository with the Makefile from this post alongside a working
application, go here: https://github.com/kevinburke/proto-make-example

*Kevin Burke is a contributor to the Go language. He runs [a software consulting
company.][burke]*

[protobuf]: https://developers.google.com/protocol-buffers/
[gob]: https://golang.org/pkg/encoding/gob/
[maintner-proto]: https://github.com/golang/build/blob/master/maintner/maintpb/maintner.proto
[mod-time]: https://golang.org/pkg/os/#FileInfo
[bazel]: https://bazel.build
[make]: https://www.gnu.org/software/make/
[burke]: https://burke.services
[hashing]: https://github.com/golang/go/blob/master/src/cmd/go/internal/cache/cache.go
