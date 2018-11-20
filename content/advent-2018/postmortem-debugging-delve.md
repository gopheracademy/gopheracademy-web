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

The source code and the details about the setup can be found in [github repository](https://github.com/narqo/postmortem-debug-go). To follow along, you will need a VM running Linux. I will use Vagrant with hostmanager plugin. Refer to Vagrantfile in the root of the repository.

To debug the problem, we need to bump into the problem first. Let’s start the VM, build our HTTP service, run it and see what will happen.

```
= vagrant ssh server-test-1
Welcome to Ubuntu 18.04.1 LTS (GNU/Linux 4.15.0-33-generic x86_64)

:~$ cd /vagrant/example/server
:/vagrant/example/server$ go build -o server ./
:/vagrant/example/server$ ./server --addr=:10080

server listening addr=:10080
```

Using [wrk][] HTTP benchmarking tool start adding some load. I run this demo on my MacBook with four cores. Running wrk with four threads and 1000 connections is enough to simulate the failure we want to debug. Run the following command in a new terminal panel:

```
= wrk -d1m -t4 -c1000 'http://server-test-1:10080'
Running 1m test @ http://server-test-1:10080
  4 threads and 1000 connections
  ···
```

After a brief period, the server gets stacked. Even after wrk finished the run, the service is unable to process an incoming request:

```
= curl --max-time 5 'http://server-test-1:10080/'
curl: (28) Operation timed out after 5001 milliseconds with 0 bytes received
```

Indeed, we have a problem! Let have a look.

---

In a similar situation with our production service, after a brief period, the total number of spawned goroutines for incoming requests has risen so much, the server became unresponsive. Even requests to pprof debug handlers were *s-u-u-u-per slow*, making it look like the server was completely "dead". Similarly, our attempts to kill the process with `SIGQUIT` to [get the stack dump of running goroutines][1] didn't seem to work.

---

## GDB and Coredump

We can start with trying to inspect the running service with GDB (GNU Debugger).

*Running a debugger in the production environment will likely require additional privileges. If in doubt, be wise and always consult with your operations team first.*

Connect to another SSH session on the VM, find server process's id, and attach to the process with the debugger:

```
= vagrant ssh server-test-1
Welcome to Ubuntu 18.04.1 LTS (GNU/Linux 4.15.0-33-generic x86_64)

:~$ ps -ef | grep server
vagrant   1628  1557  0 14:45 pts/0	00:00:01 ./server --addr=:10080

:~$ sudo gdb --pid=1628 /vagrant/example/server/server
```

With the debbuger attached to the process, we can run GDB's `bt` command (aka backtrace) to check the stack trace of the current thread:

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

Debugging a live process is “fun”, also let’s grab a coredump of the process to analyse it offline. We can do it with GDB's `gcore` command. The core file will be saved as `core.<process_id>` in the current working directory. Note, even for our simple server, the file will be pretty big. It’s likely to be huge for production service.

```
(gdb) gcore
Saved corefile core.1628
(gdb) quit

= du -h core.1628
1.2G    core.1628
```

With coredump saved, we're no longer required to keep the process running. Feel free to “kill -9” it.

*For more info on the topic of debugging with GDB, check out Go’s own "[Debugging Go Code with GDB][2]".*

## Enter Delve, Debugger for Go

[Delve][] is the debugger for Go programs. It is similar to GDB but is aware of Go's runtime, data structures and other internals of the language.

I highly recommend a talk by Alessandro Arzilli “[Internal Architecture of Delve, a Debugger For Go](https://www.youtube.com/watch?v=IKnTr7Zms1k)” from GopherCon EU 2018 if you're interested to know about Delve's internals.

Delve is written in Go so installing it as simple as running:

```
= go get -u github.com/derekparker/delve/cmd/dlv
```

With Delve installed we can start analysing the core file by running `dlv core <path to service binary> <core file>`. We start by listing all goroutines that were running when the coredump was taken. Delve's `goroutines` command does exactly this:

```
= dlv core example/server/server core.1628

(dlv) goroutines
  ···
  Goroutine 4611 - User: /vagrant/example/server/metrics.go:113 main.(*Metrics).CountS (0x703948)
  Goroutine 4612 - User: /vagrant/example/server/metrics.go:113 main.(*Metrics).CountS (0x703948)
  Goroutine 4613 - User: /vagrant/example/server/metrics.go:113 main.(*Metrics).CountS (0x703948)

```

*Unfortunately, in a real case scenario, the list can be so big, it doesn't even fit into terminal's scroll buffer. Remember that the server spawns a goroutine for each incoming request, so “goroutines” command has shown us a list of almost a million items. Let's pretend that we faced exactly this, and think of a way to work through this situation.*

We can run Delve in the "headless" mode to interact with the debugger via it's [JSON-RPC API](https://github.com/derekparker/delve/tree/master/Documentation/api).

Run the same `dlv core` command as we previously did, but this time specify that we need to start Delve’s API server:

```
= dlv core example/server/server core.1628 --listen :44441 --headless --log
API server listening at: [::]:44441
INFO[0000] opening core file core.1628 (executable example/server/server)  layer=debugger
```

After debug server is running, we can send commands to it’s TCP port and store the output as raw JSON. Let's get the list of running goroutines once again, but this time saving the results to a file:

```
= echo -n '{"method":"RPCServer.ListGoroutines","params":[],"id":2}' | nc localhost 44441 > server-test-1_dlv-rpc-list_goroutines.json
```

Now we have a (pretty big) JSON file with lots of information in it! To inspect any JSON data, I like to use [jq][]. To have an idea of what the data looks like, get the first five top objects from the JSON's `result` field:

```
= jq '.result[0:5]' server-test-1_dlv-rpc-list_goroutines.json
[
  {
    "id": 1,
    "currentLoc": {
      "pc": 4387627,
      "file": "/usr/local/go/src/runtime/proc.go",
      "line": 303,
      "function": {
        "name": "runtime.gopark",
        "value": 4387392,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "userCurrentLoc": {
      "pc": 7351940,
      "file": "/vagrant/example/server/main.go",
      "line": 56,
      "function": {
        "name": "main.run",
        "value": 7351152,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "goStatementLoc": {
      "pc": 4566329,
      "file": "/usr/local/go/src/runtime/asm_amd64.s",
      "line": 201,
      "function": {
        "name": "runtime.rt0_go",
        "value": 4566032,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "startLoc": {
      "pc": 4386096,
      "file": "/usr/local/go/src/runtime/proc.go",
      "line": 110,
      "function": {
        "name": "runtime.main",
        "value": 4386096,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "threadID": 0
  },
  {
    "id": 2,
    "currentLoc": {
      "pc": 4387627,
      "file": "/usr/local/go/src/runtime/proc.go",
      "line": 303,
      "function": {
        "name": "runtime.gopark",
        "value": 4387392,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "userCurrentLoc": {
      "pc": 4387627,
      "file": "/usr/local/go/src/runtime/proc.go",
      "line": 303,
      "function": {
        "name": "runtime.gopark",
        "value": 4387392,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "goStatementLoc": {
      "pc": 4387029,
      "file": "/usr/local/go/src/runtime/proc.go",
      "line": 240,
      "function": {
        "name": "runtime.init.4",
        "value": 4386976,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "startLoc": {
      "pc": 4387056,
      "file": "/usr/local/go/src/runtime/proc.go",
      "line": 243,
      "function": {
        "name": "runtime.forcegchelper",
        "value": 4387056,
        "type": 0,
        "goType": 0,
        "optimized": true
      }
    },
    "threadID": 0
  },
  ...
```

Every object in the JSON represents a single goroutine. By checking [“goroutines” command help](https://github.com/derekparker/delve/blob/master/Documentation/cli/README.md#goroutines) we can figure out what data Delve knows. We're interested in `userCurrentLoc` field, which is "topmost stackframe in user code".

Of all goroutines in the JSON let's list unique function names with the exact line numbers where function got paused:

```
= jq -c '.result[] | [.userCurrentLoc.function.name, .userCurrentLoc.line]' server-test-1_dlv-rpc-list_goroutines.json | sort | uniq -c

   1 ["internal/poll.runtime_pollWait",173]
1000 ["main.(*Metrics).CountS",113]
   1 ["main.(*Metrics).SetM",145]
   1 ["main.(*Metrics).startOutChannelConsumer",239]
   1 ["main.run",56]
   1 ["os/signal.signal_recv",139]
   6 ["runtime.gopark",303]
```

The majority of goroutines (1000) have stacked in `main.(*Metrics).CountS:113`. Now, this is the perfect time to look at the source code.

In the `main` package, find `Metrics` struct and look at its `CountS` method (see `example/server/metrics.go`):

```go
// CountS increments counter per second.
func (m *Metrics) CountS(key string) {
    // ···

    m.inChannel <- NewCountMetric(key, 1, second)
}
```

Our server has stacked on sending to the `inChannel` channel. Let’s find out who is supposed to read from this channel. After inspecting the code, we should find the following function (example/server/metrics.go):

```
// starts a consumer for inChannel
func (m *Metrics) startInChannelConsumer() {
    for inMetrics := range m.inChannel {
   	    // ···
    }
}
```

The function reads values out of the channel and does something with them, one by one. In what possible situations, could the sending to this channel being blocked?

When working with channels, there are only four possible "oopsies", according to Dave Cheney's [Channel Axioms](https://dave.cheney.net/2014/03/19/channel-axioms):

- send to a nil channel block forever
- receive from a nil channel block forever
- send to a closed channel panics
- receive from a closed channel returns the zero value immediately.

"Send to a nil channel block forever" – at first sight, this sounds like something possible. But, after double-checking with the code, `inChannel` is initialised in the `Metrics` constructor. So it can't be nil.

Let’s look at the list of functions we’ve got above. Could this (buffered) channel become full because we've stacked somewhere in `(*Metrics).startInChannelConsumer()`?

As you may notice, there is no `startInChannelConsumer` method in the list at all. But what if we stacked somewhere below the method’s callstack?

Delve provides the start position from where we came to the user location, that is `startLoc` field in the JSON. Search for goroutines whose start location was in `startInChannelConsumer` function:

```
= jq '.result[] | select(.startLoc.function.name | test("startInChannelConsumer$"))' server-test-1_dlv-rpc-list_goroutines.json

{
  "id": 20,
  "currentLoc": {
    "pc": 4387627,
    "file": "/usr/local/go/src/runtime/proc.go",
    "line": 303,
    "function": {
      "name": "runtime.gopark",
      "value": 4387392,
      "type": 0,
      "goType": 0,
      "optimized": true
    }
  },
  "userCurrentLoc": {
    "pc": 7355276,
    "file": "/vagrant/example/server/metrics.go",
    "line": 145,
    "function": {
      "name": "main.(*Metrics).SetM",
      "value": 7354992,
      "type": 0,
      "goType": 0,
      "optimized": true
    }
  },
  "goStatementLoc": {
    "pc": 7354080,
    "file": "/vagrant/example/server/metrics.go",
    "line": 95,
    "function": {
      "name": "main.NewMetrics",
      "value": 7353136,
      "type": 0,
      "goType": 0,
      "optimized": true
    }
  },
  "startLoc": {
    "pc": 7355584,
    "file": "/vagrant/example/server/metrics.go",
    "line": 167,
    "function": {
      "name": "main.(*Metrics).startInChannelConsumer",
      "value": 7355584,
      "type": 0,
      "goType": 0,
      "optimized": true
    }
  },
  "threadID": 0
}
```

There is a single item in the response. That's promising!

A goroutine with id "20" started at `main.(*Metrics).startInChannelConsumer:167` and went up to `main.(*Metrics).SetM:145` (`userCurrentLoc` field) until it got stacked.

Knowing the id of the goroutine dramatically narrows down our scope of interest (we don't need to dig into raw JSON anymore, I promise :). With Delve's `goroutine` command change current goroutine to the one we'd found. `stack` command will print the stack trace of the goroutine:

```
= dlv core example/server/server core.1628

(dlv) goroutine 20
Switched from 0 to 20 (thread 1628)

(dlv) stack -full
0  0x000000000042f32b in runtime.gopark
   at /usr/local/go/src/runtime/proc.go:303
   	lock = unsafe.Pointer(0xc0000824d8)
   	reason = waitReasonChanSend
   	traceEv = 22
   	traceskip = 3
   	unlockf = (unreadable empty OP stack)
   	gp = (unreadable could not find loclist entry at 0x2f3f8 for address 0x42f32b)
   	mp = (unreadable could not find loclist entry at 0x2f45f for address 0x42f32b)
   	status = (unreadable could not find loclist entry at 0x2f4c6 for address 0x42f32b)

1  0x000000000042f3d3 in runtime.goparkunlock
   at /usr/local/go/src/runtime/proc.go:308
   	lock = (unreadable empty OP stack)
   	reason = (unreadable empty OP stack)
   	traceEv = (unreadable empty OP stack)
   	traceskip = (unreadable empty OP stack)

2  0x00000000004069cd in runtime.chansend
   at /usr/local/go/src/runtime/chan.go:234
   	block = true
   	c = (*runtime.hchan)(0xc000082480)
   	callerpc = (unreadable empty OP stack)
   	ep = unsafe.Pointer(0xc000031e30)
   	~r4 = (unreadable empty OP stack)
   	gp = (*runtime.g)(0xc000001680)
   	mysg = *(unreadable read out of bounds)
   	t0 = 0

3  0x00000000004067a5 in runtime.chansend1
   at /usr/local/go/src/runtime/chan.go:125
   	c = (unreadable empty OP stack)
   	elem = (unreadable empty OP stack)

4  0x0000000000703b8c in main.(*Metrics).SetM
   at /vagrant/example/server/metrics.go:145
   	key = "metrics.raw_channel"
   	m = (*main.Metrics)(0xc000134000)
   	value = 100

5  0x0000000000704394 in main.(*Metrics).sendMetricsToOutChannel
   at /vagrant/example/server/metrics.go:206
   	m = (*main.Metrics)(0xc000134000)
   	scope = 0
   	updateInterval = (unreadable could not find loclist entry at 0x9d4ed for address 0x704393)

6  0x0000000000703f40 in main.(*Metrics).startInChannelConsumer
   at /vagrant/example/server/metrics.go:184
   	m = (*main.Metrics)(0xc000134000)
   	inMetrics = main.Metric {Type: TypeCount, Scope: 0, Key: "server.req-incoming",...+2 more}
   	nextUpdate = (unreadable could not find loclist entry at 0x9d3fd for address 0x703f3f)

7  0x000000000045cf11 in runtime.goexit
   at /usr/local/go/src/runtime/asm_amd64.s:1333
```

Bottom to top:

(6) At `(*Metrics).startInChannelConsumer:184` a new value from the channel has been received

(5) We called `(*Metrics).sendMetricsToOutChannel` first

(4) And `main.(*Metrics).SetM` next, passing `key="metrics.raw_channel"` and `value = 100`

And so on until we've been blocked in `runtime.gopark` with “waitReasonChanSend”.

Everything makes sense now!

Within a single goroutine, the function that reads values out of a buffered channel tried to put additional values into the channel. As the number of incoming values to the channel became close to its capacity, the consumer-function deadlocked itself trying to add value to the full channel. Since the single channel's consumer was deadlocked, every new incoming request that tried adding values into the channel became blocked as well.

----

And that’s our story. Using the described technique we’ve managed to find the root-cause the problem. The original piece of code was written many years ago. Nobody even looked at it and never thought it might bring such issues.

As you just saw not everything is yet ideal with the tooling. But the tooling exists and becomes better over time. I hope, I’ve encouraged you to give it a try. And I’m very interested to hear about other ways to work around a similar scenario.


*Vladimir is a Backend Developer at adjust.com. @tvii on Twitter, @narqo on Github.*

[1]: https://golang.org/pkg/os/signal/#hdr-Default_behavior_of_signals_in_Go_programs
[2]: https://golang.org/doc/gdb
[wrk]: https://github.com/wg/wrk
[Delve]: https://github.com/derekparker/delve
[jq]: https://stedolan.github.io/jq/
