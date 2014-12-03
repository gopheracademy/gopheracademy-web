+++
author = ["Derek Parker"]
date = "2014-12-03T08:00:00+00:00"
title = "Delve: Go debugger"
series = ["Advent 2014"]
+++

## Delve

[Delve](https://github.com/derekparker/delve) is a Go debugger. Currently the project is in beta, with most of the functionality implemented, and various improvements and platform support on the way.

### Why

I started work on Delve sometime shortly after Gophercon 2014. Delve began as a fun and interesting project to hack on, and has since become a useful tool with a lot of potential. Delve was created to address issues with debugging Go programs with GDB. From the official docs on [using GDB with Go](https://golang.org/doc/gdb):

 > GDB does not understand Go programs well. The stack management, threading, and runtime contain aspects that differ enough from the execution model GDB expects that they can confuse the debugger, even when the program is compiled with gccgo. As a consequence, although GDB can be useful in some situations, it is not a reliable debugger for Go programs, particularly heavily concurrent ones. Moreover, it is not a priority for the Go project to address these issues, which are difficult. In short, the instructions below should be taken only as a guide to how to use GDB when it works, not as a guarantee of success.
>
> In time, a more Go-centric debugging architecture may be required.

Delve exists to solve that problem, and provide a powerful tool for debugging Go programs.

### How Delve works

The current implementation of Delve is very Linux specific, relying heavily on the `Ptrace` family of syscalls, along with the proc filesystem. Extended platform support is the next major goal, and one of my primary focuses at the moment. There are some easy tasks towards satisfying that goal, such as removing the reliance on the proc filesystem, preferring instead to analyize internal data structures maintained by the Go runtime for thread information. Along with the easy tasks however come more difficult ones, such as translating some Ptrace syscalls into darwin/mach specific syscalls due to limited Ptrace support on OS X.

Delve works by utilizing the Go symbol table and Dwarf debug infomation encoded into various sections of a Go binary. That information, along with various syscalls for controlling execution of another process allows Delve to provide as much insight into your program as possible. The entries in the Dwarf debug sections enable Delve to calculate information about the stack, variable locations, and more. With this information and the help of various syscalls, Delve is able to print the value of variables, print thread and goroutine information, step over source lines, single step instructions, set breakpoints, and provide you with as much control as possible over your program. The ultimate goal is to provide a reliable debugging tool that Gophers everywhere can use to track down the nastiest bugs we may encounter in our software.

One major step in that direction is the proper handling of the runtime scheduler during a debugging session.

### Handling the runtime scheduler

One of the aspects of every Go program that can be confusing for existing debuggers is the [runtime scheduler](https://docs.google.com/document/d/1TTj4T2JO42uD5ID9e89oa0sLKhJYD0Y_kqxDv3I3XMw/edit). The scheduler manages and coordinates threads and goroutine execution. A traditional debugger such as GDB has no knowledge of the scheduler, which means it cannot handle events like goroutine context switching very well.

Delve was built with the scheduler in mind, since it is such an integral part of any Go program. There are many cases where the scheduler makes debugging Go programs an interesting task. For example, when your program enters a blocking syscall, or even reads from a channel, the scheduler is involved. When that happens, it is very possible for a goroutine to switch thread contexts, or at least require coordination with the scheduler thread. Without careful handling of the Go execution model, a traditional debugger could very easily hang in an unresponsive state waiting on a thread that will never resume execution.

Since Delve has access to the memory of the traced (debugged) process, it can capture information stored by runtime data structures to analyze the state of the scheduler, and any M (thread) or G (goroutine) that is currently in existance. With this information, Delve can properly continue any threads needed for coordination during controlled execution of the debugged process.

### Roadmap for the future

Delve has come a long way since I first began working on it. All of the core functionality has been implemented, however there is more to be done. Variable evaluation could be improved, support for other (non Linux) platforms must be added, support for IDE integration, and possibly more useful and powerful features developed.

I have been blown away by the interest of the community in this project, and the recent contributions by various Gophers from around the world. Go has such an amazing community, and I encourage anybody with any interest in hacking on Delve to send in your contributions. In the end, Delve is an amazingly fun project to work on, and a useful tool for any Gophers toolbelt. I am dilligently working towards version 1.0, and with help from the Go community that milestone will come quickly.
