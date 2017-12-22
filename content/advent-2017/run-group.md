+++
author = ["Peter Bourgon"]
title = "Managing goroutine lifecycles with run.Group"
linktitle = "Managing goroutine lifecycles with run.Group"
date = 2017-12-24T00:34:34Z
series = ["Advent 2017"]
+++

I stumbled over this idiom for managing goroutine lifecycles when I was writing [OK Log](https://github.com/oklog/oklog/).
Since then I've found uses for it in nearly every program I've written.
I thought it'd be nice to share it.

## Motivation

My programs tend to have the same structure:
 they're built as a set of inter-dependent, concurrent components,
 each responsible for a distinct bit of behavior.
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
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

sm := newStateMachine()
go sm.Run(ctx)

api := newAPI(sm)
go http.ListenAndServe(":8080", api)

r := getStreamReader()
go processStream(r, sm)

signalHandler() // maybe we wait for this one to return
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
sm := newStateMachine()
go sm.Run(ctx) // stopped via cancel()
```

But the HTTP server presents a problem: as written, there's no way to interrupt it.
So we need to change it slightly.
It turns out http.ListenAndServe is just a small helper function, which combines two things:
 first binding the listener, and then attaching and running the server.
If we do those two steps explicitly ourselves, we get access to the net.Listener,
 which has a Close method that, when invoked, will trigger the server to return.
Or, better still: we can leverage the graceful shutdown functionality added to http.Server in 1.8.

```go
api := newAPI(sm)
server := http.Server{Handler: api}
ln, _ := net.Listen("tcp", ":8080")
server.Serve(ln) // shutdown via server.Shutdown()
```

In this demonstrative example, we'll say that the stream processor returns when its stream io.Reader is exhausted.
But as written, there's no way to trigger an e.g. io.EOF on a plain io.Reader.
Instead, we'd need to wrap it into an io.ReadCloser, and provide a way to close the stream pre-emptively.
Or, perhaps better, the concrete type that implements io.Reader, for example a net.Conn, may also have a Close method that could work.

```go
// r := getStreamReader()
rc := getStreamReadCloser()
go streamProcessor(rc, sm) // stopped via rc.Close()
```

Finally, the ctrl-C handler also has no way to be interrupted as written.
But since it's our own code, we're presumably free to modify it to add an interrupt mechanism.
I like using a cancel chan for this kind of basic stuff: less surface area than context.Context.

```go
stop := make(chan struct{})
signalHandler(stop) // returns via close(stop) (or ctrl-C)
```

Look at all the different ways we have to terminate goroutines.
I think the only commonality between them is that they're expressions, or blocks of Go code.
And I think anything that manages goroutine lifecycles needs to accommodate this heterogeneity.
If we embrance that constraint, and try to design an API around it, what falls out?

## run.Group

My guess at an answer is [package run](https://godoc.org/github.com/oklog/run#Group),
 and the [run.Group](https://godoc.org/github.com/oklog/run).
From the package documentation:

> Package run implements an actor-runner with deterministic teardown. It is 
> somewhat similar to [package errgroup](https://godoc.org/golang.org/x/sync/errgroup), 
> except it does not require actor goroutines to understand context semantics.
> This makes it suitable for use in more circumstances; for example, goroutines
> which are handling connections from net.Listeners, or scanning input from a
> closable io.Reader.

With package run and the run.Group, we model each running goroutine as a pair of
functions, defined inline. The first function, called the execute function, is
launched as a new goroutine. The second function, called the interrupt function,
must interrupt the execute function and cause it to return.

Here's [the documentation](https://godoc.org/github.com/oklog/run#Group):

```
func (g *Group) Add(execute func() error, interrupt func(error))
    Add an actor (function) to the group. Each actor must be pre-emptable by
    an interrupt function. That is, if interrupt is invoked, execute should
    return. Also, it must be safe to call interrupt even after execute has
    returned.

    The first actor (function) to return interrupts all running actors. The
    error is passed to the interrupt functions, and is returned by Run.

func (g *Group) Run() error
    Run all actors (functions) concurrently. When the first actor returns,
    all others are interrupted. Run only returns when all actors have
    exited. Run returns the error returned by the first exiting actor.
```

And here's how it looks when we apply it to our example.

```go
var g run.Group // the zero value is useful

sm := newStateMachine()
g.Add(func() error { return sm.Run(ctx) }, func(error) { cancel() })

api := newAPI(sm)
server := http.Server{Handler: api}
ln, _ := net.Listen("tcp", ":8080")
g.Add(func() error { return server.Serve(ln), func(error) { server.Stop(ctx)} })

rc := getStreamReadCloser()
g.Add(func() error { return streamProcessor(rc, sm) }, func(error) { rc.Close() })

stop := make(chan struct{})
g.Add(func() error { return signalHandler(stop) }, func(error) { close(stop) })

log.Print(g.Run())
```

g.Run blocks until all the actors return. In the normal case, that'll be when
someone hits ctrl-C, triggering the signal handler. If something breaks, say the
stream processor, its error will be propegated through. In all cases, the first
returned error triggers the interrupt function for all actors. And in this way,
we can reliably and coherently ensure that every goroutine that's Added to the
group is stopped, when Run returns.

I designed run.Group to help orchestrate goroutines in func main, but I've found
several other uses since then. For example, it makes a great alternative to a
[sync.WaitGroup](https://golang.org/pkg/sync#WaitGroup) if you'd otherwise have
to construct a bunch of scaffolding. Maybe you'll find some uses, too.
