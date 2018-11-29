+++
author = ["Vladimir Varankin"]
title = "Postmortem debugging Go services with Delve"
linktitle = "Postmortem debugging Go services with Delve"
date = 2018-12-02
series = ["Advent 2018"]
+++

One day, several instances of one of our production services stopped accepting incoming traffic. HTTP requests successfully went through the load balancer reaching the instance and just hanged. What followed became an exciting exercise in debugging of a running production service written in Go.

Below is a step-by-step guide that demonstrates the process which helped us in identifying the root causes of the problem.

To make things easier we will take a simple HTTP service, written in Go, as our debugging target. The implementation details of the service are not very important now (we will dig into the code later). A real-world production service will likely to consist of many different components, that implement business logic and the infrastructure of the service. Let’s convince ourselves that the service was already “battle-tested” by running it production for many months :)

The source code and the details about the setup can be found in [repository][] on GitHub. To follow along, you will need a VM running Linux. I use Vagrant with the [vagrant-hostmanager](https://github.com/sevos/vagrant-hostmanager) plugin. Have a look at the [`Vagrantfile`](https://github.com/narqo/postmortem-debug-go/blob/master/Vagrantfile) in the root of the repository for detailed information.

To debug the problem, we need to bump into the problem first. Let’s start the VM, build our HTTP service, run it and see what will happen:

```
$ vagrant up
Bringing machine 'server-test-1' up with 'virtualbox' provider...

$ vagrant ssh server-test-1
Welcome to Ubuntu 18.04.1 LTS (GNU/Linux 4.15.0-33-generic x86_64)
···
vagrant@server-test-1:~$ cd /vagrant/example/server
vagrant@server-test-1:/vagrant/example/server$ go build
vagrant@server-test-1:/vagrant/example/server$ ./server --addr=:10080
server listening addr=:10080
```

We can test the server is workin by sending a request using curl. In a new terminal window run the following command:

```
$ curl 'http://server-test-1:10080'
OK
```

To simulate the failure we want to debug we need to send a bunch of requests. We can do this with [wrk][] HTTP benchmarking tool. My MacBook has four cores, so running wrk with four threads and 1000 connections is usually enough.

```
$ wrk -d1m -t4 -c1000 'http://server-test-1:10080'
Running 1m test @ http://server-test-1:10080
  4 threads and 1000 connections
  ···
```

After a brief period the server freezes. Even after wrk finished the run, the server is unable to process an incoming request:

```
$ curl --max-time 5 'http://server-test-1:10080/'
curl: (28) Operation timed out after 5001 milliseconds with 0 bytes received
```

Indeed, we have a problem! Let's have a look.

---

*In the real situation with our production service, after the server has started, the total number of spawned goroutines for incoming requests has risen so much that the server became unresponsive. Requests to pprof debug handlers were s-u-u-u-per slow, making it look like the server was completely "dead". Similarly, our attempts to kill the process with `SIGQUIT` to [get the stack dump of running goroutines][1] didn't seem to work.*

---

### GDB and Coredump

We can start with trying to inspect the running service with GDB (GNU Debugger).

---

*Running a debugger in the production environment will likely require additional privileges. If in doubt, be wise and always consult with your operations team first.*

---

Connect to another SSH session on the VM, find server's process id, and attach to the process with the debugger:

```
$ vagrant ssh server-test-1
Welcome to Ubuntu 18.04.1 LTS (GNU/Linux 4.15.0-33-generic x86_64)
···
vagrant@server-test-1:~$ pgrep server
1628
vagrant@server-test-1:~$ cd /vagrant
vagrant@server-test-1:/vagrant$ sudo gdb --pid=1628 example/server/server
GNU gdb (Ubuntu 8.1-0ubuntu3) 8.1.0.20180409-git
···
```

With the debugger attached to the process, we can run GDB's `bt` command (aka backtrace) to check the stack trace of the current thread:

```
(gdb) bt
#0  runtime.futex () at /usr/local/go/src/runtime/sys_linux_amd64.s:532
#1  0x000000000042b08b in runtime.futexsleep (addr=0xa9a160 <runtime.m0+320>, ns=-1, val=0) at /usr/local/go/src/runtime/os_linux.go:46
#2  0x000000000040c382 in runtime.notesleep (n=0xa9a160 <runtime.m0+320>) at /usr/local/go/src/runtime/lock_futex.go:151
#3  0x0000000000433b4a in runtime.stoplockedm () at /usr/local/go/src/runtime/proc.go:2165
#4  0x0000000000435279 in runtime.schedule () at /usr/local/go/src/runtime/proc.go:2565
#5  0x00000000004353fe in runtime.park_m (gp=0xc000066d80) at /usr/local/go/src/runtime/proc.go:2676
#6  0x000000000045ae1b in runtime.mcall () at /usr/local/go/src/runtime/asm_amd64.s:299
#7  0x000000000045ad39 in runtime.rt0_go () at /usr/local/go/src/runtime/asm_amd64.s:201
#8  0x0000000000000000 in ?? ()
```

Honestly, I’m not a GDB expert, but it seems that Go runtime is putting threads to sleep. *But why?*

Debugging a working process is “fun” but let’s save a coredump of the process and analyse it offline. We can do this with GDB's `gcore` command. The core file will be saved as `core.<process_id>` in the current working directory.

```
(gdb) gcore
Saved corefile core.1628
(gdb) quit
A debugging session is active.

	Inferior 1 [process 1628] will be detached.

Quit anyway? (y or n) y
Detaching from program: /vagrant/example/server/server, process 1628
```

With core file saved, we're no longer required to keep the process running. Feel free to “kill -9” it.

Note that even for our simple server, the core file will be pretty big (for me it was 1.2G). For a real production service it’s likely to be huge.

*For more info on the topic of debugging with GDB, check out Go’s own "[Debugging Go Code with GDB][2]".*

### Enter Delve, Debugger for Go

[Delve][] is the debugger for Go programs. It is similar to GDB but is aware of Go's runtime, data structures and other internals of the language.

I highly recommend a talk by Alessandro Arzilli “[Internal Architecture of Delve, a Debugger For Go](https://www.youtube.com/watch?v=IKnTr7Zms1k)” from GopherCon EU 2018 if you're interested to know about Delve's internals.

Delve is written in Go so installing it as simple as running:

```
$ go get -u github.com/derekparker/delve/cmd/dlv
```

With Delve installed we can start analysing the core file by running `dlv core <path to service binary> <core file>`. We start by listing all goroutines that were running when the coredump was taken. Delve's `goroutines` command does exactly this:

```
$ dlv core example/server/server core.1628

(dlv) goroutines
  ···
  Goroutine 4611 - User: /vagrant/example/server/metrics.go:113 main.(*Metrics).CountS (0x703948)
  Goroutine 4612 - User: /vagrant/example/server/metrics.go:113 main.(*Metrics).CountS (0x703948)
  Goroutine 4613 - User: /vagrant/example/server/metrics.go:113 main.(*Metrics).CountS (0x703948)
```

Unfortunately, in a real case scenario, the list can be so big, it doesn't even fit into terminal's scroll buffer. Remember that the server spawns a goroutine for each incoming request, so “goroutines” command has shown us a list of almost a million items. Let's pretend that we faced exactly this and think of a way to work through this situation.

Delve allows running it in the "headless" mode and interact with the debugger via [JSON-RPC API](https://github.com/derekparker/delve/tree/master/Documentation/api).

Run the same `dlv core` command as we just did, but this time specify that we want starting Delve’s API server:

```
$ dlv core example/server/server core.1628 --listen :44441 --headless --log
API server listening at: [::]:44441
INFO[0000] opening core file core.1628 (executable example/server/server)  layer=debugger
```

After debug server is running, we can send commands to its TCP port and store the output as raw JSON. Let's get the same list of running goroutines we've just seen, but this time saving the results to a file:

```
$ echo -n '{"method":"RPCServer.ListGoroutines","params":[],"id":2}' | nc -w 1 localhost 44441 > server-test-1_dlv-rpc-list_goroutines.json
```

Now we have a (pretty big) JSON file with lots of raw information. I like to use [jq][] command to inspect any JSON data and just to have an idea of what the data looks like, I query the first three objects in the JSON's `result` field:

```
$ jq '.result[0:3]' server-test-1_dlv-rpc-list_goroutines.json
[
  {
    "id": 1,
    "currentLoc": {
      "pc": 4380603,
      "file": "/usr/local/go/src/runtime/proc.go",
      "line": 303,
      "function": {
        "name": "runtime.gopark",
        "value": 4380368,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "userCurrentLoc": {
      "pc": 6438159,
      "file": "/vagrant/example/server/main.go",
      "line": 52,
      "function": {
        "name": "main.run",
        "value": 6437408,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "goStatementLoc": {
      "pc": 4547433,
      "file": "/usr/local/go/src/runtime/asm_amd64.s",
      "line": 201,
      "function": {
        "name": "runtime.rt0_go",
        "value": 4547136,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "startLoc": {
      "pc": 4379072,
      "file": "/usr/local/go/src/runtime/proc.go",
      "line": 110,
      "function": {
        "name": "runtime.main",
        "value": 4379072,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "threadID": 0,
    "unreadable": ""
  },
  {
    "id": 2,
    "currentLoc": {
      "pc": 4380603,
      "file": "/usr/local/go/src/runtime/proc.go",
      "line": 303,
      "function": {
        "name": "runtime.gopark",
        "value": 4380368,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "userCurrentLoc": {
      "pc": 4380603,
      "file": "/usr/local/go/src/runtime/proc.go",
      "line": 303,
      "function": {
        "name": "runtime.gopark",
        "value": 4380368,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "goStatementLoc": {
      "pc": 4380005,
      "file": "/usr/local/go/src/runtime/proc.go",
      "line": 240,
      "function": {
        "name": "runtime.init.4",
        "value": 4379952,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "startLoc": {
      "pc": 4380032,
      "file": "/usr/local/go/src/runtime/proc.go",
      "line": 243,
      "function": {
        "name": "runtime.forcegchelper",
        "value": 4380032,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "threadID": 0,
    "unreadable": ""
  },
  ...
```

Every object in the JSON represents a single goroutine. [`goroutines` command manual](https://github.com/derekparker/delve/blob/master/Documentation/cli/README.md#goroutines) tells us what Delve knows about each goroutine. We're interested in `userCurrentLoc` field, which is, as manual describes it, the "topmost stackframe in user code", meaning it is the last location in the service code the goroutine came across.

In order to get an overview of what goroutines did at the moment the core file was created, let's collect and count all distinct function names and line numbers for all `userCurrentLoc` fields in the JSON.

```
$ jq -c '.result[] | [.userCurrentLoc.function.name, .userCurrentLoc.line]' server-test-1_dlv-rpc-list_goroutines.json | sort | uniq -c

   1 ["internal/poll.runtime_pollWait",173]
1000 ["main.(*Metrics).CountS",95]
   1 ["main.(*Metrics).SetM",105]
   1 ["main.(*Metrics).startOutChannelConsumer",179]
   1 ["main.run",52]
   1 ["os/signal.signal_recv",139]
   6 ["runtime.gopark",303]
```

The majority of goroutines (1000 in the snippet above) have stuck in function `main.(*Metrics).CountS` at line 95. Now, this is the time to look at the [source code][repository] of our service.

In the package `main`, find `Metrics` struct and look at its `CountS` method (see [`example/server/metrics.go`](https://github.com/narqo/postmortem-debug-go/blob/2c42ca73ebd500fe8da1c6ac8ecaf4af143aca78/example/server/metrics.go#L94)):

```go
// CountS increments counter per second.
func (m *Metrics) CountS(key string) {
    m.inChannel <- NewCountMetric(key, 1, second)
}
```

Our server has stuck on sending to the `inChannel` channel. Let’s find out who is supposed to read from this channel. After inspecting the code, we should find the following function ([example/server/metrics.go](https://github.com/narqo/postmortem-debug-go/blob/2c42ca73ebd500fe8da1c6ac8ecaf4af143aca78/example/server/metrics.go#L109)):

```
// starts a consumer for inChannel
func (m *Metrics) startInChannelConsumer() {
    for inMetrics := range m.inChannel {
   	    // ···
    }
}
```

The function reads values out of the channel and does something with them, one by one. In what possible situations could the sending to this channel being blocked?

When working with channels, there are only four possible "oopsies", according to Dave Cheney's [Channel Axioms](https://dave.cheney.net/2014/03/19/channel-axioms):

- send to a nil channel blocks forever
- receive from a nil channel blocks forever
- send to a closed channel panics
- receive from a closed channel returns the zero value immediately.

"Send to a nil channel block forever" – at first sight, this seems like something possible. But, after double-checking with the code, `inChannel` is [initialised in the `Metrics` constructor](https://github.com/narqo/postmortem-debug-go/blob/2c42ca73ebd500fe8da1c6ac8ecaf4af143aca78/example/server/metrics.go#L73). So it can't be nil.

As you may notice, there was no `startInChannelConsumer` method in the list of function we've previously collected with jq. Could this (buffered) channel become full because we've stuck somewhere inside `main.(*Metrics).startInChannelConsumer()`?

Delve provides the start position from where we came to the location in the code described in `userCurrentLoc` field in the JSON. This location is stored in `startLoc` field. With the following jq command search for all goroutines whose start location was in `startInChannelConsumer` function:

```
$ jq '.result[] | select(.startLoc.function.name | test("startInChannelConsumer$"))' server-test-1_dlv-rpc-list_goroutines.json

{
  "id": 20,
  "currentLoc": {
    "pc": 4380603,
    "file": "/usr/local/go/src/runtime/proc.go",
    "line": 303,
    "function": {
      "name": "runtime.gopark",
      "value": 4380368,
      "type": 0,
      "goType": 0,
      "optimized": true
    }
  },
  "userCurrentLoc": {
    "pc": 6440847,
    "file": "/vagrant/example/server/metrics.go",
    "line": 105,
    "function": {
      "name": "main.(*Metrics).SetM",
      "value": 6440672,
      "type": 0,
      "goType": 0,
      "optimized": true
    }
  },
  "startLoc": {
    "pc": 6440880,
    "file": "/vagrant/example/server/metrics.go",
    "line": 109,
    "function": {
      "name": "main.(*Metrics).startInChannelConsumer",
      "value": 6440880,
      "type": 0,
      "goType": 0,
      "optimized": true
    }
  },
  ···
}
```

There is a single item in the result. That's promising!

A goroutine with id "20" started in `main.(*Metrics).startInChannelConsumer` at line 109 (see `startLoc` field in the result) and went up to the `main.(*Metrics).SetM` line 105 (`userCurrentLoc` field), and got stuck there.

Knowing the id of the goroutine dramatically narrows down our scope of interest (*and we don't need to dig into raw JSON anymore, I promise* :). With Delve's `goroutine` command we change current goroutine to the one we've found. Then we can use `stack` command to print the stack trace of this goroutine:

```
$ dlv core example/server/server core.1628

(dlv) goroutine 20
Switched from 0 to 20 (thread 1628)

(dlv) stack -full
0  0x000000000042d7bb in runtime.gopark
   at /usr/local/go/src/runtime/proc.go:303
       lock = unsafe.Pointer(0xc000104058)
       reason = waitReasonChanSend
···
3  0x00000000004066a5 in runtime.chansend1
   at /usr/local/go/src/runtime/chan.go:125
       c = (unreadable empty OP stack)
       elem = (unreadable empty OP stack)

4  0x000000000062478f in main.(*Metrics).SetM
   at /vagrant/example/server/metrics.go:105
       key = (unreadable empty OP stack)
       m = (unreadable empty OP stack)
       value = (unreadable empty OP stack)

5  0x0000000000624e64 in main.(*Metrics).sendMetricsToOutChannel
   at /vagrant/example/server/metrics.go:146
       m = (*main.Metrics)(0xc000056040)
       scope = 0
       updateInterval = (unreadable could not find loclist entry at 0x89f76 for address 0x624e63)

6  0x0000000000624a2f in main.(*Metrics).startInChannelConsumer
   at /vagrant/example/server/metrics.go:127
       m = (*main.Metrics)(0xc000056040)
       inMetrics = main.Metric {Type: TypeCount, Scope: 0, Key: "server.req-incoming",...+2 more}
       nextUpdate = (unreadable could not find loclist entry at 0x89e86 for address 0x624a2e)
```

Bottom to top:

(6) At `main.(*Metrics).startInChannelConsumer` a new `inMetrics` value from the channel has been received

(5) We called `main.(*Metrics).sendMetricsToOutChannel` and processed to line 146 of `example/server/metrics.go`

(4) Then `main.(*Metrics).SetM` was called.

And so on until we've been blocked in `runtime.gopark` with `waitReasonChanSend`.

Everything makes sense now!

Within a single goroutine, the function that reads values out of a buffered channel tried to put additional values into the channel. As the number of incoming values to the channel became close to its capacity, the consumer-function deadlocked itself trying to add value to the full channel. Since the single channel's consumer was deadlocked, every new incoming request that tried adding values into the channel became blocked as well.

----

And that’s our story. Using the described technique we’ve managed to find the root cause of the problem. The original piece of code was written many years ago. Nobody even looked at it and never thought it might bring such issues.

As you just saw not everything is ideal with the tooling yet. But the tools exist and become better over time. I hope, I’ve encouraged you to give them a try. And I’m very interested to hear about other ways to work around a similar scenario.


*Vladimir is a Backend Developer at adjust.com. @tvii on Twitter, @narqo on Github.*

[1]: https://golang.org/pkg/os/signal/#hdr-Default_behavior_of_signals_in_Go_programs
[2]: https://golang.org/doc/gdb
[repository]: https://github.com/narqo/postmortem-debug-go
[wrk]: https://github.com/wg/wrk
[Delve]: https://github.com/derekparker/delve
[jq]: https://stedolan.github.io/jq/
