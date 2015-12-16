+++
author = ["NATS.io team"]
date = "2015-12-11T13:45:04-05:00"
linktitle = "NATS: High Performance Messaging"
series = ["Advent 2015"]
title = "NATS: High Performance Cloud Native Messaging Written in Go"

+++


Performance at scale is critically important for anybody attempting to build distributed systems today. Pieces of a given service might be spread across physical or virtual infrastructure, or might be comprised of thousands or even millions of devices (Internet of Things). But, to the end user they need to operate seamlessly - as though they are one entity. This requires extremely fast, lightweight, always-on communication.

NATS is an extremely lightweight, and massively scalable Publish/Subscribe (PubSub) messaging system for Cloud Native applications. It was originally created by [Derek Collison] (https://twitter.com/derekcollison) as the messaging layer inside of Cloud Foundry, when he was designing that product. The original version of [NATS](www.nats.io) was written in Ruby, but was ported over to Go. The Go implementation of the NATS server is called [gnatsd](https://github.com/nats-io/gnatsd), and immediately offered performance well in excess of Ruby-nats.

When Derek founded Apcera, NATS was again used as the control plane inside the product, but was ported to Go, since Apcera is a very big proponent of Go for scalable distributed systems architectures, for many reasons which are explained in depth during a [presentation](https://www.youtube.com/watch?v=qC9WhjmewIk) in 2014 by Derek (eg. how Go handles concurrency, the simplicity of the compiled language, etc.).

As important as understanding what NATS is, is realizing what NATS is not. NATS is not intended as a traditional enterprise messaging system - you can think of it more as an ephemeral nervous system, that is always on, and always available. By sticking to the core tenets of simplicity and speed, NATS - much like Go - provides an excellent foundation for delivering modern distributed systems at scale.

This was quickly tuned for performance enhancements (eg. the initial version used regexps to parse protocol messages, whereas the current implementation uses a custom parser with zero allocations). Third party benchmarking in 2014 clocked NATS at ~6 Million messages / second, while maintaining ultra-low latency. Today, NATS is capable of sending approximately 8 million messages / second at minimal latency, and its speed continues to climb.

Another characteristic of Go that we really like at Apcera - which made it ideal for NATS - is simplicity. At it’s core, NATS is designed to be extremely lightweight, and extremely fast. It provides an always-on dial tone for the foundation of your distributed systems, and it doesn’t make any underlying assumptions about the audience (i.e. subscribers). It’s by sticking to these core tenets of simplicity that NATS is able to scale at speed well in excess of other messaging systems.

## Some of the interesting aspects of NATS include:

### Very simple [plain-text protocol] (http://nats.io/documentation/internals/nats-protocol/)
Unlike traditional messaging systems that use a binary message format that require an API to consume, the text-based NATS protocol makes it easy to implement clients in a wide variety of programming and scripting languages.
The NATS server implements a [zero allocation byte parser](https://youtu.be/ylRKac5kSOk?t=10m46s) that is fast and efficient.

### Subject Routing
Matches subjects to subscribers using a [trie of nodes and hashmaps](https://github.com/nats-io/gnatsd/blob/master/sublist/sublist.go).
Uses [] byte as keys, but avoids byte-to-string conversions.

### One of the smallest images on Docker Hub
Thanks to Go resulting binaries being compact in size, the image itself is also lightweight (official Docker image is less than 10MB).

### Auto-Pruning of Interest Graphs
To Support resiliency and high-availability, NATS provides built-in mechanisms to automatically prune the registered listener interest graph - including slow consumers and lazy listeners.
Slow Consumers - if a consumer is not processing messages quickly enough, the NATS server shuts it off. Each of the connections has a pending state. The NATS server does an accounting on the number of bytes the subscriber has yet to process. When it reaches max pending, which is a constant (default 10 MB), the NATS server disconnects the client. The max byte threshold is [configurable](http://nats.io/documentation/server/gnatsd-config/).
Lazy Listeners - To support scaling, NATS provides for auto-pruning of client connections. If a subscriber does not respond to ping requests from the server within the [ping-pong interval](http://nats.io/documentation/internals/nats-protocol), the client is cut off (disconnected). The client will need to have reconnect logic to reconnect with the server.

One Go idiom that NATS has implemented is the use of networked channels.  This makes writing a go NATS application simple and straightforward.

Here is simple code to connect to a NATS server, create a networked channel, and write ten integers:

```go

nc, nil := nats.Connect(nats.DefaultURL)
ec, nil := nats.NewEncodedConn(nc, nats.DEFAULT_ENCODER)
defer ec.Close()

ch := make(chan int, 1024)
if err := ec.BindSendChan("foo", ch); err != nil {
  log.Fatalf("Failed to bind to a send channel: %v\n", err)
}

// send 10 integers
for i := 0; i < 10; i++ {
  ch <- i
}

ec.Flush()

And the corresponding NATS code to receive:

nc, nil := nats.Connect(nats.DefaultURL)
ec, nil := nats.NewEncodedConn(nc, nats.DEFAULT_ENCODER)
defer ec.Close()

ch := make(chan int, 1024)
if _, err := ec.BindRecvChan("foo", ch); err != nil {
  log.Fatalf("Failed to bind to a recv channel: %v\n", err)
}

// receive 10 integers
for i := 0; i < 10; i++ {
  val := <- ch
  fmt.Printf("received: %v\n", val)
}
```
Sending complex data types can be accomplished through further encoding, such as the JSON encoder.

### Recent Updates:

## TLS/SSL
You can set up your own TLS enabled config file, a single self-signed server, or a cluster
TLS is currently supported in the Go, C, C#, Node.js, and Ruby clients, and will soon be available in all Apcera supported clients.
[Here is an example](https://github.com/nats-io/gnatsd/blob/master/test/tls_test.go#L15) of how to set up a Go client connection.


You can read more about setting up your Go environment for NATS [here.](http://nats.io/documentation/tutorials/go-install/)

If you like to get involved with NATS, there are many ways to do so! We'd love to hear your feedback and welcome you to the [community](http://nats.io/community/).

NATS Twitter: [@nats_io](http://www.twitter.com/nats_io)
<br>
NATS Github: [github.com/nats-io](https://github.com/nats-io)
<br>
Request to join the NATS [Slack Community](https://docs.google.com/a/apcera.com/forms/d/104yA7oqq7SPoMDG_J9MnVE74gVwBnTmVHKP5ABHoM5k/viewform?embedded=true)
