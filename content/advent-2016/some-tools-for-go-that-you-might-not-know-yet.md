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

Here are a few useful tools that you might not have in your toolbox yet:
`interfacer`, `zb`, `realize`, and `binstale`. They have nothing in
common except that each of them solves a particular problem well. 



# *interfacer:* Should I rather use an interface here?

`interfacer` has a very specific
purpose: It looks at the parameters of your functions and points out
those that you could replace by an interface type. 

Why this? 

Maybe you have heard of the following advice: A function should expect
an interface and return a specific type. I fail to remember where I came
across that rule, and the precise wording might also be different, but
the point is that if a function parameter is an interface then the
function is much more versatile and can, for example, receive a mock
type when used in a unit test. 

So whenever you feel you have missed an opportunity to have one of your
functions receive an interface rather then a struct, run `interfacer` 
and it will tell you if you did.

## An example

Imagine that one day, you start writing a BBQ sensor library for
controlling the temperature of your Thanksgiving turkey. The library
contains an `Alerter` interface consisting of function `Alerter`, and a
`Sensor` struct that implements `Alerter`.  


```go 
type Alerter interface {
	Alert()
}

type Sensor struct{}

func (Sensor) Alert() {
	fmt.Println("Turkey is done!")
}

```

A couple hours and few thousand lines later (yes, you feel productive 
this day) you define a function `sensorAlert` that expects a `Sensor` 
struct and calls its `Alert` method.

```go 
func sensorAlert(s Sensor) {
	s.Alert()
}
```

You vaguely remember that `Alert` belongs to some interface but you can't 
remember which one. You're too lazy to search for it (and after all, 
it is already 11pm), so you run 

    $ interfacer bbq.go

and get this advice:

    bbq.go:15:18: s can be Alerter

You quickly fix the `sensorAlert` function and go to bed, knowing that
you now can easily pass some `MockSensor` struct to `sensorAlert` when
you'll write the tests tomorrow.

[Interfacer on GitHub](https://github.com/mvdan/interfacer/)


# *zb:* Take some shortcuts to the go toolchain

After you finished some work on your latest project, you run
`gometalinter`. It takes some time to finish, and you discover that some
of the lint tools have descended into the vendor directory, and now the
output is cluttered with lots of useless messages.

Then you run `go build`, only to observe that some tests failed. Aw,
forgot to run `go generate`.

While you fix this, you realize how time is passing, and you wish your
tools were faster and a bit smarter.

`zb` to the rescue.

In contrast to the previous tool, `zb` is a little Swiss army knife.
It provides a bunch of commands that shall speed up your 
write/build/test cycle. Some of its highlights:

* It speeds up builds by running concurrent `go install` commands where
possible.
* It speeds up tests and lint tools by caching the results.
* It remembers calling `go generate` in case you forgot.
* It is aware of the `vendor` directory and keeps some operations
out of `vendor` (like, for example, linting). 

I cannot list all of the available commands here; otherwise I would just
end up rephrasing the README file here. If you got curious, just head
over to [zb's README](https://github.com/joshuarubin/zb/blob/master/README.md) 
and have a look.

Caveat: The author describes the tool as opinionated, and I would
tend to agree. On the other hand, there is no obligation to use 
all of the available commands. Just pick the ones you find useful and
that don't get in the way of your workflow.

[zb on GitHub](https://github.com/joshuarubin/zb)

# *realize:* Trigger your toolchain via Ctrl-S

The standard go tools - `go build`, `go test`, etc. are quick and
uncomplicated, but as your projects get larger and more complex, you
start wishing for some kind of automation that triggers all builds and
tests each time you save a source file.

`realize` is your friend. 

Activating `go build`, `test`, `run`, `generate`, `fmt`, etc. is just a matter of
flipping some boolean switches in a config file (as opposed to
specifying the complete command line). 

Plus, you can add custom commands for pre- and post-processing, set
paths to ignore, choose to save output, log, or error streams from the
build, and more. 

And besides a colorful shell output (where you can quickly tell
successful builds from failed builds by the color), `realize` also has a
Web UI to monitor all of your build processes in one browser window.

[realize on GitHub](https://github.com/tockins/realize) 


# *binstale:* Are my binaries up to date?

Do you know if the binaries in your `$GOPATH/bin` directory are still up
to date? `go get` happily installs binary after binary, but then they
start collecting dust. And at one point you remember you once installed
that nice tool that helped you doing *xyz* for you, and eventually you
find it in `$GOPATH/bin`, but starting it fails with some
incomprehensible error message. 

You are pretty sure it worked without flaws back then, so maybe the
binary got stale? You decide to try updating the source code.

You run a recursive search in `$GOPATH/bin` to find a repository of
the same name as the binary. Eventually you find the repository and
call `go get` on it. This fixes your binary, and you feel relieved...

...until you realize that there could be dozens, if not hundreds, of 
stale binaries in your `$GOPATH/bin` directory!

Meet `binstale`.

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
auto-update any binaries. But updating is just a matter of copying 
and pasting the repository path to `go get -u` and you're done.

[binstale on GitHub](https://github.com/shurcooL/binstale). 


# Conclusion

These are only a few examples from a steadily growing base of 
useful command line tools written to make a developer's life 
easier.

If you have an idea for a cool tool, don't hesitate to sit down and
write it. But first, be sure to check the public repositories - 
the tool you have in mind might already exist somewhere, thanks to
a thriving Go community.

