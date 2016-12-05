+++
author = ["Christoph Berger"]
date = "2016-12-10T00:00:00+01:00"
title = "Some Tools For Go That You Might Not Know Yet"
series = ["Advent 2016"]
draft = true
+++

Year's end is coming closer. Time to clean up repositories and 
polishing up the toolset. All the well-known tools have been installed
already--is there anything else to add to the toolbox?

Here are a few tools that do not seem to be mainstream yet but that
I think are noteworthy. Some of them already seem to be at the edge
of becoming


# *interfacer:* Should I rather use an interface here?

`interfacer` has a very specific purpose: It looks at the parameters
of your functions and points out those that you could replace by an
interface type. 

Why this? 

Maybe you know that rule of thumb: A function should expect an interface
and return a specific type. I fail to remember where I came across
that rule, and the precise wording might also be different, but the
point is that if a function parameter is an interface then the function
is much more versatile and can, for example, receive a mock type when
used in a unit test. 

So whenever you feel you have missed an opportunity to have one of your
functions receive an interface rather then a struct, run `interfacer` 
and it will tell you if you did.

## An example

Imagine that one day, you started writing a larger library that contains 
a `Doer` interface and a `DoIt` struct that implements `Doer`. A couple 
hours and few thousand lines later (yes, you felt productive this day) 
you defined function `DoSomething` that expects a `DoIt` struct and calls 
its `Do` method.

```go
type Doer interface {
	Do()
}

type DoIt struct{}

func (DoIt) Do() {
}

func DoSomething(d DoIt) {
	d.Do()
}
```

You vaguely remember that `Do` belongs to some interface but you can't 
remember which one. You're to lazy to search for it (and after all, 
it is already 11pm), so you run 

    $ interfacer dosomething.go

and get this advice:

    dosomething.go:12:18: d can be Doer

You quickly fix the `DoSometing` function and go to bed, knowing that
you know can easily pass some mock `DoIt` struct to `DoSomething` when
you'll write the tests tomorrow.

(

# *zb:* Take some shortcuts on the go toolchain

In contrast to the previous tool, `zb` is a little swiss army knife.
It provides a bunch of commands that shall speed up your 
write/build/test cycle. Some of its higlights:

* It speeds up builds by running concurrent `go install` commands where
possible.
* It speeds up tests and lint tools by caching the results.
* It remembers calling `go generate` in case you forgot.
* It is aware of the `vendor` directory and keeps some operations
out of `vendor` (like, for example, linting). 

I cannot list all of the available commands here; otherwise I would 
just end up rephrasing the README file here. If you got curious, 
just head over to the [zb repository](https://github.com/joshuarubin/zb)
and have a look.

Caveat: The author describes the tool as opinionated, and I would not
tend to disagree. On the other hand, there is no obligation to use 
all of the available commands. Just pick the ones you find useful and
that don't go in the way of your workflow.


# *realize:* Trigger your toolchain via Ctrl-S

TODO

Granted - [DevOp](https://github.com/jhsx/devop) is not the first Go 
build system, and it will not be the last. And for everyone who loves
tools like this, there is another one who already has built a bash script 
or two for the same purpose and regards such tools as utterly useless.

Anyway, 

A tool that is a bit more tailored to fit the Go build process is 
[realize](https://github.com/tockins/realize). Activating go run, 
install, generate, fmt, etc is just a matter of flipping a boolean
switch in the config file (as opposed to specifying the complete command
line). 

TODO


# *binstale:* Are my binaries up to date?

This is the tool I like most of all tools in this article. 

Do you know if the binaries in your `$GOPATH/bin` directory are still
up to date? `go get` happily installs binary after binary, but then
they start drawing dust. And at one point you remember you once installed
that nice tool that helped you doing xyz for you, and eventually you find
it in `$GOPATH/bin`, but starting it fails with some incomprehensive
error message. 

You are pretty sure it worked without flaws back then, so maybe the
binary got stale? Ok, try recompiling it.

You run a recursive search in `$GOPATH/bin` to find a repository of
the same name as the binary. Eventually you find the repository and
call `go get` on it. This fixes your binary, and you feel relieved...

...until you realize that there could be dozens, if not hundreds, of 
stale binaries in your `$GOPATH/bin` directory!


Meet [binstale](https://github.com/shurcooL/binstale). 

This little gem tells you in an instant whether a given go-gettable 
binary needs updating. 

    $ binstale realize
	realize
		STALE: github.com/tockins/realize (build ID mismatch)

And if you have a minute or two, it does the same for all of your 
binaries. 

	$ binstale
	CanvasStreamTest
		STALE: github.com/cryptix/CanvasStreamTest (build ID mismatch)
	Go-Package-Store
		STALE: github.com/shurcooL/Go-Package-Store (build ID mismatch)
	aligncheck
		STALE: github.com/alecthomas/gometalinter/vendor/src/github.com/opennota/check/cmd/aligncheck (build ID mismatch)
		STALE: github.com/opennota/check/cmd/aligncheck (build ID mismatch)
	asmfmt
		STALE: github.com/klauspost/asmfmt/cmd/asmfmt (build ID mismatch)
	balancedtree
		STALE: github.com/appliedgo/balancedtree (build ID mismatch)
	benchcmp
		STALE: code.google.com/p/go.tools/cmd/benchcmp (build ID mismatch)
		STALE: golang.org/x/tools/cmd/benchcmp (build ID mismatch)
	benchstat
		STALE: rsc.io/benchstat (build ID mismatch)
	binstale
		STALE: github.com/shurcooL/binstale (build ID mismatch)
	bug
		STALE: github.com/driusan/bug (build ID mismatch)
	...

You might have noticed that some binaries have more than one 
matching repository (aligncheck and benchcmp in the above
sample output). For this reason, `binstale` currently does not
auto-update any binaries. But that's just a matter of copying 
and pasting the repository path to `go get -u` and you're done.


# Conclusion

These are only a few tools from the constant stream of releases
and announcements. Tools like these are a sign of a living community
that is eager to make contributions that help others. 

If you have an idea for a cool tool, don't hesitate! Sit down and
write it! But first, be sure to check the public repositories - 
the tool your mind came up with might already exist somewhere. 


