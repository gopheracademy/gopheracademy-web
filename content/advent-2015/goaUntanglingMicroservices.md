+++
author = ["Raphael Simon"]
date = "2015-12-07"
linktitle = "goa: Untangling Microservices"
series = ["Advent 2015"]
title = "goa: Untangling Microservices"

+++

# goa: Untangling microservices

## The Raise of Microservice Architectures and APIs

After suffering through a monolithic Rails application for a number of
years, we (the RightScale Engineering team) shifted our focus to
microservice architectures. As many others, we have encountered some of
their pitfalls as well. One of them is that building good APIs is
difficult. Changing APIs is even more difficult. And any APIs that get
exposed to customers are almost impossible to ever change, it seems. For
this reason we have focused on tools that help us design, review, and
implement the APIs of our microservices. One of the results of this
focus is [goa](http://goa.design), which we're just starting to use as
our HTTP microservice framework of choice.

## Introducing **goa**

There are already numerous good web application packages out there.
In fact we have been using [goji](https://goji.io/) at RightScale
successfully in production for some time. These packages focus on
providing modular and composable web application stacks which is great
for building independent services. However none of them help with the
critical task consisting of designing an API.
<div style="float: right;
    height: 0;
    padding-top: 25%;
    width: 40%;
    background-image: url(https://rawgit.com/raphael/goa/gh-pages/images/DRII.svg);
    background-repeat: no-repeat;
    background-size: contain;">
</div>

Building an API is a multi-step process. The API first gets designed,
resources and associated actions are identified, the request endpoints,
payloads and parameters all get defined. Once that's done the design
goes trough a review process: will the UI another team has to build on
top have all the information it requires? will dependent service X be
able to list the resources it needs efficiently? will dependent service
Y be able to update the fields of this other resource? After a few back
and forth it's time to actually implement the API. And after a while
it's back to square 1 with new requirements for APIv2.

The review process is especially hard to do with no special tooling.
Who is going to write a [Swagger](http://swagger.io) specification from
scratch just to throw it away as soon as implementation starts? However
it's also a critical step for the overall success of the service.
Without a clear and complete description of the API there's a good
chance that something will end up not quite right or missing entirely.

That's where [goa](http://goa.design) comes in. goa lets you write the
specification of your API in code. It then uses that code to produce
a number of outputs including HTTP handlers that take care of validating
the incoming requests. This means that **the specification is translated
automatically into the implementation, what got reviewed is what is
implemented**.

The final implementation, however, is very familiar looking and can
plug-in to many existing HTTP processing packages. HTTP requests are
accepted by the `net/http` server, routed by a router (goa uses
[httprouter](https://github.com/julienschmidt/httprouter)) and handled
by the application code. The only difference being that the application
code is composed of two parts: the generated handler which validates the
request and creates the context object (more on that later) and the user
code that provides the business logic.
<div style="height: 0;
 padding-top: 25%;
 width: 100%;
 background-image: url(https://rawgit.com/raphael/goa/gh-pages/images/Routing.svg);
 background-repeat: no-repeat;
 background-size: contain;">
</div>

## The goa Design Language

At first I wasn’t sure whether creating a DSL to describe an API design
in Go would even be possible or yield something that is usable. goa
started as an experiment but after many iterations of various degrees
of ugliness the end result is actually quite nice. Credits go to
[Gomega](https://onsi.github.io/gomega/) for showing how using anonymous
functions can help produce a clean and terse DSL.

Let's go through a simple example to illustrate how it works. Imagine
an API service that manages bottles of wine, let's call it `winecellar`.
This service exposes one endpoint that makes it possible to retrieve
information on a wine bottle given its ID. First we define the API
itself using the `API` global DSL function. This function accepts a name
and an anonymous function that can define additional properties such as
the base path for all requests, the supported URL schemes, the host as
well as metadata like information (description, contact, license etc.):
```go
package design

import (
        . "github.com/raphael/goa/design" // "dot" imports make the DSL easier to read.
        . "github.com/raphael/goa/design/dsl"
)

var _ = API("winecellar", func() { // The API function defines an API given its name.
        Description("The winecellar service API")
        BasePath("/cellar")        // Base path or prefix to all requests.
                                   // Can be overridden in action definitions using an
                                   // absolute path starting with //.
        Host("cellar.goa.design")  // Default API host used by clients and Swagger.
        Scheme("http")             // Supported API URL scheme used by clients and Swagger.
        Scheme("https")            // Scheme("http", "https") works too
})
```
Note that the name of the package is irrelevant, we use `design` as a
convention.

Now that we have defined our API we need to define the `show bottle`
request endpoint. To do that we first need to define a resource
(`Bottle`) and in the definition of the resource add the `show` action
that exposes that one endpoint:
```go
var _ = Resource("Bottle", func() { // Define the Bottle resource
        DefaultMedia(BottleMedia)   // Default media type used to render the bottle resources
        BasePath("/bottles")        // Gets appended to the API base path

        Action("show", func() {              // Define the show action on the Bottle resource
                Routing(GET("/:bottleID"))   // The relative path to the show endpoint. The full path is
                                             // built concatenating the API and resource base paths with it.
                                             // Uses a wildcard to capture the requested bottle ID.
                                             // Wildcards can start with : to capture a single path segment
                                             // or with * to capture the rest of the path.
                Description("Retrieve bottle with given ID")
                Params(func() {              // Define the request parameters found in the URI (wildcards)
                                             // and the query string.
                        Param(               // Define a single parameter
                                "bottleID",  // Here it corresponds to the path segment captured by :bottleID
                                Integer,     // The JSON type of the parameter
                                "The name of the bottle to retrieve", // An optional description
                        )
                })
                Response(OK)                 // Define a potential response
                Response(NotFound)           // An action may define any number of responses.
                                             // Their content is defined through ResponseTemplates (not shown
                                             // in this simplistic example). Here we use the default response
                                             // templates defined in goa.
        })
```
A resource may specify a default media type used to render `OK`
responses. In goa the media type describes the data structure
rendered in the response body.  In the example the `Bottle` resource
refers to the `BottleMedia` media type. Here is the definition for it:
```go
var BottleMedia = MediaType("application/vnd.goa.example.bottle+json", func() {
        Description("A bottle of wine")
        Attributes(func() {
                Attribute("id", Integer, "ID of bottle") // Attribute defines a single field in
                                                         // the media type data structure given its
                                                         // name, type and description.
                Attribute("href", "API href of bottle")  // The default type for attributes is String.
                Attribute("name", "The bottle  name", func() { // Like with API, Resource and Action an attribute
                                                         // definition may use an anonymous function as
                                                         // last argument to define additional properties.
                        MinLength(1)                     // Here we define validation rules specifying a
                        MaxLength(255)                   // minimum and maximum number of characters in a bottle
                        // name.
                })
                Attribute("color", func() {              // Descriptions are optional.
                        Enum("red", "white", "rose", "yellow", "sparkling") // Possible field values
                })
                Attribute("sweetness", Integer, func() {
                        Minimum(1)                       // Minimum and maximum int field values.
                        Maximum(5)
                })

                View("default", func() {                 // Views are used to render a media type.
                        Attribute("id")                  // A media type can have one or more views
                        Attribute("href")                // and must define the "default" view.
                        Attribute("name")                // The view simply lists the fields to render.
                        Attribute("color")               // It can also specify the view to use to render
                        Attribute("sweetness")           // fields whose type is itself a media type
                                                         // (the "default" view by default). Not used here.
                })
        })
})
```
We now have a complete description of our API together with its
endpoint, the accepted request parameters and the details on the
response content. In case you are wondering request payloads (for
request that have bodies) are defined using the same DSL used to
define media types (minus views). There are a few more advanced
constructs supported by the DSL such as the ability to link to other
media types or reuse types in multiple definitions. The dsl package
[GoDoc](https://godoc.org/github.com/raphael/goa/design/dsl) lists all
the supported keywords with additional examples.

Now that we have written down the design of our API it can be reviewed.
While reviewers may be able to read the DSL spec straight we can make
their task more attractive by automatically generating browsable
documentation. As explained further down, goa can generate the
[Swagger](http://swagger.io) specification and the standard Swagger UI
can be used to view them.

## The Magic: Code Generation

The purpose of specifying the API using a DSL is to make it executable.
In Go the preferred method for this is to generate code and this is the
path goa takes. goa comes with the [goagen](http://goa.design/goagen.html)
tool which is the goa code generator. The processing of the design
occurs in the following stages:

1. `goagen` parses the command line to determine the type of output
   desired and invokes the appropriate generator.
2. The generator writes the code that will produce the final output
   to a temporary Go workspace.
3. The DSL is compiled together with the output producing code in the
   temporary workspace.
4. The resulting tool executes evaluating and validating the DSL.
   The result of evaluating the DSL are simple data structures that
   describe the API. The output producing code traverses these data
   structures in memory and writes the corresponding output.

`goagen` supports many different outputs. Each output maps to a tool
command. The following commands are currently supported:

* `app`: generates the service boilerplate code including controllers,
  contexts, media types and user types.
* `main`: generates a skeleton file for each resource controller as well
  as a default `main` implementation.
* `client`: generates an API client Go package and tool.
* `js`: generates a JavaScript API client based on [axios](https://github.com/mzabriskie/axios).
* `swagger`: generates the API [Swagger](http://swagger.io) specification.
* `schema`: generates the API [Hyper-schema](http://json-schema.org/latest/json-schema-hypermedia.html) JSON.
* `gen`: invokes a third party generator package.
* `bootstrap`: invokes the `app`, `main`, `client` and `swagger`
   commands.

The data structures produced in step 3 above from executing the DSL
describe the resources that make up the API and for each resource the
actions complete with a description of their parameters, payload and
responses. Note that the term *resource* here is very loosely defined.
Resources in goa merely provide a convenient way to group API endpoints
(called *actions* in the DSL) together. The actual semantic is
irrelevant to goa - in other words goa is not an opinionated framework
by design.

### Glue Code

The `app` output deserves special attention as it generates the glue
code between the underlying HTTP server and the controller (your) code.
The code takes care of validating the incoming requests and coercing the
types to the ones described in the design. This in turns means that the
controller code does not have to worry about deserializing and "binding"
the request body for example. It also means that all the validation
rules specified in the design have already been executed so that the
value of parameters for example don't need to be validated by your
code. The end result is controller code that is terse and only deals
with what matters: your special sauce.

Here is a code snippet to illustrate the above, this code implements
the `show` action of a the `Bottle` resource defined in the previous
example. The function signature was generated by `goagen` and the
default implementation (which simply writes an empty response) replaced
with actual code:
```go
// Retrieve bottle with given ID.
func (b *BottleController) Show(ctx *app.ShowBottleContext) error { // The signature was generated by `goagen main`.
	bottle := b.db.GetBottle(ctx.BottleID) // This example stores the database driver in a controller field.
	if bottle == nil {                     // (the same controller instance handles all requests)
		return ctx.NotFound()          // NotFound has been generated from the corresponding Response
	}                                      // definition in the DSL.
	return ctx.OK(bottle)                  // So was OK. The default OK response template that comes with goa
                                               // defines the media type of the response payload using the resource
                                               // default media type.
```
As you can see the code has access to the request state (`BottleID` here)
via fields exposed by the context. The values of the fields have been
validated by goa and their types match the types used in the design
(here `BottleID` is an int). The context also exposes the `NotFound` and
`OK` methods used to write the response. Again these methods exist
because the design specified that these were the responses of this
action. The design also defines the response payload so that in this
case the `OK` method accepts an instance of `app.Bottle` which is a
type that was generated from the `BottleMedia` definition.
The [cellar](https://github.com/raphael/goa/blob/master/examples/cellar)
example contains implementations for many more actions.

### Documentation

Another very valuable output is documentation in the form of
[JSON schema](http://json-schema.org/latest/json-schema-hypermedia.html)
or [swagger](http://swagger.io). Being able to look at the documentation
makes it a lot easier and more efficient to vet the API design without
having to write a single line of actual implementation code.

<div style="float: right;
 height: 0;
 padding-top: 45%;
 width: 60%;
 background-image: url(https://rawgit.com/raphael/goa/gh-pages/images/goa-swagger.png);
 background-repeat: no-repeat;
 background-size: contain;">
</div>

This screen shot shows documentation that was produced automatically via
the free [swagger.goa.design](http://swagger.goa.design) service: I
placed my design in a public github repository and then pointed
swagger.goa.design at it. It then downloaded the repo from github,
produced the swagger specification and loaded it in
[swagger UI](https://github.com/swagger-api/swagger-ui).

<div style="clear: both;">
</div>

### Clients

One of the nice side-effects of having a complete spec of the API is
that goa can produce not only server-side code to implement the API but
also client-side code to make it easier to invoke the API. In its
current form, the `goagen` tool can generate three types of clients:

1. a Go package for clients written in Go
2. a Javascript package for clients running in node.js or the browser
3. a command-line tool to invoke the API from the linux or windows
   command line

Going back to the problem statement: how to deal with an exponentially
growing number of interconnected microservices - this is huge. It means
that the team in charge of developing a given microservice can also
easily deliver the clients. This in turn means that the same clients are
reused throughout which helps with consistency, troubleshooting etc.
Things like enforcing the [X-Request-ID](https://devcenter.heroku.com/articles/http-request-id)
header, CSRF or CORS which would otherwise be a tedious manual endeavor
now become easily achievable. Here is an example of the command line
help of a generated client:
```
./cellar-cli --help-long
usage: cellar-cli [<flags>] <command> [<args> ...]

CLI client for the cellar service (http://goa.design/getting-started.html)

Flags:
     --help           Show context-sensitive help (also try --help-long and
		      --help-man).
 -s, --scheme="http"  Set the requests scheme
 -h, --host="cellar.goa.design"  
		      API hostname
 -t, --timeout=20s    Set the request timeout, defaults to 20s
     --dump           Dump HTTP request and response.
     --pp             Pretty print response body

Commands:
 help [<command>...]
   Show help.


 create bottle [<flags>] <path>
   Record new bottle

   --payload=PAYLOAD  Request JSON body

 delete bottle <path>

 list bottle [<flags>] <path>
   List all bottles in account optionally filtering by year

   --years=YEARS  Filter by years

 show bottle <path>
   Retrieve bottle with given id
```
The tool also provides contextual help for each action:
```
./cellar-cli show bottle --help
usage: cellar-cli show bottle <path>

Retrieve bottle with given id

Flags:
      --help           Show context-sensitive help (also try --help-long and
                       --help-man).
  -s, --scheme="http"  Set the requests scheme
  -h, --host="cellar.goa.design"  
                       API hostname
  -t, --timeout=20s    Set the request timeout, defaults to 20s
      --dump           Dump HTTP request and response.
      --pp             Pretty print response body

Args:
  <path>  Request path, format is /cellar/accounts/:accountID/bottles/:bottleID
```
The implementation of the client tool relies on the generated client
Go package to make the actual requests. The package exposes a function
for each action exposed by the API, see the files generated for the
cellar example [clients](https://github.com/raphael/goa/blob/master/examples/cellar/client)
for more details.

The other client `goagen` can produce is a JavaScript module.
Similarly to the Go package the JavaScript module exposes one function
per API action. It uses the [axios](https://github.com/mzabriskie/axios)
library to make the actual requests. Again the cellar example contains
the [generated JavaScript](https://github.com/raphael/goa/blob/master/examples/cellar/js/client.js)
together with [an example](https://github.com/raphael/goa/blob/master/examples/cellar/js/index.html)
on how to use it if you are curious.

### Plugins

Last but not least `goagen` makes it very easy to implement
custom generators through *goagen plugins*. A plugin is merely a Go
package which exposes a `Generate` function with the following signature:
```go
func Generate(api *design.APIDefinition) ([]string, error)
```
where `api` is the API definition computed from the DSL. On success
`Generate` should return the path to the generated files. On failure the
error message gets displayed to the user (and `goagen` exits with status 1).

Any package exposing this function can then be used by goagen simply by
providing its path on the command line, for example:
```
goagen gen -d github.com/raphael/goa/examples/cellar/design --pkg-path=github.com/bketelsen/gorma
```
would use @bketelsen `gorma` plugin over the goa `cellar` example.

### Final Overview

Summing it all up, the diagram below shows all the various outputs of
the `goagen` tool:
![goagen diagram](https://cdn.rawgit.com/raphael/goa/gh-pages/images/goagen.svg "goagen")

## The Engine: Runtime

goa is not just about code generation though. It also includes a set
of functionality to support the execution of the web application. The
goal is to provide a production ready runtime environment that helps
dealing with the challenges of running services in a microservice
environment. This includes structured logging, X-Request-ID header
support, proper panic recovery and many other features described below.

### The Request Context

goa provides a powerful [Context](https://godoc.org/github.com/raphael/goa#Context)
object to all request handlers. This object makes it possible to carry
deadlines and cancellation signals, gives access to the request and
response state and allows writing log entries.

The goa Context interface implements the [golang context.Context](https://godoc.org/golang.org/x/net/context)
interface which provides a concurrency safe way of storing and
retrieving values on top of the deadline and cancelation support
described above. The idea is that the context can be passed around to
all the various service sub-systems (e.g. persistence layer or external
service interfaces) which can all read or update it as see fit. The
context implementation takes care of updating the deadline properly so
that setting a longer timeout down the chain doesn't override the
previously set short timeout for example. The
[golang ctxhttp](https://godoc.org/golang.org/x/net/context/ctxhttp)
(repeat that quickly 5 times) provides a context-aware HTTP client that
will honor timeouts and abort requests when a cancelation signal
triggers. Having access to the context in all the sub-systems also
means that the entire request state is available to them. This can be
very handy and helps with decoupling the application layers.

### Logging

goa supports structured logging via the excellent
[log15](https://godoc.org/gopkg.in/inconshreveable/log15.v2) package.
The context object is also a logger and exposes logger methods (`Info`
`Warn`, `Err` and `Crit`). Each log entry has a message and a series of
name/value pairs. goa pre-populates the key/value pairs with the name
of the service, controller and action as well as a request specific ID
so that any call to one of the logger methods will tag the log entry
with these values. Obviously additional values can be stored in the
logger context. Sub-systems may also instantiate their own logger
inheriting the parent logger context (and handler see below).

The logger is backed by handlers which do the actual writing. `log15`
comes with a bunch of handlers that can write to syslog, loggly etc.
The default goa handler writes to `Stdout` which is handy for dev or for
services running in containers (depending on your logging strategy). The
service logger handler can be initialized prior to starting it via the
[Logger](https://godoc.org/github.com/raphael/goa#pkg-variables)
package variable:
```go
func main() {
	// Create goa service
	service := goa.New("cellar")

        // Initialize logger to use syslog.
        syslogHandler := log15.SyslogHandler("cellar", log15.LogfmtFormat())
        goa.Logger.SetHandler(syslogHandler)

        // ...
```

### Middleware

goa supports both "classic" `net/http` [middleware](https://justinas.org/writing-http-middleware-in-go/)
as well as goa specific middleware that can leverage the context object.
As a simple example here is the source for the `RequestID` middleware
that handles the [X-Request-ID](https://devcenter.heroku.com/articles/http-request-id)
header:
```go
// RequestID is a middleware that injects a request ID into the context of each request.
// Retrieve it using ctx.Value(ReqIDKey). If the incoming request has a RequestIDHeader header then
// that value is used else a random value is generated.
func RequestID() Middleware {
	return func(h Handler) Handler {
		return func(ctx *Context) error {
			id := ctx.Request().Header.Get(RequestIDHeader)
			if id == "" {
				id = fmt.Sprintf("%s-%d", reqPrefix, atomic.AddInt64(&reqID, 1))
			}
			ctx.SetValue(ReqIDKey, id)

			return h(ctx)
		}
	}
}
```
The middleware takes and returns a request handler, it uses closure to
wrap the handler passed as argument and add its own logic.

goa currently includes the following middleware:

* A `LogRequest` middleware that logs the request and responses.
* The RequestID middleware shown above.
* A `Recover` middleware that recovers and logs panics.
* a `Timeout` middleware that sends a cancelation signal throught the
  context after a given amount of time.
* a `RequireHeader` middleware that checks that a given header has a
  given value (useful to implement shared secret auth).
* a `CORS` middleware which provides a simple DSL for configuring [CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/Access_control_CORS).

Obviously you can mount your own middleware through the `Use` method.
This method is implemented by both the [Service](https://godoc.org/github.com/raphael/goa#Service)
and the [Controller](https://godoc.org/github.com/raphael/goa#Controller)
interfaces. This means that middleware can be applied to all requests sent
to the service or only to the endpoints exposed by specific controllers.

### Error Handling

In goa request handlers can return errors (instances of `error`). When
they do goa checks whether the controller has a error handler and if it
does invokes it. If the controller does not have a error handler then
goa invokes the service-wide error handler.

The [default](https://godoc.org/github.com/raphael/goa#DefaultErrorHandler)
service-wide error handler simply logs the error and returns a response
with status code `400` if the error is an instance of
`goa.BadRequestError` - `500` otherwise. The default error handler also
writes the message of the error to the response body. goa also comes
with a [terse](https://godoc.org/github.com/raphael/goa#TerseErrorHandler)
error handler which won't write the error message to the body for
internal errors (useful for production).

As with middleware error handlers can be mounted on a specific controller
or service-wide via the `SetErrorHandler` method exposed by both the
[Service](https://godoc.org/github.com/raphael/goa#Service)
and the [Controller](https://godoc.org/github.com/raphael/goa#Controller)
interfaces.

### Graceful Shutdown

A goa service can be instantiated calling either the [New](https://godoc.org/github.com/raphael/goa#New)
or [NewGraceful](https://godoc.org/github.com/raphael/goa#NewGraceful) package
functions. Calling `NewGraceful` returns a server backed by the
[graceful](https://godoc.org/github.com/tylerb/graceful) package Server
type. When sending any of the signals listed in the goa
[InterruptSignals](https://godoc.org/github.com/raphael/goa#pkg-variables)
package variable to the process the graceful server:

* disables keepalive connections.
* closes the listening socket, allowing another process to listen on that port immediately.
* sends a cancellation signal through the context.

## What's Next?

goa is still very new and while I’m quite excited about its potential,
the proof is in the pudding. There are a number of goa services that are
slated to go to production at RightScale in the near future and I’m sure
that new "interesting" challenges will come out. At the same time goa
seems to solve many problems and so far the adoption has been very
positive. I have been amazed at how quickly the open source community
started contributing back (special thanks to @bketelsen for spreading
the good word) and can’t wait to see what others do with goa.
But that’s enough talk, [try it for yourself](https://github.com/raphael/goa) and let
[me](mailto:raphael@goa.design) know how it goes!
