+++
author = ["Raphael Simon"]
date = "2015-12-01T11:58:00-08:00"
linktitle = "goa: Untangling Microservices"
series = ["Advent 2015"]
title = "goa: Untangling Microservices"

+++

# goa: Untangling microservices

## The Downfall of Monolithic Software

When I started working at [RightScale](https://www.rightscale.com),
about 7 years ago, Ruby on Rails was all the rage. Ruby was such a
refreshing language coming from a "classic" C++ background. You could
develop new features at the speed of light, “monkey patch” your way
around bugs, look at the source code of all the dependencies, debug live
production systems, heck even run `dbg` to debug ruby itself. Rails
provided a nice framework taking care of all the plumbing and letting us
focus on what matters: our special sauce.

Life was good... for a while. About 3 to 4 years and half a million
lines of code later the picture started to change: new features were
becoming increasingly difficult to implement as the web of software
interdependencies became harder and harder to untangle - even for those
of us that had been there "from the beginning". At that point it made
sense to start breaking down our systems into smaller, more modular
services that would communicate with each other over HTTP.

## Go to The Rescue

About 2 years ago we started using Go to develop some of these new
systems. Go offered an order of magnitude (or two) performance
improvements making real concurrency something that is actually easy to
accomplish. Go producing statically linked binaries also simplified the
deployment model: no more worrying about which version of
the ruby runtime a given service requires.

## The Challenge of Microservice Style Architecture

I’m sure this platform evolution story will sound familiar to many. The
move to a microservice style architecture is reinforced with the quick
adoption of container based technologies. Nothing is for free though and
some of the complexity that existed in the initial monolithic system
can now be found when looking at the interdependencies between the many
services. Figuring out what is affected when changing a service can
sometimes be as or even more challenging than understanding the
ramification of a change in a software module in the initial system.
There is also a host of new operational challenges that comes along with
having to manage a fleet of interconnected services. Finding the root
cause of failures can be quite challenging, so is scaling the services
in a way that a given cluster does not overload another for example.

As we worked through these issues it became clear that the *interface*
of services becomes a critical piece in the microservice model: adopting
consistent patterns when developing the interfaces help at many
different levels: development, deployment, production support all
benefit from adopting good interface standards. To take just a few
examples:

* Versioning interfaces makes it possible to remain agile and update
  services to accommodate new product needs without impacting existing
  systems.
* Adding a "bulk" version of operations exposed by the service
  interfaces can help improve performance drastically.
* Writing consistent interfaces with common patterns allows for shared
  modules between the various clients (be it JavaScript modules for UIs,
  Go packages or Rubygems).
* Properly classifying error codes help simplify error handling.
* Publishing documentation for the APIs in a consistent format helps
  speed up development and adoption.

etc. the list goes on.

## Interfaces, interfaces, interfaces

The end result is that a lot of focus is now put on designing the REST
API for the services that make up the platform. The API is not just a
mean for transporting data from a client to a server: it is part of the
*semantic* of the service. Making the right choices when designing an
API can have drastic effects down the line both in terms of ease of
development but also - and more importantly - in terms of user
experience for both internal and external users. Performance, security,
flexibility are all dependent on having good APIs.

As the importance of designing good APIs became clear **so did the lack
of tools to support it**. This lead RightScale to develop [Praxis](http://praxis-framework.io/),
a ruby web application framework that makes it possible to describe the
*design* of a REST API explicitly and leverage that design during
implementation.

## Introducing **goa**

What about Go? I hear you asking. Wouldn’t it be great to mix the
awesome performance, concurrency support, ease of deployment, and
statically typed language benefits with a design-first approach to API
development? My thoughts exactly and the reasons behind [goa](http://goa.design).

At first I wasn’t sure whether creating a DSL to describe an API design
in Go would even be possible or yield something that is usable (that’s
an area where dynamic languages definitely shine). goa started as an
experiment but after many iterations of various degrees of ugliness
I’m finally quite happy with the end result. Credits go to [Gomega](https://onsi.github.io/gomega/)
for showing how using anonymous functions can help produce a clean and
terse DSL. The goa DSL abstractions are taken directly from Praxis with
a few tweaks to make them more amenable to the static nature and
philosophy of the Go language.

Here is what the DSL looks like for defining a type that can be used
in the definition of request payloads or response media types:
```go
var BottlePayload = Type("BottlePayload", func() {
	Attribute("name", func() {
		MinLength(2)
	})
	Attribute("vineyard", func() {
		MinLength(2)
	})
	Attribute("vintage", Integer, func() {
		Minimum(1900)
		Maximum(2020)
	})
	Attribute("color", func() {
		Enum("red", "white", "rose", "yellow", "sparkling")
	})
})
```
This type can then be used when defining resource actions:
```go
var _ = Resource("bottle", func() {
	DefaultMedia(Bottle)
	BasePath("bottles")
	Parent("account")
	Action("create", func() {
		Routing(POST(""))
		Description("Record new bottle")
		Payload(BottlePayload, func() {
			Required("name", "vineyard")
		})
		Response(Created)
	})
})
```
As you can see the DSL is fairly self-descriptive. This last example
also shows an interesting property which is that types can be referred
to in different contexts and each context can add specific validations.
Here the `create` action specifies that the `name` and `vineyard`
fields of the `BottlePayload` data structure are required when the type
is used in the payload (request body) of the `create` action for example.

Obviously the DSL contains many more keywords, the point here was just
to give you a sense of what it looks like. Should you want to know more
consult the `dsl` package [GoDoc](https://godoc.org/github.com/raphael/goa/design/dsl).

## The Magic: Code Generation

goa comes with the `goagen` tool that runs the DSL which ends up
producing simple data structures that represent the API design. These
data structures describe the resources that make up the API and for
each resource the actions complete with a description of their
parameters, payload and responses. Note that the term *resource* here is
very loosely defined. It's just a convenient way to group API endpoints
(called *actions* in the DSL) together. The actual semantic is
irrelevant to goa - goa is not an opinionated framework by design.

`goagen` uses these data structures to generate many different outputs.
The generated code takes care of validating the incoming requests and
coercing the types to the ones described in the design, user code then
only has to worry about the business logic.

Here is a code snippet to illustrate the above, this code implements
the `list` action of a `Bottle` resource:
```go
// List lists all the bottles in the account optionally filtering by year.
func (b *BottleController) List(ctx *app.ListBottleContext) error {
	var bottles []*app.Bottle
	var err error
	if ctx.HasYears {
		bottles, err = b.db.GetBottlesByYears(ctx.AccountID, ctx.Years)
	} else {
		bottles, err = b.db.GetBottles(ctx.AccountID)
	}
	if err != nil {
		return ctx.NotFound()
	}
	return ctx.OK(bottles, "default")
}
```
As you can see the code has access to the request state (`AccountID` and
`Years` here) via fields exposed by the context. The values of the
fields have been validated by goa and their types match the types used
in the design (here `AccountID` is a int and `Years` a slice of int).
The context also exposes the `NotFound` and `OK` methods used to send
the response. Again these methods exist because the design specified
that these were the responses of this action. The design also defines
the response payload so that in this case the `OK` method accepts a
slice of `app.Bottle`. The [cellar](https://github.com/raphael/goa/blob/master/examples/cellar)
example contains implementations for many more actions.

Moving on to the next topic, `goagen` generates documentation from the
design in the form of [JSON schema](http://json-schema.org/latest/json-schema-hypermedia.html)
or [swagger](http://swagger.io). This makes it possible to review the
documentation of the API prior to writing a single line of
implementation, a very valuable tool for validating the API with all the
stakeholders. The [swagger.goa.design](http://swagger.goa.design)
service can inspect the design package of goa applications hosted in
public GitHub repositories and dynamically generate and load their
swagger representation in swagger UI.

Another interesting generation target are API clients: Go package,
command line tool and JavaScript clients. Going back to the problem
statement: how to deal with an exponentially growing number of
interconnected microservices - this is huge. It means that the team in
charge of developing a given microservice can also deliver the clients.
This in turn means that the same client is being reused throughout
which helps with consistency, troubleshooting etc. Things like
enforcing the [X-Request-ID](https://devcenter.heroku.com/articles/http-request-id)
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

The other client `goagen` can produce is the JavaScript module. The
module can be used by both client and server side JavaScript.
Similarly to the Go package the JavaScript module exposes one function
per API action. It uses the [axios](https://github.com/mzabriskie/axios)
library to make the actual requests. Again the cellar example contains
the [generated JavaScript](https://github.com/raphael/goa/blob/master/examples/cellar/js/client.js)
together with [an example](https://github.com/raphael/goa/blob/master/examples/cellar/js/index.html)
on how to use it if you are curious.

Summing it all up, the diagram below shows all the various outputs of
the `goagen` tool:
![goagen diagram](https://cdn.rawgit.com/raphael/goa/master/images/goagenv4.svg "goagen")

## The Engine: Runtime

goa is not just about code generation though. The package also provides
a powerful *context* object that makes it possible to carry deadlines
and cancellation signals to all the request handlers. The
[Timeout middleware](https://godoc.org/github.com/raphael/goa#Timeout)
takes advantage of that to send a cancelation signal to the request
handler after a given amount of time. As we've seen before, the context
object also exposes the request and response states wrapping them in
convenient methods that are specific to each action as described in the
design.

From an operational standpoint, goa supports structured logging via the
[log15](https://godoc.org/gopkg.in/inconshreveable/log15.v2) package, the
ability to insert middleware [globally](https://godoc.org/github.com/raphael/goa#Application.Use)
to the service or only on certain [controllers](https://godoc.org/github.com/raphael/goa#Controller.Use),
a clean error [handling model](https://godoc.org/github.com/raphael/goa#hdr-Error_Handling)
and [graceful shutdown](https://godoc.org/github.com/raphael/goa#GracefulApplication).

All these features have one goal in common: to provide a production
ready runtime environment that helps dealing with the challenges of
running services in a microservice environment. As another example goa
comes with a [`X-Request-ID`](https://devcenter.heroku.com/articles/http-request-id)
middleware built-in to help track requests as they travel through the
various services.

## What's Next?

goa is still very new and while I’m quite excited about its potential,
the proof is in the pudding. There are a number of goa services that are
slated to go to production at RightScale in the near future and I’m sure
that new "interesting" challenges will come out. At the same time goa
seems to solve many problems and so far the adoption has been very
positive. I have been amazed at how quickly the open source community
started contributing back and can’t wait to see what others do with goa.
But that’s enough talk, [try it for yourself](https://github.com/raphael/goa) and let
[me](mailto:raphael@goa.design) know how it goes!
