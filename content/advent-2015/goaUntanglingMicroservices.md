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
only has to worry about the business logic. `goagen` generates
documentation from the design in the form of [JSON schema](http://json-schema.org/latest/json-schema-hypermedia.html) or
[swagger](http://swagger.io). This makes it possible to review the documentation of
the API prior to writing a single line of implementation, a very
valuable tool for validating the API with all the stakeholders.

Another interesting generation target are API clients: Go package,
command line tool and JavaScript clients. Going back to the problem
statement: how to deal with an exponentially growing number of
interconnected microservices - this is huge. It means that the team in
charge of developing a given microservice can also deliver the clients.
This in turn means that the same client is being reused throughout
which helps with consistency, troubleshooting etc. Things like
enforcing the [X-Request-ID](https://devcenter.heroku.com/articles/http-request-id) header, CSRF or CORS
which would otherwise be a tedious manual endeavor now become easily
achievable.

The diagram below shows all the various outputs of the `goagen` tool:
![goagen diagram](https://cdn.rawgit.com/raphael/goa/master/images/goagenv4.svg "goagen")

## The Engine: Runtime

goa is not just about code generation though. The package also provides
a powerful *context* object that makes it possible to carry deadlines
and cancellation signals to all the request handlers. The context
object also exposes the request and response states wrapping them in
convenient methods that are specific to each action as described in the
design.

From an operational standpoint, goa supports structured logging, the
ability to insert middleware globally to the service or only on certain
controllers, a clean error handling model and graceful shutdown. All
these features have one goal in common: to provide a production ready
runtime environment that helps dealing with the challenges of running
services in a microservice environment. As another example goa comes
with a [`X-Request-ID`](https://devcenter.heroku.com/articles/http-request-id) middleware built-in
to help track requests as they travel through the various services.

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
