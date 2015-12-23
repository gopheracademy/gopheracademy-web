+++
author = ["Samuel Lampa"]
date = "2015-12-22T00:00:00-00:00"
title = "Composable Pipelines Improved"
series = ["Advent 2015"]
+++

I wrote a [post here on GopherAcademy](https://blog.gopheracademy.com/composable-pipelines-pattern) earlier this year,
about an idea for a "framework-less" pattern for [Flow-Based Programming](http://www.jpaulmorrison.com/fbp/) style programs in Go,
or let's just call it "composable concurrent pipelines". During the year, I have experimented more, and added some minor modifications to this pattern, which
I describe below.

Please note that the code examples below are kept short and thus incomplete, for
readability. For a full working example of the presented pattern in action,
please see [this gist](https://gist.github.com/samuell/07ee336c9fb39c45b89b)!

## The old way

The basic idea in that earlier post was to expand on the [generator pattern described in a slide by Rob Pike](https://talks.golang.org/2012/concurrency.slide#25) by storing the concurrent processes in structs rather than just functions. This allows representing in- and out-ports as struct fields that can be used to connect in- and out-ports of multiple processes in a more fluent way, which the post described.

I have realized some further simplifications of this pattern though. One unnecessary thing suggested in that post was to use a method for each out-port, that would ensure that a channel is created for that out-port before returning it.

The pattern proposed that a process would be defined like so:

```go
type AProcess struct {
	In  chan string
	Out chan string
}

func (p *AProcess) OutChan() chan string {
	p.Out = make(chan string, 16)
	return p.Out
}

func (p *AProcess) Init() {
	go func() {
		defer close(p.Out)
		for line := range p.In {
			// Do something with `line` here ...
			p.Out <- line
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
// Code to drive this pipeline left out here for brevity
```

## A better way

What I realized though is that if we use a "factory function" to create new tasks and pre-populate the channel fields, we
just need to assign one such field to a field of another processes to connect two processes.

So, a task would be defined like so, including its factory function:

```go
type AProcess struct {
	In  chan string
	Out chan string
}

func NewProcess() *AProcess {
	return &AProcess{
		In:  make(chan string),
		Out: make(chan string),
	}
}

func (p *AProcess) Init() {
	go func() {
		defer close(p.Out)
		for line := range p.In {
			// Do something with `line` ...
			p.Out <- line
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
// Again, Code to drive this pipeline left out here for brevity
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
processes, we were looping over the output of the out-port of the last component, right in the programs main-method.

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
(i.e. use the `go` keyword) like my pattern did inside the `Init()` methods, he had `Run()` methods without any `go` statements in them and instead called the `go` keyword outside of the processes.

So, if we replaced the `Init()` method in the code examples above with the following `Run()` method:

```go
func (p *AProcess) Run() {
	defer close(p.Out)
	for line := range p.In {
		// Do something with `line` ...
		p.Out <- line
	}
}
```

... then we would execute these `Run()` methods like so (adapting it to our example code here):

```go
proc1 := NewProcess()
proc2 := NewProcess()
go proc1.Run()
go proc2.Run()
// Again, Code to drive this pipeline left out here for brevity
```

This suggests an elegant solution to the problem of driving a chain of go-routines from the main thread: Skip the go
keyword for the last process in the chain! So, for example like so:

```go
proc1 := NewProcess()
proc2 := NewProcess()
go proc1.Run() // Execute in separate go-routine
proc2.Run() // Execute the last process in the main thread
```

Now, this can be packaged into a convenient Pipeline component:

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

... that we can use like this (assuming we have already initiated and connected `proc1` and `proc2`):

```go
// Add processes to pipeline and run
pipeline := NewPipeline()
pipeline.AddProcesses(proc1, proc2)
pipeline.Run()
```

So a full code example of using the refined "framework-less flow-based-programming inspired" pattern, could look like so, (leaving out the process implementations for brevity):

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

Just note that if the last process is sending some output on a channel as well we need another process in the end that just
receives these outputs as inputs, doing nothing with it.

We could for example implement a special "sink" process for that:

```go
type Sink struct {
	In chan string
}

func NewSink() (s *Sink) {
	return &Sink{
		In: make(chan string),
	}
}

func (proc *Sink) Run() {
	for _ = range proc.In {
		// Do nothing ...
	}
}
```

## SciPipe

This pattern is now serving as the basis for a scientific workflow library that I'm experimenting with, which I call [SciPipe](http://scipipe.org).

Very briefly, the implementation of SciPipe consists of the pattern above with the addition of a specialized process type (`ShellProcess`), that can take a shell
command pattern and generate a component out of that with one in-port per input file, and out-port per output file,
and that will fire off tasks executing a formatted shell command for every full set of inputs received on the in-ports (read more
in the [SciPipe README](https://github.com/samuell/scipipe/blob/master/README.md)).

SciPipe is still in prototype phase but there are a fair number of fully working toy examples in the [examples folder](https://github.com/samuell/scipipe/tree/master/examples), so the basic idea seems to be working.

Feedback and suggestions for improvement of the idea and the code are much welcome!

Twitter: [@smllmp](http://twitter.com/smllmp)
