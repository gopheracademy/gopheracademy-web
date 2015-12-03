+++
author = ["Derek Parker"]
date = "2015-12-03T08:00:00+00:00"
title = "Debugging Go programs with Delve"
series = ["Advent 2015"]
+++

Tracking down bugs in your code can be a very frustrating experience. This is even more true of highly parallel code. Having a good debugger at your disposal can make all the difference when it comes to tracking down a difficult, or hard to reproduce bug in your code. Throughout this post I will discuss [Delve](https://www.github.com/derekparker/delve), which is a debugger specifically built for Go.

[Delve](https://www.github.com/derekparker/delve) aims to solve the various issues felt by developers when debugging their Go code with traditional tools such as [GDB](https://www.gnu.org/software/gdb/). For more information on why existing tools fall short see the introduction paragraph on the [Go gdb documentation](https://golang.org/doc/gdb) and feel free to check out my [Gophercon 2015 talk](htt://www.youtube.com/watch?v=InG72scKPd4) where I discuss some of the technical details.

For the remainder of this post we will introduce [Delve](https://www.github.com/derekparker/delve) a bit more and walk through some usage examples.

## Getting set up

*Delve is only available on Linux and OSX, with Windows support [coming soon](https://github.com/derekparker/delve/pull/276).*

If you haven't already installed [Delve](https://www.github.com/derekparker/delve), check out the [installation instructions](https://github.com/derekparker/delve/wiki/Building) to get started. Note that if you're on OSX you must follow the instructions to codesign the binary. Once you're finished you will have everything you need to begin debugging Go programs.

## Debugging a program

Let's be honest, if you're reaching for a debugger, things already are not going your way. Your program is not working and you have no idea why. With that in mind, the tools you use should not get in your way. Ease of use is a major goal, and can be demonstrated by explaining how to start a debug session.

##### Build and debug:

	$ dlv debug

Run that command in the same directory you would run `go build` from and it will compile your program, passing along flags to make the resulting binary easier to debug, and then start your program, attach the debugger to it, and land you at a prompt to begin inspecting your program.

##### Build test binary and debug:

	$ dlv test

If you do not have a `main` function, or want to debug your program in the context of your test suite, use the above command. Again, this will build a test binary, using the correct flags for an optimal debugging experience, and land you at a prompt where you can begin issuing commands.

##### Attach to running process:

	$ dlv attach <pid>

Attach to a running process and begin debugging. This command will immediately stop the process and begin a debug session. Keep in mind, however, you may run into issues attempting to debug a binary compiled with certain optimizations.

##### Trace instead of debug:

	$ dlv trace [regexp]

Compile and start program, setting tracepoints at any function that matches `[regexp]`. This will not begin a full debug session, but will print information whenever a tracepoint is hit.

##### Additional commands

These will likely be your most used commands, however Delve has the following subcommands as well:

* `$ dlv exec ./path/to/binary` - Run and attach to an existing binary.
* `$ dlv connect` - connect to headless debug server.

## What now?

You should now see the `(dlv)` prompt and are now ready to begin inspecting your program!

Let's consider a small program such as:

```go
package main

import (
	"fmt"
	"sync"
)

func dostuff(wg *sync.WaitGroup, i int) {
	fmt.Printf("goroutine id %d\n", i)
	fmt.Printf("goroutine id %d\n", i)
	wg.Done()
}

func main() {
	var wg sync.WaitGroup
	workers := 10

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go dostuff(&wg, i)
	}
	wg.Wait()
}
```

Let's begin a debug session with `$ dlv debug` and start by setting a breakpoint at `main`:

```
(dlv) break main.main
Breakpoint 1 set at 0x22c7 for main.main ./test.go:15
```

The output tells us the breakpoint ID, the address the breakpoint was set at, the function name, and the file:line.

We can continue to that breakpoint using the `continue` command. Once you stop at that breakpoint, explore your program by typing `next` and then pressing the `Enter` key (Delve will repeat the last command given when it receives an empty one). The `next` command will step the program forward by one source code line. Now, let's try looking around: use the `print` command to print the value of `workers`, like so:

```
(dlv) print workers
10
```

Delve can also evaluate certain expressions, so the following also works:

```
(dlv) print workers < 100
true
```

Let's set another breakpoint at our `dostuff` function:

```
(dlv) break dostuff
Breakpoint 2 set at 0x205f for main.dostuff ./test.go:9
```

Again, let's `continue` which should land us at the breakpoint we just set:

```
(dlv) continue
> main.dostuff() ./test.go:9 (hits goroutine(6):1 total:1)

     4:         "fmt"
     5:         "sync"
     6: )
     7:
     8: func dostuff(wg *sync.WaitGroup, i int) {
=>   9:         fmt.Printf("goroutine id %d\n", i)
    10:         fmt.Printf("goroutine id %d\n", i)
    11:         wg.Done()
    12: }
    13:
    14: func main() {
```

Let's print out the value of `i` using the following command: `(dlv) print i`. Now, let's use the `next` command and then print out the value of `i` again. You'll notice it's the same, and this is no coincidence.

We have created 10 goroutines executing this function and yet we land on the same goroutine. This is because Delve, being a Go specific debugger, has knowledge of Go specific runtime features such as Goroutines. When you execute the `next` command, Delve will make sure to put you on the next source line in the context of _that_ goroutine. This prevents the frustrating "thrashing" effect from other tools, where you may end up on a completely different goroutine after using a command like `next`.

## Wrapping up

This has only been a _very_ introductory tour into what Delve can do, and we've only just scratched the surface. Feel free to use Delve on your own programs, and check out the `help` command for all the ways you can inspect your program.

Please note that Delve is pre-1.0; there are plans to improve existing functionality as well as add new features. 

## How to contribute

The project is open source, so feel free to check it out. We are planning to release a 1.0 version very soon, and can use all the feedback and contributions we can get! Check out the [repo](https://github.com/derekparker/delve)  and don't hesitate to file an issue or submit a patch!

If you're interested in hacking on Delve, but are unsure of where to start, or how the internals of a debugger work feel free to ask for pointers, guidence and material to help you make your contribution.
