+++
author = ["Pieter Louw"]
date = "2017-12-13T09:15:00Z"
series = ["Advent 2017"]
linktitle = "gRPC Go: Beyond the basics"
title = "gRPC Go: Beyond the basics"
+++

# Introduction

As a newcomer to gRPC (in Go) there are many resources setting out what gRPC is, where it originated from and how to create a basic service and client. After completing an introduction to gRPC and setting up a basic implementation I felt a bit lost as to where I need to go next.

gRPC consists of more than just sending binary blobs over HTTP/2. gRPC is also a set of libraries that will provide higher-level features consistently across platforms that other libraries typically do not. The purpose of this blog is to be a guideline for where to find the resources and leverage these libraries and features to make the most of the gRPC ecosystem after you understand the basics of gRPC.


Note:*To minimise bloat this article assumes knowledge of [gRPC](https://grpc.io) and [Protocol Buffers](https://developers.google.com/protocol-buffers/)*.

## Quick recap

Before we begin, let's refresh our memory and do a quick recap of what a gRPC service and client look like and how it's defined in a protobuf definition file (.proto).

We will create a service to query Go releases with 2 methods, `GetReleaseInfo` and `ListReleases`.

``` proto
service GoReleaseService {
    rpc GetReleaseInfo(GetReleaseInfoRequest) returns (ReleaseInfo) {}
    rpc ListReleases(ListReleasesRequest) returns (ListReleasesResponse) {}
}

message GetReleaseInfoRequest {
    string version = 1;
}

message ListReleasesRequest {} //empty

message ListReleasesResponse {
    repeated ReleaseInfo releases = 1;
}

message ReleaseInfo {
    string version = 1;
    string release_date = 2;
    string release_notes_url = 3;
}
```

Compiling this with the `protoc` tool with the [grpc plugin](https://github.com/golang/protobuf/protoc-gen-go) will create generated Go code to marshal/unmarshal the messages (i.e `GetReleaseInfoRequest`) between Go code and the protocol buffer binary messages. The gRPC plugin will also generate code to register and implement service interface handlers as well as code to create a gRPC client to connect to the service and send messages.

Let's look at a basic service and client implementation.

### Service

``` go
type releaseInfo struct {
    ReleaseDate     string `json:"release_date"`
    ReleaseNotesURL string `json:"release_notes_url"`
}

/* goReleaseService implements GoReleaseServiceServer as defined in the generated code:
// Server API for GoReleaseService service
type GoReleaseServiceServer interface {
    GetReleaseInfo(context.Context, *GetReleaseInfoRequest) (*ReleaseInfo, error)
    ListReleases(context.Context, *ListReleasesRequest) (*ListReleasesResponse, error)
}
*/
type goReleaseService struct {
    releases map[string]releaseInfo
}

func (g *goReleaseService) GetReleaseInfo(ctx context.Context,
                        r *pb.GetReleaseInfoRequest) (*pb.ReleaseInfo, error) {

    // lookup release info for version supplied in request
    ri, ok := g.releases[r.GetVersion()]
    if !ok {
        return nil, status.Errorf(codes.NotFound, 
            "release verions %s not found", r.GetVersion())
    }

    // success
    return &pb.ReleaseInfo{
        Version:         r.GetVersion(),
        ReleaseDate:     ri.ReleaseDate,
        ReleaseNotesUrl: ri.ReleaseNotesURL,
    }, nil
}

func (g *goReleaseService) ListReleases(ctx context.Context, r *pb.ListReleasesRequest) (*pb.ListReleasesResponse, error) {
    var releases []*pb.ReleaseInfo

    // build slice with all the available releases
    for k, v := range g.releases {
        ri := &pb.ReleaseInfo{
            Version:         k,
            ReleaseDate:     v.ReleaseDate,
            ReleaseNotesUrl: v.ReleaseNotesURL,
        }

        releases = append(releases, ri)
    }

    return &pb.ListReleasesResponse{
        Releases: releases,
    }, nil
}

func main() {
    // code redacted

    lis, err := net.Listen("tcp", *listenPort)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    log.Println("Listening on ", *listenPort)
    server := grpc.NewServer()

    pb.RegisterGoReleaseServiceServer(server, svc)

    if err := server.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}

```

### Client

``` go
func main() {
    flag.Parse()

    conn, err := grpc.Dial(*target, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("grpc.Dial err: %v", err)
    }

    client := pb.NewGoReleaseServiceClient(conn)

    ctx := context.Background()
    rsp, err := client.ListReleases(ctx, &pb.ListReleasesRequest{})

    if err != nil {
        log.Fatalf("ListReleases err: %v", err)
    }

    releases := rsp.GetReleases()
    if len(releases) > 0 {
        sort.Sort(byVersion(releases))

        fmt.Printf("Version\tRelease Date\tRelease Notes\n")
    } else {
        fmt.Println("No releases found")
    }

    for _, ri := range releases {
        fmt.Printf("%s\t%s\t%s\n",
            ri.GetVersion(),
            ri.GetReleaseDate(),
            ri.GetReleaseNotesURL())
    }
}
```

## Go gRPC API

After understanding the [why](https://grpc.io/faq/) and after doing an introduction on the [how](https://grpc.io/docs/tutorials/basic/go.html) of gRPC, the next step would be to familiarize yourself with the [official Go gRPC API](https://godoc.org/google.golang.org/grpc).

A client connection can be configured by supplying `DialOption` functional option values to the `grpc.Dial` function and server configuration is done by supplying `ServerOption` functional option values to the `grpc.NewServer` function.

*It's not necessary for this article to understand what functional options are, but to read more about functional options look [here - Dave Cheney](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis) and [here - Rob Pike](https://commandcenter.blogspot.co.za/2014/01/self-referential-functions-and-design.html)*.

The API also has a concept called *interceptors* which basically makes it possible to add middleware functionality to both Unary (single request/response) and Streaming calls.

Interceptors are very useful to wrap functionality around a RPC call. It helps to separate things like logging/auth/monitoring/tracing from the logic of the RPC service and can help to create a uniform way (for example : logging) for each call in one place.

Other functionality that the API offer are things like the handling of messages with a different codec than Protocol Buffers (i.e FlatBuffers), enabling compression of message, control maximum message sizes, add headers and trailers, enabling tracing and even creating load balancing functionality (*the Load Balancing API is still experimental*)

Find the full documentation of the API [here](https://godoc.org/google.golang.org/grpc).

To showcase the use of the API let's look at some use cases and build on top of our basic example above.

### Securing our service

If we look at the `grpc.NewServer` function definition (`func NewServer(opt ...ServerOption) *Server`) we will see that it is a [variadic function](https://blog.learngoprogramming.com/golang-variadic-funcs-how-to-patterns-369408f19085) that accepts a variable number of `grpc.ServerOption` values.

To enable TLS for our service we need to use the `grpc.Creds` function which returns a `grpc.ServerOption` to send to the `grpc.NewServer` function.

Let's look at the example.

Service code:

```go
creds := credentials.NewTLS(&tls.Config{
    // TLS config values here
})

serverOption := grpc.Creds(credentials.NewTLS(tlsConfig))
server := grpc.NewServer(serverOption)
```

The code to create a `tls.Config` is standard Go. The real lines of code that's of interest are the following:

``` go
serverOption := grpc.Creds(credentials.NewTLS(tlsConfig))
server := grpc.NewServer(serverOption)
```

The `credentials.NewTLS()` function construct a `credentials.TransportCredentials` value based on the `tls.Config` supplied.
The `grpc.Creds()` funtional option takes the `credentials.TransportCredentials` value and sets credentials for server connections.

Now the gRPC server will accept TLS connections.

Let's turn our focus to enabling TLS on the client side.

In the basic example we supply a `grpc.WithInsecure()` value to the `grpc.Dial` function. The `grpc.Insecure()` function returns a `DialOption` value which disables transport security for the client connection. By default, transport security is required so to disable transport security we need to set `WithInsecure`.
But we want to enable TLS transport security. This is done with the `grpc.WithTransportCredentials()` function. Just like the `grpc.Creds()` function we used to enable transport security on the server side, the `grpc.WithTransportCredentials()` function also accepts a `credentials.TransportCredentials` but the difference is it returns a `DialOption` value and not a `ServerOption` value, and `grpc.Dial` function accepts `DialOption` values.

Client code:

``` go
creds := credentials.NewTLS(&tls.Config{
    //TLS Config values here
})

conn, err := grpc.Dial(*target, grpc.WithTransportCredentials(creds))
if err != nil {
    log.Fatalf("grpc.Dial err: %v", err)
}
```

Now our service are enabled with TLS to encrypt data sent over the wire.

There are many other options like message and buffer sizes and specifying a custom codec (something other than Protocol Buffer) available.To see what other options are available the API docs are your friend.

For more server side options, see [ServerOption](https://godoc.org/google.golang.org/grpc#ServerOption).

For more client side options, see [DialOption](https://godoc.org/google.golang.org/grpc#DialOption).

**Tip:** `DialOption` values all have a **With** prefix, i.e grpc.**With**TransportCredentials

*For more resources on enabling TLS for gRPC in Go:*

[Using gRPC with Mutual TLS in Golang](http://krishicks.com/post/2016/11/01/using-grpc-with-mutual-tls-in-golang/)

[Go Secure Hello World Example](https://github.com/kelseyhightower/helloworld)

[Secure gRPC with TLS/SSL](https://bbengfort.github.io/programmer/2017/03/03/secure-grpc.html)

[Go gRPC Auth Support](https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-auth-support.md)

### Adding middleware

Like I said previously, the gRPC API has a concept called interceptors which enables us to write middleware functionality to our calls.
To illustrate the use of interceptors we will write interceptors to add logging and basic authentication to our calls.

Interceptors can be created for both client and servers, and both support interceptors for Unary RPC calls as well as Streaming calls.

To create an interceptor you would need to create a function with a definition that matches the relevant type of interceptor you want to create.
For example, if you want to create an Unary interceptor for your server, then based on the definitions below we would need to create a function with the same definition as `UnaryServerInterceptor` and supply that function to `grpc.UnaryInterceptor()` to create a `ServerOption` value that will be used to set the option for the server.

``` go
// DialOptions to set interceptors on the client side
func WithUnaryInterceptor(f UnaryClientInterceptor) DialOption
func WithStreamInterceptor(f StreamClientInterceptor) DialOption

// Client interceptor function definitions
type UnaryClientInterceptor func(ctx context.Context,
        method string, req, reply interface{},
        cc *ClientConn, invoker UnaryInvoker,
        opts ...CallOption) error
type StreamServerInterceptor func(srv interface{}, 
        ss ServerStream, 
        info *StreamServerInfo, 
        handler StreamHandler) error

// ServerOptions to set interceptors on the server side
func UnaryInterceptor(i UnaryServerInterceptor) ServerOption
func StreamInterceptor(i StreamServerInterceptor) ServerOption

// Server interceptor function defitions
type UnaryServerInterceptor func(ctx context.Context, 
        req interface{}, 
        info *UnaryServerInfo, 
        handler UnaryHandler) (resp interface{}, err error)
type StreamServerInterceptor func(srv interface{}, 
        ss ServerStream, 
        info *StreamServerInfo, 
        handler StreamHandler) error
```

The API documents every parameter but I want to quickly focus on how metadata is handled by interceptors.
Metadata can be accessed by each interceptor via the `context.Context` value (for Unary calls) and `ServerStream` value (for Streaming calls). This is useful if we need to access the metadata (i.e authorization details) set in the call to authorize a call for example.

Enough with the theory, let's implement our logging and authorization middleware.

Server:

``` go
// general unary interceptor function to handle auth per RPC call as well as logging
func unaryInterceptor(ctx context.Context, 
            req interface{}, 
            info *grpc.UnaryServerInfo, 
            handler grpc.UnaryHandler) (interface{}, error) {
    start := time.Now()

    //skip auth when ListReleases requested
    if info.FullMethod != "/proto.GoReleaseService/ListReleases" { 
        if err := authorize(ctx); err != nil {
            return nil, err
        }
    }

    h, err := handler(ctx, req)

    //logging
    log.Printf("request - Method:%s\tDuration:%s\tError:%v\n", 
        info.FullMethod, 
        time.Since(start), 
        err) 

    return h, err
}
```

This function will be called with each incoming request before the actual service method is called. We can add general logging and use the different parameter values that get passed in to make decisions of our own like using the `grpc.UnaryServerInfo` value to exclude authorization checks for certain requests or use the `context.Context` value to extract metadata to check authorization like this:

``` go
// code from the authorize() function:
md, ok := metadata.FromIncomingContext(ctx)
if !ok {
    return status.Errorf(codes.InvalidArgument, "retrieving metadata failed")
}

elem, ok := md["authorization"]
if !ok {
    return status.Errorf(codes.InvalidArgument, "no auth details supplied")
}
```

To enable the interceptor on the server we supply a `ServerOption` that will set the server's unary inceptor to our function called `unaryInterceptor` using the `grpc.UnaryInterceptor()` function.

``` go
// supply Transport credentials and UnaryInterceptor options
server := grpc.NewServer(
    grpc.Creds(credentials.NewTLS(tlsConfig)),
    grpc.UnaryInterceptor(unaryInterceptor)
)
```

On the client side we will need to send the authorization details with the call. To do this we supply a  `DialOption` to the `grpc.Dial` function using the `grpc.WithPerRPCCredentials()` functional option which expects a `credentials.PerRPCCredentials` value.

Below we have a struct type called `basicAuthCreds` which satisfy the `credentials.PerRPCCredentials` interface:

``` go
// basicAuthCreds is an implementation of credentials.PerRPCCredentials
// that transforms the username and password into a base64 encoded value similar
// to HTTP Basic xxx
type basicAuthCreds struct {
    username, password string
}

// GetRequestMetadata sets the value for "authorization" key
func (b *basicAuthCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
    return map[string]string{
        "authorization": "Basic " + basicAuth(b.username, b.password),
    }, nil
}

// RequireTransportSecurity should be true as even though the credentials are base64, we want to have it encrypted over the wire.
func (b *basicAuthCreds) RequireTransportSecurity() bool {
    return true
}

//helper function
func basicAuth(username, password string) string {
    auth := username + ":" + password
    return base64.StdEncoding.EncodeToString([]byte(auth))
}
```

We then create a value for `basicAuthCreds` and then supply it to the `grpc.WithPerRPCCredentials()` functional option :

``` go
grpcAuth := &basicAuthCreds{
    username: *username,
    password: *password,
}

conn, err := grpc.Dial(*target,
    grpc.WithTransportCredentials(creds),
    grpc.WithPerRPCCredentials(grpcAuth),
)
```

When the call happens the gRPC client will now generate the basic auth credentials and add it to the calls' metadata.

### Summary

We have reached the end of our overview of the Go gRPC API and shown what is possible.

In summary, we made our basic server more secure and added middleware without having to change code in our basic service. What's also nice is we can add more methods to our service which will automatically "inherit" the security and middleware functionality already created, we can just focus on the business logic required.

There are other behavior that can be changed via the API but will not go into detail now:

- [Set own logger implementation for underlying logger](https://godoc.org/google.golang.org/grpc/grpclog#SetLoggerV2)
- Enable tracing to trace RPCs using the golang.org/x/net/trace package (`grpc.EnableTracing = true`)
- Set own [backoff](https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md) configuration.

I encourage the reader to spend some time going over the API as well as it's [subdirectories](https://godoc.org/google.golang.org/grpc#pkg-subdirectories).

Hopefully this is enough information to leverage the API and create your own specific implementation.

## Go gRPC ecosystem

Let's move towards looking at what is available in the wider gRPC ecosystem that serves as an extention to the official API.

### go-grpc-middleware

If you recall in our example above all interceptor functionality (logging and auth) were contained in one interceptor. The API only allow one unary interceptor handler and one streaming interceptor handler for both client and servers.

This is where the go-grpc-middleware package come in very handy as it supplies functionality to chain interceptors into one interceptor:

``` go
streamingChain := grpc_middleware.ChainStreamServer(
    loggingStream,
    monitoringStream,
    authStream
)
unaryChain := grpc_middleware.ChainUnaryServer(
    loggingUnary,
    monitoringUnary,
    authUnary
)
myServer := grpc.NewServer(
    grpc.StreamInterceptor(streamingChain),
    grpc.UnaryInterceptor(unaryChain),
)
```

The gRPC Middleware package also have great ready-to-use interceptors for auth, logging (logrus, zap), monitoring (Prometheus ), tracing (OpenTracing), retries, server side validation etc.

For more info:

- [Github](https://github.com/grpc-ecosystem/go-grpc-middleware)
- [Godoc](https://godoc.org/github.com/grpc-ecosystem/go-grpc-middleware)

See also:

- [gRPC tracing on Stackdriver](https://rakyll.org/grpc-trace/)

### gRPC and the web

gRPC was mainly developed for services talking RPC with each other internally in a system. It also has great support for mobile clients talking to gRPC services but how does gRPC fit into existing web technologies?

#### grpc-gateway

gRPC Gateway is a great project if you already have gRPC services but your API need to be exposed as a traditional RESTful API.
It includes a plugin to the `protoc` tool which generates a reverse-proxy server which translates a RESTful JSON API into gRPC.
It also generates Swagger/API documentation.

For more info:

- [Blog by CoreOS on grpc-gateway](https://coreos.com/blog/grpc-protobufs-swagger.html)
- [Github](https://github.com/grpc-ecosystem/grpc-gateway)

##### grpc-websocket-proxy

gRPC WebSocket Proxy is a proxy to transparently upgrade grpc-gateway streaming endpoints to use websockets. It enables bidirectional streaming of JSON payloads on top of grpc-gateway.

For more info:

- [Github](https://github.com/tmc/grpc-websocket-proxy)
- [Godoc](https://godoc.org/github.com/tmc/grpc-websocket-proxy/wsproxy)

#### gRPC Web

gRPC has support for several languages but it's a common question as to where gRPC fit into the world of the web browser. It has support for server Javascript (Node.js), but what about client-side Javascript directly from the browser?

Enter [gRPC-Web](https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-WEB.md).

There's an official specification for gRPC-Web, but there's no official implementation, however Improbable created their own implementation based on the specification and is used in production at the moment. They also [open sourced their solution](https://github.com/improbable-eng/grpc-web/) which includes a client side implementation in Typescipt, `protoc` plugin and server side proxy in Go.

*The blog on [Day1 of the 2017 advent series](https://blog.gopheracademy.com/advent-2017/gopherjs-grpc-web/) have an excellent article on gRPC-Web and a great example to create a client in GopherJS*.

For more info:

- [Improbable blog about gRPC-Web](https://improbable.io/games/blog/grpc-web-moving-past-restjson-towards-type-safe-web-apis)
- ["Official" gRPC-Web repo](https://github.com/grpc/grpc-web) (Private repo)
- [Caddy gRPC plugin](https://caddyserver.com/docs/http.grpc) that also supports gRPC-Web proxying.
- [Vue.js example using gRPC-Web](https://github.com/b3ntly/vue-grpc)
- [Starter kit for Angular 2 projects using gRPC-Web](https://github.com/b3ntly/ng2-gRPC)

### Closing

Thank you for reading this blog. Hopefully this blog helped you into diving deeper into the wonderful world of gRPC in Go.
Although gRPC is considered a framework, the API gives us a flexible API to leverage and control behavior and to make our services robust and production ready.

If you have any feedback, remarks or questions you can send me a tweet @pieterlouw

The source code for the example can be found [here](https://github.com/pieterlouw/grpc-beyond).
