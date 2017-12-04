+++
author = ["Pieter Louw"]
date = "2017-12-13"
series = ["Advent 2017"]
linktitle = "gRPC Go: Beyond the basics"
title = "gRPC Go: Beyond the basics"

+++

# Introduction

Starting with [gRPC (in Go)](https://grpc.io/docs/quickstart/) there are many resources about what it is and where it comes from. After I finished an introduction to gRPC and implemented a basic service I felt a bit lost as to where I need to go next to find out how I can make the calls more secure or set different values to options.

gRPC is more than just sending binary blobs over HTTP/2. gRPC is also a set of libraries that will provide higher-level features consistently across platforms that common HTTP libraries typically do not. The purpose of this article is to be a guideline of where to look after you understand the basics of gRPC and leverage these libraries and features to make the most of the gRPC ecosystem.

Note:*To minimise bloat this article assumes knowledge of [gRPC](https://grpc.io) and [Protocol Buffers](https://developers.google.com/protocol-buffers/) and how to create a gRPC service and client in Go*.

## Quick recap

Before we begin, let's refresh our memory and do a quick recap of what a gRPC service and client look like and how it's defined in a protobuf definition file (.proto).

We will create a basic user service with 2 methods, `Register` and `Login`.

``` proto
syntax = "proto3";

package proto;

service UsersService {
    rpc Register(RegisterRequest) returns (Empty) {}
    rpc Login(LoginRequest) returns (LoginResponse) {}
}

message RegisterRequest {
    string username = 1;
    string password = 2;
    string email = 3;
}

message Empty {}

message LoginRequest {
    string username = 1;
    string password = 2;
}

message LoginResponse {
    string sessionid = 1;
}
```

`Register` accepts a `RegisterRequest` message and returns an `Empty` message.
`Login` accepts  a `LoginRequest` message and returns a `LoginResponse` message.

Compiling this with the `protoc` tool with the [grpc plugin](https://github.com/golang/protobuf/protoc-gen-go) will create generated Go code to marshal/unmarshal the messages (i.e `RegisterRequest`) between Go code and the protocol buffer binary messages. The gRPC plugin will also generate code to register and implement service interface handlers as well as code to create a gRPC client to connect to the service and send messages.

Let's look at a basic service and client implementation.

### Service

``` go
    /* usersService implements UsersServiceServer as defined in the generated code:
type UsersServiceServer interface {
	Register(context.Context, *RegisterRequest) (*Empty, error)
	Login(context.Context, *LoginRequest) (*LoginResponse, error)
}
*/
type usersService struct{}

func (u *usersService) Register(ctx context.Context, r *userspb.RegisterRequest) (*userspb.Empty, error) {

	// code to do validation here i.e, check if user already exists
	// return nil, status.Errorf(codes.AlreadyExists, "user %s already exists", r.GetUsername())

	// code to add user to database

	// success
	return &userspb.Empty{}, nil
}

func (u *usersService) Login(ctx context.Context, r *userspb.LoginRequest) (*userspb.LoginResponse, error) {

	// code to validate user and credentials

	// create a session

	return &userspb.LoginResponse{
		Sessionid: "this-will-be-the-unique-session-id",
	}, nil
}

func main() {
	flag.Parse()
	svc := &usersService{}

	lis, err := net.Listen("tcp", *listenPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Println("Listening on ", *listenPort)
	server := grpc.NewServer()

	userspb.RegisterUsersServiceServer(server, svc)

	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

```

### Login Client

``` go
func main() {

	flag.Parse()

	conn, err := grpc.Dial(*target, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("grpc.Dial err: %v", err)
	}

	usersClient := userspb.NewUsersServiceClient(conn)

	ctx := context.Background()
	_, err = usersClient.Login(ctx, &userspb.LoginRequest{
		Username: "scaramoucheX2",
		Password: "Can-you-do-the-fandango?",
	})

	if err != nil {
		log.Fatalf("usersClient.Login err: %v", err)
	}
}
```

## Go gRPC API

After understanding the [why](https://grpc.io/faq/) and an introduction on the [how](https://grpc.io/docs/tutorials/basic/go.html) of gRPC the next step would be to familiarize yourself with the [official Go gRPC API](https://godoc.org/google.golang.org/grpc).

The Go gRPC API uses functional options to control behaviour of connections, gRPC clients and gRPC servers.

*It's not necessary for this article to understand what functional options are, but to read more about functional options look [here - Dave Cheney](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis) and [here - Rob Pike].(https://commandcenter.blogspot.co.za/2014/01/self-referential-functions-and-design.html)*

A client connection can be configured by supplying `DialOption` values to the `grpc.Dial` function and server configuration is done by supplying `ServerOption` values to the `grpc.NewServer` function.

The API also has a concept called *interceptors* which basically makes it possible to add middleware functionality to both Unary (single request/response) and Streaming calls.

Interceptors are very useful to wrap functionality around a RPC call. It helps to seperate things like logging/auth from the logic of the RPC service and can help to create a uniform way for something like logging for each call in one place.

*link to article explaining http.Handler chaining to create middleware*

Other functionality that the API offer are things like handling of messages with a different codec than Protocol Buffers (i.e FlatBuffers), enabling compression of message, control maximum message sizes, add headers and trailers, enabling tracing and even creating load balancing functionality (*the Load Balancing API is still experimental*)

Find the full documentation of the API [here](https://godoc.org/google.golang.org/grpc)

To showcase the use of the API let's build on top of our basic example above.

### Securing our service

If we look at the `grpc.Server` definition (`func NewServer(opt ...ServerOption) *Server`) we will see that it is a [variadic function](https://blog.learngoprogramming.com/golang-variadic-funcs-how-to-patterns-369408f19085) that accepts a variable number of `grpc.ServerOption` values.

To enable TLS for our service we will use the `grpc.Creds` function which returns a `grpc.ServerOption` to send to the `grpc.NewServer`.
Let's look at the example.

Server code:

```go
    creds := credentials.NewTLS(&tls.Config{
        // TLS config values here
    })

    s := grpc.NewServer(grpc.Creds(creds))

    serverOption := grpc.Creds(credentials.NewTLS(tlsConfig))
    server := grpc.NewServer(serverOption)

    userspb.RegisterUsersServiceServer(server, svc)

    if err := server.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
```

The code to create a `tls.Config` is standard Go. The real lines of code that's of interest are the following:

``` go
    serverOption := grpc.Creds(credentials.NewTLS(tlsConfig))
    server := grpc.NewServer(serverOption)
```

Now the gRPC server will accept TLS connections.

Let's turn out focus to enabling TLS on the client side.

In the basic example we supply a `grpc.WithInsecure()` value to the `grpc.Dial` function. The `grpc.Insecure()` function returns a `DialOption` value which disables transport security for the client connection. By default, transport security is required so to disable transport security we need to set `WithInsecure`.
But we want to enable TLS transport security. This is done with the `grpc.WithTransportCredentials()` function. Just like the `grpc.Creds()` we used to enable transport security for the server `grpc.WithTransportCredentials()` also accepts a `credentials.TransportCredentials` but the difference is it returns a `DialOption` value and not a `ServerOption` value, and `grpc.Dial` function accepts `DialOption` values.

``` go
func main() {

    flag.Parse()

    creds := credentials.NewTLS(&tls.Config{
        //TLS Config values here
    })

    conn, err := grpc.Dial(*target, grpc.WithTransportCredentials(creds))
    if err != nil {
        log.Fatalf("grpc.Dial err: %v", err)
    }

    usersClient := userspb.NewUsersServiceClient(conn)

    ctx := context.Background()
    _, err = usersClient.Login(ctx, &userspb.LoginRequest{
        Username: "scaramoucheX2",
        Password: "Can-you-do-the-fandango?",
    })

    if err != nil {
        log.Fatalf("usersClient.Login err: %v", err)
    }
}
```

There are many other options like message and buffer sizes and specifying a custom codec (something other than Protocol Buffer) available.T o see what other options are available the API docs are your friend.
For more server side options , see [ServerOption](https://godoc.org/google.golang.org/grpc#ServerOption)
For more client side options, see [DialOption](https://godoc.org/google.golang.org/grpc#DialOption)

*Tip:* `DialOption` values all have a **With** prefix, i.e grpc.**With**TransportCredentials

*More reading on TLS and authentication:*

[Using gRPC with Mutual TLS in Golang](http://krishicks.com/post/2016/11/01/using-grpc-with-mutual-tls-in-golang/)

[Go Secure Hello World Example](https://github.com/kelseyhightower/helloworld)

[Go gRPC Auth Support](https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-auth-support.md)

## Adding middleware

Like I said previously, the gRPC API has a concept called interceptors which enables us to write middleware functionality to our calls.
To illustrate the use  of interceptors we will write a interceptors to add logging and basic authentication to our calls.

Interceptors can be created for both client and servers, and both support interceptors for Unary RPC calls as well as Streaming calls.
To create an interceptor you would need to create a function with a definition that matches the relevant type of interceptor you want to create.

``` go
// DialOptions to set interceptors on the client side
func WithUnaryInterceptor(f UnaryClientInterceptor) DialOption
func WithStreamInterceptor(f StreamClientInterceptor) DialOption

// Client interceptor function definitions
type UnaryClientInterceptor func(ctx context.Context, method string, req, reply interface{}, cc *ClientConn, invoker UnaryInvoker, opts ...CallOption) error
type StreamServerInterceptor func(srv interface{}, ss ServerStream, info *StreamServerInfo, handler StreamHandler) error

// ServerOptions to set interceptors on the server side
func UnaryInterceptor(i UnaryServerInterceptor) ServerOption
func StreamInterceptor(i StreamServerInterceptor) ServerOption

// Server interceptor function defitions
type UnaryServerInterceptor func(ctx context.Context, req interface{}, info *UnaryServerInfo, handler UnaryHandler) (resp interface{}, err error)
type StreamServerInterceptor func(srv interface{}, ss ServerStream, info *StreamServerInfo, handler StreamHandler) error
```

The API document every parameter but I want to quickly focus on how metadata is handled by interceptors.
Metadata can be accessed by each interceptor via the `context.Context` value (for Unary calls) and `ServerStream` value (for Streaming calls). This is useful if we need to access the metadata (i.e authorization details) set in the call to authorize a call for example.

Enough with the theory, let's implement our logging middleware (client and server) as well as our authorization middleware (server).

Server:

``` go
func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    log.Printf("") //logging 

    if err := authorize(ctx); err != nil {
        return err
    }

    return handler(ctx, req)
}

// authorize function that is used by the interceptor functions.
// warning: this is only for illustration purposes - don't implement authorization that is hardcoded!
func authorize(ctx context.Context) error {
    if md, ok := metadata.FromContext(ctx); ok {
        if len(md["username"]) > 0 && md["username"][0] == "scaramoucheX2" &&
            len(md["password"]) > 0 && md["password"][0] == "Can-you-do-the-fandango?" { 
            return nil
        }

        return status.Errorf(codes.Unauthenticated, "auth failed")
    }

    return status.Errorf(codes.InvalidArgument, "no auth details supplied")
}

func main() {
    // gRPC server setup code omitted to keep example code small


    // supply Transport credentials and UnaryInterceptor options
    server := grpc.NewServer(
        grpc.Creds(credentials.NewTLS(tlsConfig)),
        grpc.UnaryInterceptor(unaryInterceptor)
    )
}
```















## Go gRPC ecosystem

Let's step outside the official API and move towards looking at what is available in the wider gRPC ecosystem.

### go-proxy

### go-middleware

### gRPC and the web

##### grpc-gateway
https://coreos.com/blog/grpc-protobufs-swagger.html
##### grpc-web sockets

#### gRPC Web
- GopherJS article
- Improbable
- Caddy plugin




#### Other

- gRPC tracing on Stackdriver (https://rakyll.org/grpc-trace/)
- Protobuf tips (https://jbrandhorst.com/post/go-protobuf-tips/)
- Basic Auth: https://github.com/grpc/grpc-go/issues/106

    - gogo protobuf
    - grpc web
    - grpc gateway
    - gopherjs
    - vue and angular examples



### Conclusion

- Although gRPC is considered a framework the API gives us a flexible API to leverage and control behavior.
- gRPC is more than just another request/response architecture
- 

### Resources
