+++
author = ["Damian Gryski"]
date = "2015-12-06T08:00:00+00:00"
title = "Go Fuzz"
series = ["Advent 2015"]
+++

In April of this year, Dmitry Vyukov released the first version of
[go-fuzz](https://github.com/dvyukov/go-fuzz), a coverage-guided fuzz testing
tool based on ideas from [afl](http://lcamtuf.coredump.cx/afl/).  With very
little fanfare, he unleashed it on the Go standard library and started filing
huge numbers of crashers and other bugs found via automated, randomized
testing.

Fuzzing is testing code by feeding it random data.  It dates back to to
[Professor Barton Miller's work in late
1980s](http://pages.cs.wisc.edu/~bart/fuzz/).  In the 2000s, it was picked up
by the security community, because it turned out that many of these crashes
could be turned into code exploits.  Coverage-guided fuzzing, like afl and
go-fuzz do, use information gathered at run-time about which code paths were
executed to determine if a random input was "interesting" and should be used as
the seed for further random tests.

Any program dealing with user input should use fuzz testing to make sure it can
deal with unexpected inputs gracefully.  It's especially suited to packages
like file formats, serialization routines, and compression algorithms.

But even with the great success go-fuzz was having in finding issues in the Go
standard library (probably some of the best tested and well reviewed Go code in
existence), it seemed few other people were using it in their own projects.

At the beginning of July, Dmitry presented [go-fuzz at
Gophercon](https://www.youtube.com/watch?v=a9xrxRsIbSU).

Leading up to the release Go 1.5, I launched a
[fuzz-a-thon](https://groups.google.com/forum/#!topic/Golang-Nuts/4PmyYvcnpIs).
The idea was simple: Dmitry is doing his best to make the Go standard library
robust.  We, the community, need to do the same for our own packages.  I
[tweeted every day reasons to use
go-fuzz](https://twitter.com/search?f=tweets&vertical=default&q=%23golangfuzz).

At the beginning of August, Filippo Valsorda posted an [excellent account of
using go-fuzz at
CloudFlare](https://blog.cloudflare.com/dns-parser-meet-go-fuzzer/).  Filippo's
post (and [his talk at GothamGo](https://www.youtube.com/watch?v=QEhPaj3vvPA)
detailed other ways to use go-fuzz for randomized tests, not *only* to find
inputs that make your program crash.

A few weeks later, I published my own simple walk-through of [fuzzing and
patching a simple file format
library](https://medium.com/@dgryski/go-fuzz-github-com-arolek-ase-3c74d5a3150c).

People are beginning to use it.  More and more packages include fuzz tests.
The [go-fuzz trophy case](https://github.com/dvyukov/go-fuzz#trophies)
contains almost 400 bugs.

Please fuzz your own code.
