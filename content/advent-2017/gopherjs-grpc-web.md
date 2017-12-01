+++
author = ["Johan Brandhorst"]
date = "2017-12-01"
linktitle = "Using GopherJS with gRPC-Web"
series = ["Advent 2017"]
title = "Using GopherJS with gRPC-Web"

+++

## Introduction

This article will talk about how to connect a GopherJS frontend to a
Go backend. If you haven't heard about GopherJS before, it's an open
source Go-to-JavaScript transpiler, allowing us to write Go code and
run it in the browser. I recommend taking a look at the
[official GitHub repo](https://github.com/gopherjs/gopherjs) and
Dmitri Shuralyov's DotGo presentation
[_Go in the browser_](https://www.dotconferences.com/2016/10/dmitri-shuralyov-go-in-the-browser)
for a deeper introduction.

Writing GopherJS apps is great fun and lets us avoid writing JavaScript
and all the problems associated with it. However, we'll often want to
communicate with a backend server in order to read or write state or
issue RPC calls to other backend servers. This is generally done via a
RESTful JSON API, or maybe something like GraphQL with JSON. But using
these approaches come with several downsides:

* Having two sources of truth for the interface
* Loosely typed JSON objects
* Complex API versioning
* No streaming support
* JSON Marshalling/Unmarshalling is
[_slow_](https://auth0.com/blog/beating-json-performance-with-protobuf/)

Fortunately there's now a great alternative to REST and GraphQL which
builds on the existing [gRPC ecosystem](https://grpc.io/). gRPC, of
course, is the open source RPC framework developed by Google, donated
to the [CNCF](https://www.cncf.io/), and generally accepted as one of
the best ways to faciliate RPCs between microservices today. It
usually uses `protobuf` as the payload, but is designed to be agnostic
of the payload layer.
[Protobuf](https://developers.google.com/protocol-buffers/), of course,
is a payload format, also developed by Google, used for fast and
efficient data transfers.

## gRPC-Web

gRPC-Web is essentially a gRPC client in the browser.
It allows the use of normal gRPC-like requests over the wire,
with binary marshalled protobuf messages as the payload. It currently
requires a small proxy to be compatible with existing gRPC backends,
but this requirement will eventually be dropped, as it becomes
possible to implement the gRPC wire protocol in the browser.

It has an
[official spec](https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-WEB.md),
but currently no official client. Google was working on a client in a
private repo, which was scheduled for a Q3 2017 release, but work has
mostly died down for now. Instead, there's a spec-compliant
[third party client](https://github.com/improbable-eng/grpc-web/)
available, developed by [Improbable](https://improbable.io/).
Improbable wrote
[a great blog post](https://improbable.io/games/blog/grpc-web-moving-past-restjson-towards-type-safe-web-apis)
to introduce the library, which I encourage you to read.

gRPC-Web supports unary and server-side streaming methods only at this
time, pending the finalizing and implementing of the
[WHATWG Streams API](https://streams.spec.whatwg.org/) in browsers.

The Improbable client is written in TypeScript, which is better than
JavaScript, but obviously not as good as Go. This is why I created the
GopherJS gRPC-Web bindings.

## GopherJS gRPC-Web bindings

The
[GopherJS gRPC-Web bindings](https://github.com/johanbrandhorst/protobuf)
allows the use of gRPC as easily as any other Go gRPC client, and with
a suitably proxied gRPC backend it can act as the communications layer
between the frontend and backend of a website. It supports all 4
streaming modes supported by gRPC, bridging the client-side streaming
gap in the gRPC-Web spec with the
[use of WebSockets](https://jbrandhorst.com/post/client-streaming/).

Working with protobuf usually consists of 3 steps, and with my bindings,
it is no different:

1. Define the interface
1. Generate the code
1. Use the generated code

I'll now give a quick example of how to use the bindings, which should
look extremely familiar if you've already used protobuf and gRPC. If
you want to, you can following along these instructions after cloning
[my boilerplate repo](https://github.com/johanbrandhorst/grpcweb-boilerplate).

### Define the interface

We'll define a simple unary server method for getting information about
a user, with the user ID as the input. We'll also add a fancy
bi-directional streaming method. The following `protofile`
is all you need to generate the server and client interfaces to perform
these RPC calls:

```protobuf
syntax="proto3";

package site;

message GetUserRequest {
    string user_id = 1;
}

message User {
    string user_id = 1;
    string name = 2;
    uint32 age = 3;
}

message ChatMessage {
    string name = 1;
    string message = 2;
}

service Website {
  // GetUser fetches user details from a user ID
  rpc GetUser(GetUserRequest) returns (User) {}
  // Chat allows asynchronous communications between
  // the client and the server.
  rpc Chat(stream ChatMessage) returns (stream ChatMessage) {}
}
```

### Generate the code

We generate the server and client interfaces using
[`protoc`](https://github.com/google/protobuf). Using `protoc` can be
daunting, but I won't dedicate time to it in this post, if you want
more of an introduction, I recommend reading
[my blog post](https://jbrandhorst.com/post/go-protobuf-tips/) on the
subject. The following command will generate both the server and client
code and interfaces:

```bash
$ protoc site.proto --go_out=plugins=grpc:./server/ --gopherjs_out=plugins=grpc:./client/
```

We use the standard Go protobuf plugin,
[`protoc-gen-go`](https://github.com/golang/protobuf/tree/master/protoc-gen-go),
and my GopherJS protobuf plugin,
[`protoc-gen-gopherjs`](https://github.com/johanbrandhorst/protobuf/tree/master/protoc-gen-gopherjs)
to generate the server and client respectively. It is important that
the generated files are put into different folders as they expect to
define their own packages. This will generate two files:
`server/site.pb.go` and `client/site.pb.gopherjs.go`.

### Use the generated code

The code generated by `protoc-gen-go` defines an interface that the
backend must implement. In our case, it looks like this:

```go
type WebsiteServer interface {
	// GetUser fetches user details from a user ID
	GetUser(context.Context, *GetUserRequest) (*User, error)
	// Chat allows asynchronous communications between
	// the client and the server.
	Chat(Website_ChatServer) error
}
```

This is all standard gRPC stuff, so I wont talk too much about this.

The code generated by `protoc-gen-gopherjs` instead exposes a client
that, when pointed at a gRPC server, allows calling to the functions
implemented by the backend. The functions exposed intentionally
mirror that of the client generated by `protoc-gen-go`, in order
to make it more familiar. Lets take a look:

```go
type WebsiteClient interface {
	// GetUser fetches user details from a user ID
	GetUser(ctx context.Context, in *GetUserRequest, opts ...grpcweb.CallOption) (*User, error)
	// Chat allows asynchronous communications between
	// the client and the server.
	Chat(ctx context.Context, opts ...grpcweb.CallOption) (Website_ChatClient, error)
}

// NewWebsiteClient creates a new gRPC-Web client.
func NewWebsiteClient(hostname string, opts ...grpcweb.DialOption) WebsiteClient {
	...
}
```

The generated functions take a context, that can be used for cancellation,
just like in a normal gRPC request. It optionally takes `grpc.CallOption`s
that allow per-call settings to be applied. Supported dial options can be
found on
[`godoc`](https://godoc.org/github.com/johanbrandhorst/protobuf/grpcweb).
Note that some dial options are not entirely supported by
the bi-directional streaming method type.

The `NewWebsiteClient` function takes a hostname, the address of the server
to connect to, and optionally some `DialOption`s, that allow per-client
settings.

All the functions exposed by the GopherJS gRPC-Web bindings block until
their respective calls have completed.

## Server side requirements

As mentioned earlier, the GopherJS gRPC-Web bindings (for now) require
a small proxy in front of a generic gRPC server. The proxy is developed
by Improbable, and there are two different packages. Since we're working
with a Go gRPC backend, we'll use the importable package which makes
this extremely simple. This is all that is required to proxy gRPC-Web
requests into gRPC requests in your backend:

```go
import "github.com/improbable-eng/grpc-web/go/grpcweb"
...
gs := grpc.NewServer()
...
wrappedServer := grpcweb.WrapServer(gs)
```

It's as simple as wrapping the `*grpc.Server` with `grpcweb.WrapServer`.

In addition, if you want to use the bi-directional streaming
capabilities of the GopherJS bindings, you'll need to wrap the server again.

```go
import "github.com/johanbrandhorst/protobuf/wsproxy"
...
wsproxy := wsproxy.WrapServer(http.HandlerFunc(wrappedServer.ServeHTTP))
```

This second wrapper intercepts websocket requests made to the server and
translates them into gRPC streaming requests to support client side
and bi-directional streaming.

## Wrapping up

The GopherJS gRPC-Web bindings solve most of the problems associated with
a classic RESTful JSON API:

* One source of truth for the interface design - the `protofile`.
* Strongly typed data structures via protobuf and Go.
* Simple API versioning - protobuf is backwards compatible by design.
* Built-in streaming support; server-side, client-side and bi-directional.
* Fast and efficient marshalling and unmarshalling via protobuf.

If you want to take a look at an example of the GopherJS gRPC-Web bindings
in use, you can dive into my
[`grpcweb-example` repo](https://github.com/johanbrandhorst/grpcweb-example)
and take a look at the [demo website](https://grpcweb.jbrandhorst.com).

If you want to try it out for yourself, I would encourage you to clone
[the boilerplate repo](https://github.com/johanbrandhorst/grpcweb-boilerplate)
I set up to get going quickly.

I hope this post has inspired you to try something new next time you're
writing a webserver with a frontend client. If you have any questions
or comments, please reach out to me
[`@johanbrandhorst`](https://twitter.com/johanbrandhorst) or
`jbrandhorst` on Gophers Slack, and check out my
[my blog](https://jbrandhorst.com) for more stuff related to Go,
GopherJS and gRPC.
