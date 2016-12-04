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
I think are noteworthy. 

# *interfacer:* Should I rather use an interface here?

To be honest, I am not sure whether "not mainstream" really applies to
the first tool presented here: [`interfacer`](https://github.com/mvdan/interfacer/).
After all, it already has almost 500 stars on GitHub! (Disclaimer: I 
know that GitHub stars server two very different purposes--liking and
bookmarking--so they aren't a reliable reference of popularity, but 
still, 500 isn't too bad.)

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

Imagine you just finished writing a larger library that contains a `Doer` 
interface and a `DoIt` struct that implements `Doer`. A couple hours and
few thousand lines later (yes, you felt productive this day) you defined 
function `DoSomething` that expects a `DoIt` struct and calls its `Do`
method.

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

I do not list all of the available commands here; otherwise I would 
just end up rephrasing the README file here. If you got curious, 
just head over to the [zb repository](https://github.com/joshuarubin/zb)
and have a look.


# *devop:* Save your file and trigger anything you need.

# *binstale:* Are my binaries up to date?
