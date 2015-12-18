+++
author = ["Samuel Lampa"]
date = "2015-12-18T00:00:00-08:00"
title = "Composable Pipelines Improved"
series = ["Advent 2015"]
+++

I wrote a [post here on GopherAcademy](https://blog.gopheracademy.com/composable-pipelines-pattern) earlier this year,
about an idea for a "framework-less" pattern for [Flow-Based Programming](http://www.jpaulmorrison.com/fbp/) style programs in Go,
or let's just call it "composable concurrent pipelines". During the year, I have experimented more, and added some minor modifications to this pattern, which
I describe below.

## The old way

The basic idea described in that earlier post was the following: Rather to do [like in the generator "pattern"](https://talks.golang.org/2012/concurrency.slide#25),
where concurrent processes were packed into functions that returns the output channels on which their lazy-evaluated output will be provided,
we instead store each process into structs, with each in-port and out-port (I use the term port from Flow-based programming here, though technically in Go, it
is in this case just a channel stored in a named struct field) as separate struct fields, and the main code of the processes in a
go-routine, to make it concurrent.

The basic approach has the benefits that the code for routing out-ports to in-ports becomes a lot clearer
than with the generator pattern, since each port is handled separately, even on separate lines of code.

An unnecessary feat of that provided pattern though, was using methods for each out-ports, that would ensure that a channel
is created for that out-port, and then returning that channel. The pattern proposed that a process would be defined like so:

```go
type AProcess struct {
	In  string
	Out string
}

func (p *AProcess) OutChan() chan string {
	p.Out = make(chan []byte, 16)
	return p.Out
}

func (p *AProcess) Init() {
	go func() {
		for line := range p.In {
			// Do something with `line` ...
		}
	}()
}
```

Connecting such processes together would go something like:

```go
proc1 := &AProcess{}
proc2 := &AProcess{}
// Connect
proc2.In = proc1.OutChan()
```

## A better way

What I realized, is that if we use a function to create new tasks, that pre-populates all the channel fields, we can
just assign those fields to each other.

So, a task would be defined like so, including its factory function:

```go
type AProcess struct {
	In  string
	Out string
}

func NewProcess(*AProcess) {
	return &AProcess{
		In:  make(chan string),
		Out: make(chan string),
	}
}

func (p *AProcess) Init() {
	go func() {
		for line := range p.In {
			// Do something with line ...
		}
	}()
}
```

Then, to connect two such processes together, we would go:

```go
proc1 := NewProcess()
proc2 := NewProcess()
// Connect
proc2.In = proc1.Out
```
... and we could even connect the other way around, since both the In- and Out-port fields are initiated with channels,
so it doesn't matter which of these channels we use, as long as it is the same channel used on the corresponding out-
and in-port:

```go
// <snip>
// Connect the other way around, setting the out-port to the channel in the in-port
proc1.Out = proc2.In
```

## Let's create a pipeline component too

There was one other ugly part of that previously blogged example. In order to drive the execution of a set of connected
processes, we were just looping over the output of the out-port of the last component, right in the programs main-method.

That is, the following part in the previous post:

```go
for i := range printer.DrivingBeltChan() {
	linesProcessed += i
}
```

This was due to the fact that (as far as I know) the execution of separate go-routines can not really be started until they get
a signal from the main go-routine, over a channel, for example.

I couldn't come up with a better suggestion for how to drive such a chain of processes, until [Egon Elbre](https://twitter.com/egonelbre)
[elaborated on some tips](https://groups.google.com/forum/#!msg/golang-nuts/vgj_d-MjUHA/T9sE64Yrcq0J) on how to enhance the pattern.

While I did not use Egon's whole suggestion since it included a fair bit of reflection and departed from my idea
of a framework-less pattern, his code examples did a nice trick; Rather than letting the processes fire up go-routines,
like my pattern did inside the `Init()` methods, he had plain `Run()` methods without any go-routines in them, and instead
fired off as go-routines (with the `go` keyword) outside of the processes.

So, if we replaced the `Init()` method in the code examples above with the following `Run()` method:

```go
func (p *AProcess) Run() {
	for line := range p.In {
		// Do something with line ...
	}
}
```

... then we would execute these `Run()` methods like so (adapting it to our example code here):

```go
proc1 := NewProcess()
proc2 := NewProcess()
go proc1.Run()
go proc2.Run()
```

This suggests an elegant solution of the problem of driving a chain of go-routines from the main thread: Just skip the go
keyword for the last process in the chain! So, for example like so:

```go
proc1 := NewProcess()
proc2 := NewProcess()
go proc1.Run() // Execute in separate go-routine
proc2.Run() // Execute the last process in the main thread
```

Now, this can of course be packaged into a convenient Pipeline component:

```go
type Pipeline struct {
	processes []interface{} // TODO: We could use a base-process type here instead of interface{} ...
}

func NewPipeline() *Pipeline {
	return &Pipeline{}
}

func (pl *Pipeline) AddProcesses(procs ...interface{}) {
	for _, proc := range procs {
		pl.AddProcess(proc)
	}
}

func (pl *Pipeline) Run() {
	for i, proc := range pl.processes {
		if i < len(pl.processes)-1 {
			go proc.Run() // Start separate go-routines for all but the last process
		} else {
			proc.Run() // Run the last process in the main go-routine
		}
	}
}
```

... that we now can use like this (assuming we have already initiated and connected together `proc1` and `proc2`):

```go
// Add processes to pipeline and run
pipeline := NewPipeline()
pipeline.AddProcesses(proc1, proc2)
pipeline.Run()
```

So, a full code example of using the refined "framework-less flow-based-programming inspired" pattern (apart from
the component implementations), could look like so:

```go
// Init processes
proc1 := NewProcess()
proc2 := NewProcess()

// Connect processes
proc2.In = proc1.Out

// Add processes to pipeline and run
pipeline := NewPipeline()
pipeline.AddProcesses(proc1, proc2)
pipeline.Run()
```

Just note that if the last process is sending its output on a channel too, we need another process in the end that just
receives inputs and does nothing with it.

We could even implement a special "sink" process for that:

```go
type Sink struct {
	In chan string
}

func NewSink() (s *Sink) {
	return &Sink{}
}

func (proc *Sink) Run() {
	for _ := range proc.In {
		// Do nothing ...
	}
}
```

## SciPipe

This pattern is now serving as the basis for a scientific workflow library that I'm experimenting with, which I call [SciPipe](http://scipipe.org).

Very briefly, the implementation of SciPipe consists of the pattern above with the addition of a specialized process type (`ShellProcess`), that can take a shell
command pattern and generate a component out of that with one in-port per input file, and out-port per output file,
and that will fire off tasks executing the shell command for every full set of inputs received on the in-ports (read more
in the [SciPipe README](https://github.com/samuell/scipipe/blob/master/README.md)).

SciPipe is still in the prototype phase, but there are a fair number of toy examples in the [examples folder](https://github.com/samuell/scipipe/tree/master/examples), that
are fully working, so the basic idea seems to be working. 

Feedback and suggestions for improvement of the idea and the code, are very much welcome, as we hope to
make SciPipe a usable tool in the near future.

### A few words on the ideas behind SciPipe

I will refer to the [SciPipe website](http://scipipe.org) or [README](https://github.com/samuell/scipipe/blob/master/README.md) for more background information, but just a few words about the thinking behind creating a scientific
workflow system in Go...

In short idea of using Go was based on the realization that Go's concurrency primitives provide an excellent basis for an
implicit task scheduler, where tasks are "scheduled" or run, as soon as data arrives on the in-ports of any of the
processes. Or, a better way to put this I guess, is that Go's internal scheduler is taking care of the scheduling, so that
we don't need to implement a dedicated task scheduler, just to run tasks concurrently. We just have to take care of the data
dependencies between processes, and wire up a network of Go channel according to these dependencies and feed data over that
network of channels, and the scheduling is simply inherent in this concurrent data flow network.

Also, using a full-fledged programming language to write workflows in, enables re-using the great tooling infrastructure
around the language, such as syntax highlighting and auto-completion support for many editors, etc.

Furthermore, the fact that components are pure Go, enables combining processes that call external programs via shell commands,
with processes implemented in pure Go when possible. Since Go is a decently performant language with excellent concurrency
support, the drive would be to move over as many processes as possible into Go, to lessen the dependencies and possibly
increase performance, of the workflow.

Finally, I quite like the idea of being able to compile at least the workflow part of a scientific workflow into a single
executable, to upload for execution basically anywhere (and if and when all the external programs is uses have been replaced
with Go counterparts, this will go for the full workflow)! :)
