+++
author = ["Peter Bourgon"]
title = "Managing goroutine lifecycles with run.Group"
linktitle = "run group"
date = 2017-12-24T00:34:34Z
series = ["Advent 2017"]
+++

# Motivation

My programs tend to have the same structure:
 they're built as a set of inter-dependent, concurrent components,
 each responsible for a conceptually distinct bit of behavior.
All of these components tend to be modeled in the same way, more or less:
 whether implemented as structs with methods or free functions,
 they're all things that are _running_: doing stuff, responding to events, changing state, talking to other things, and so on.
And when I write programs, in the style of a large func main with explicit dependencies,
 I generally construct all of the dependencies from the leaves of the dependency tree,
 gradually working my way up to the higher-order components,
 and then eventually `go` the specific things that I want to run.

In this example, I have 
 a state machine, 
 an HTTP server serving an API, 
 some stream processor feeding input to the state machine, 
 and a ctrl-C signal handler.

```go
// TODO
```

We have this setup phase in our func main, where we set up a context.
We set up all of the common dependencies, like the logger.
And then, we make all of the components, in the order dictated by their dependency relationships.
Our state machine has an explicit Run method, which we `go` to get it started.
Our HTTP API needs to have its handler served in its own `go` routine.
Our stream processor is modeled as a function, taking the input stream and state machine as dependencies, which we also `go` in the background.
And our ctrl-C handler also needs to be running, waiting for its signal.

I think this is the best way to structure object graphs and dependencies in Go programs, 
 and [I've written at length about it before](https://peter.bourgon.org/go-best-practices-2016/#program-design).
But there's some trickiness in the details here.
We know that we must [never start a goroutine without knowing how it will stop](https://dave.cheney.net/2016/12/22/never-start-a-goroutine-without-knowing-how-it-will-stop).
But how do we actually do this, in a way that's both
 intuitive enough for new maintainers to easily grok and extend, and
 flexible enough to handle the nontrivial use cases we have here?

To me, the complication hinges not on how to start the goroutines, 
 or handle communication between them,
 but on how to deterministically tear them down.
Returning to our example, let's consider how each of the components might be stopped.

The state machine is clear: since it takes a context.Context, presumably it will return when the context is canceled.

```go
// TODO
```

But the HTTP server presents a problem: as written, there's no way to interrupt it.
So we need to change it slightly.
It turns out http.ListenAndServe is just a small helper function, which combines two things:
 first binding the listener, and then attaching and running the server.
If we do those two steps explicitly ourselves, we get access to the net.Listener,
 which has a Close method that, when invoked, will trigger the server to return.
Or, better still: we can leverage the graceful shutdown functionality added in 1.8.

```go
// TODO
```

In this demonstrative example, the stream processor will return when its stream is exhausted.
But as written, there's no way to trigger an e.g. io.EOF on a plain io.Reader.
Instead, we'll need to wrap it into an io.ReadCloser, and provide a way to close the stream pre-emptively.
(The concrete type that implements io.Reader, for example a net.Conn, may also have a Close method that could work.)

```go
// TODO
```

Finally, the ctrl-C handler also has no way to be interrupted as written.
But since it's our own code, we're presumably free to modify it to add an interrupt mechanism.
I like using a cancel chan for this kind of basic stuff: less surface area than context.Context.

```go
// TODO
```

Look at all the different ways we have to terminate goroutines.
I think anything that manages goroutine lifecycles needs to accommodate _at least_ all of these use cases.
And I think the only commonality between the cancelation mechanisms is that they're blocks of Go code.
If we embrance that constraint, and try to design a goroutine lifecycle manager around it, what sort of API falls out?

# run.Group

I think the answer is [run.Group](https://godoc.org/github.com/oklog/run#Group).

```
// TODO GoDoc
```
