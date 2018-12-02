+++
author = ["Andrei Tudor Călin"]
date = "2018-12-08T08:00:00Z"
title = "On the tension between generic code and special cases"
linktitle = "generic code vs special cases"
series = ["Advent 2018"]
+++

The `io.Reader` and `io.Writer` interfaces appear in practically
all Go programs, and represent the fundamental building blocks for
dealing with streams of data. An important feature of Go is that the
abstractions around objects such as sockets, files, or in-memory
buffers are all expressed in terms of these interfaces. When a
Go program speaks to the outside world, it almost always does so
through `io.Reader`s and `io.Writer`s, irrespective of the specific
platform or communication medium it uses. This universality is a
key factor in making code that deals with streams of data composable
and re-usable[^1].

This post examines the design and implementation of `io.Copy`,
a function which connects a `Reader` to a `Writer` in perhaps the
simplest way possible: it transfers data from one to the other.

In the general case[^2], `io.Copy` allocates a buffer, then alternates
reading from the source reader into the buffer with writing from the
buffer to the destination writer. This works well for many cases,
and is certainly correct from a semantic point of view.

That being said, what if for some particular choice of reader and
writer, we could do better? How could we teach `Copy` about it?

Code that uses high level abstractions such as `Reader` and `Writer`
must often answer questions like these, and must deal with this
tension.  In general, different platforms, programming languages,
or even libraries deal with this question in different ways.

Let's examine the case of `io.Copy` in particular, in the hope of
acquiring more general wisdom.

#### One possible try: teaching Copy about specific types

Imagine a `Copy` that looks like this:

```go
package hypotheticalio

import "bytes"

func Copy(dst Writer, src Reader) (int64, error) {
	switch s := src.(type) {
	case *bytes.Buffer:
		n, err := dst.Write(s.Bytes())
		return int64(n), err
	default:
		// generic code path
	}
}
```

Notice how our hypothetical `io` package now imports `bytes` so that it
can use the `Buffer` type in the type switch. This prohibits `bytes`
from ever importing `io`, because Go does not allow circular imports.
Perhaps we do not notice the problem just yet, and we move on.

Time goes by, and we discover even more special cases worth considering:

```go
package hypotheticalio

import (
	"bytes"
	"net"
	"os"
)

func Copy(dst Writer, src Reader) (int, error) {
	switch s := src.(type) {
	case *bytes.Buffer:
		n, err := dst.Write(s.Bytes())
		return int64(n), err
	case *net.TCPConn:
		return platformSpecificThings(dst, s)
	case *os.File:
		return differentPlatformSpecificCode(dst, s)
	default:
		// generic code path
	}
}
```

The code for `Copy` is changing a lot, even though the _meaning_
of the code has not changed at all. Not only that, but `Copy` now
concerns itself with platform-specific bits, and it knows about
operating systems, networking, and so on. It used to be nice and
generic, but it is now a difficult to maintain mess of special cases.

It seems like something has gone wrong. This `Copy` _does_ accomodate
both special cases and generic code, but it pays a terrible price to
do so, and it imposes terrible restrictions upon the rest of the world.

#### Perhaps a better try: decoupling Copy from the world using interfaces

Instead of teaching `Copy` about specific types, the `io` package
introduces two interfaces: `ReaderFrom` and `WriterTo`.

A `ReaderFrom` can be thought of as an object that consumes the data
from a `Reader` into itself. By contrast, a `WriterTo` can be
thought of as an object that pushes the data out of itself into a
`Writer`.

Conceptually, a data transfer from an object to another occurs in both
cases, but the way the transfer is expressed makes all the difference.
`Copy` doesn't need to know anything specific about the types it is
working with anymore. If one of them implements `ReaderFrom` or `WriterTo`,
`Copy` calls that method, and performs no other work. `Copy` now looks
like this:

```go
package io

func Copy(dst Writer, src Reader) (int64, error) {
	if wt, ok := src.(WriterTo); ok {
		return wt.WriteTo(dst)
	}
	if rt, ok := dst.(ReaderFrom); ok {
		return rt.ReadFrom(src)
	}
	// generic code path
}
```

Something interesting has happened: compared to the hypothetical
scenario from before, `Copy` now has very little reason to ever
change. It is completely generic once again. Not only that, but it
can delegate to pieces of code which _do_ have more specific knowledge
of types just as well as it did before.

Nothing comes for free, though, and this loose coupling has its
own cost.  Capabilities are no longer known statically to `Copy`
through specific types, but must be discovered dynamically at runtime,
using type assertions.

Interestingly, instead of manifesting itself through messy code,
high maintenance costs and prohibitive import restrictions, the
tension between generic code and special cases now manifests itself
through the loss of compile time information. For package such as `io`,
which is imported by the whole world, this certainly seems like a
trade worth making.

Callers can specialize `io.Copy` by themselves, without changing the
function itself. All they need to do is implement `io.ReaderFrom` or
`io.WriterTo`. The standard library does this in many places. For example:

* `*bytes.Buffer` has both a [WriteTo](https://golang.org/pkg/bytes/#Buffer.WriteTo),
  which drains the buffer into an `io.Writer`, an a
  [ReadFrom](https://golang.org/pkg/bytes/#Buffer.ReadFrom) which fills
  the buffer from an `io.Reader`

* `*net.TCPConn` has a [ReadFrom](https://golang.org/pkg/net/#TCPConn.ReadFrom),
  which may use `sendfile(2)` (or a similar interface) on most platforms

* the `net/http` implementation of `ResponseWriter` has a
  [ReadFrom](https://golang.org/src/net/http/server.go#L566) which may make use
  of the aforementioned `sendfile(2)` special case

It is important to note that these are all optimizations which should not
affect the semantics of programs in any way. As such, the worst thing that
can happen to clients of package `io` is that a specific optimization
might not kick in. Let's examine one such case. Consider the following
wrapper type:

```go
type CountingWriter struct {
	W io.Writer
	N int64
}

func (cw *CountingWriter) Write(b []byte) (int, error) {
	n, err := cw.W.Write(b)
	cw.N += int64(n)
	return n, err
}
```

When used as an `io.Writer`, `*CountingWriter` hides the properties
of the underlying `Writer` from callers. As such, code that relies on
detecting capabilities at runtime, such as `io.Copy`, will only see an
`io.Writer` when it looks at a `*CountingWriter`.

Callers that nevertheless want a specific feature of the underlying
`Writer` to be used in such cases must accomodate for it themselves,
by discovering the interesting capabilities and using types with more
specific wrapper methods. This can be prohibitively difficult in
certain cases[^3].

Furthermore, note how `io.ReaderFrom` and `io.WriterTo` do not appear
in the _signature_ of `io.Copy`. They appear in the _documentation_
instead: a far weaker contract.

#### Closing thoughts

One way or another, the fundamental tension between generic code
and special cases appears in any code that deals with abstractions.
To accomodate both, the nature of Go interfaces enables one specific
kind of loose coupling between components, but this method is not
without its subtle costs.  Even so, the end result can remain elegant
and easy to maintain.

[^1]: compare Go to platforms which exhibit [red-blue](http://journal.stuffwithstuff.com/2015/02/01/what-color-is-your-function/) issues

[^2]: see the source code [here](https://github.com/golang/go/blob/112f28defcbd8f48de83f4502093ac97149b4da6/src/io/io.go#L401-L423)

[^3]: observe the combinatorical explosion which [this library](https://github.com/felixge/httpsnoop) solves
