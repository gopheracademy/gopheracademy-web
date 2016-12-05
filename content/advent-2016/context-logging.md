+++
author = ["Sam Vilain"]
date = "2016-12-06T05:00:00+00:00"
series = ["Advent 2016"]
title = "Using Go's 'context' library for making your logs make sense"
+++

One of the shiny new toys in Go 1.7 is the 'context' library.  Not
shiny as in it is genuinely new.  It started out at
`golang.org/x/net/context`, which is where you'll need to import it
from if you're on 1.6 or before - but don't worry, the old import path
is completely forwards compatible.  This library has been considered
significant enough to make it into the standard library, and for good
reason.

It can be a little hard to see why it is so useful or how to employ
the API in practice.  In this post I'll show how I've found it can be
used for good effect for one important problem: correlating your
production service's logs.  But you can also extend the trick for uses
such as performance profiling, making sure your exceptions can be
matched against your system logs and more.  There's also a few
"anti-patterns", or uses which are considered harmful.

## By 'service' do you mean API service or what?

It doesn't matter an awful lot.  It could be a regular HTTP service of
some kind, a consumer of kafka topics, a cog in a vast stream
processing machine, anything - the key characteristic being that a
single Go program is responsible for processing more than one
independent thing at a time, or at some point you would want to switch
it from processing one thing at a time to many things at the same
time.  Why else are you using Go?

## Enough future tense.  Give me some content.  What does `context` do?

`context` enables a bunch of cool libraries implementing
*cross-cutting concerns* to get access to the stuff you know you
shouldn't have to muck with all the time.  Let me explain...

One of the things you'll notice after not very long at all using Go is
that it doesn't really use global variables very much.  This seems all
well and good and clean when you're writing your own code to implement
logic, but quickly becomes problematic when it comes to calling or
writing library code.  Some very basic things that you didn't realize
assumed single-threaded operation and required a global, or used a
thread local state variable in your old favorite language, all of a
sudden now must be passed around as formal stack arguments.

Sometimes, this is a good thing - it documents the interdepencies
between components.  Other times, there isn't a true dependency - one
piece of code can operate with or without the extra information that
would be in that global.

In the first case, you can keep using globals, making their access
safe using either a `sync.Mutex` or by initializing them in an
`init()` function.  That works for truly global state, but then you
realize that because you have only one process with a bunch of
goroutines instead of one process per request if you're from python,
or one thread per request if you're from ruby or java, that globals
*just won't work* for communicating optional, supplemental contextual
state.  There is just nowhere to store this kind of *per-request* or
*per-processing unit* information except to pass it down, scope by
scope.

You could just keep adding extra function arguments, one per concern.
This is fine and proper for true dependencies, but when you're talking
about concerns like logging or profiling, it just seems a bit ugly to
pollute your function arguments for business logic with a log object,
a database transaction handle, a profiling object, and whatever else.
It's also tedious and costly to do all that refactoring as solutions
to new concerns are introduced.

You need another plan for this.  That plan is `context`.

Instead of passing around a bunch of random objects for these concerns
all over your code base, you can pass around just one, and those
libraries can get things out of them by themselves when they need them.

## Globals, huh?  Anything else?

Yes.  It fixes the "stop" and "reload" buttons for people using your
site.  Again, in more detail...

A problem that HTTP services have to deal with is stopping work when
connections are closed by the other end.  If you don't, your handler
method is still running and might not notice that anything is awry
until it tries to write the response.  For anything which might do a
database update, this is bad.  Instead, you probably want to know
*before* you commit an open transaction that the user hasn't clicked
the "stop" button.  If it's running an expensive query, and your user
keeps hitting "reload" because the page didn't load, you might now
have an extra query running for every time that impatient user
retried.

Similarly, if you're writing code which does a lot of heavy
computational work, and you're not sure if it will complete in a
reasonable time, you might want to have some kind of escape hatch
so that it can stop if it's taking too long, and you can fix it.

As with globals, the old tricks used in interpreted and excessive
threaded languages don't work.  You can't just cancel a thread, kill a
forked process or set an alarm.  Canceling threads was never a good
idea anyway; it wouldn't kill any sub-threads those threads made, for
example.  To work effectively without these things, everything which
does significant work has to check for these termination signals, and
they all need to do it the same way for cancellation to be possible.
That's fine if it's a system function (as with threads or processes),
or your language uses a virtual machine or an interpreter or something
slow like that.  The virtual machine can stand in for the system.  But
in a language like Go, again this must be done in your code.

And if you change the policy on how to stop the code processing, you
sure don't want to have to change your code all over again.  `context`
solves this problem with its `Deadline` and `CancelFunc` APIs.

This falls into the general bucket of quasi-global concerns.  I'm not
going to focus on cancellation in this post, but beware that some
people
[may assume](https://twitter.com/chris_csguy/status/804700188380172288)
that if your function accepts `ctx` as its first argument, that it
supports cancellation.  Don't let them down!

## That sounds great!  Now how will this make my logs better?

There's many ways to make sure that a service is operating correctly
under the hood.  Let's talk about one standard approach - logging -
and how `context` can help you turn your logs into rich sources of
insight.

Initially, you might start with the built-in `log` library, and its
`Printf` interface:

```go
    package somelibrary
    
    import "log"
    
    func DoSomethingAwesome(so, param string, wow int) {
        // ... awesomeness ...
        log.Printf("did something %s awesome with %v (%d times)",
            so, param, wow)
    }
```

This writes to "standard error".  You test this in your program and
it works out great for getting the feature working:

```
2016/12/01 11:48:50 handling /doit for Bert
2016/12/01 11:48:50 did something so awesome with param (2 times)
2016/12/01 11:48:50 finished handling /doit for Bert
```

But then later once you deploy your code to production, you look at
your logs and see this sort of thing:

```
2016/12/01 11:49:12 handling /doit request for Bert
2016/12/01 11:49:12 handling /doit request for Alex
2016/12/01 11:49:12 did something totally awesome with cheese (3 times)
2016/12/01 11:49:12 did something so awesome with param (2 times)
2016/12/01 11:49:12 finished handling /doit for Alex
2016/12/01 11:49:12 finished handling /doit for Bert
```

Alex reports something went wrong.  But because your program is a high
performance parallelized wonder, you can't be sure which line relates
to her request.

## Tagging requests with request IDs

When you solved this problem in
[blub](http://wiki.c2.com/?BlubParadox), it was pretty easy: you just
assigned a request ID to the request, and then some magic library made
that request ID log in every line logged anywhere in the program.

So how can we do this in Go?

First, let's ditch the standard `log` library, which is horribly
unstructured, and use Uber's `zap` logger.  You could equally well do
this with `logrus` or something like that, of course.

Then we set up a local logging library.

```go
    package myappcontext
    
    import (
        "context"
    
        "github.com/uber-go/zap"
    )
    
    type correlationIdType int
    const (
        requestIdKey correlationIdType = iota
        sessionIdKey
    )
    
    var logger zap.Logger
    
    func init() {
        // a fallback/root logger for events without context
        logger = zap.New(
            zap.NewJSONEncoder(zap.TimeFormatter(TimestampField)),
            zap.Fields(zap.Int("pid", os.Getpid()),
                zap.String("exe", path.Base(os.Args[0]))),
        )
    }
    
    // WithRqId returns a context which knows its request ID
    func WithRqId(ctx context.Context, rqId string) context.Context {
        return context.WithValue(ctx, requestIdKey, requestId)
    }

    // WithSessionId returns a context which knows its session ID
    func WithSessionId(ctx context.Context, sessionId string) context.Context {
        return context.WithValue(ctx, sessionIdKey, sessionId)
    }
    
    // Logger returns a zap logger with as much context as possible
    func Logger(ctx context.Context) zap.Logger {
        newLogger := logger
        if ctx != nil {
            if ctxRqId, ok := ctx.Value(requestIdKey).(string); ok {
                newLogger = newLogger.With(zap.String("rqId", ctxRqId))
            }
            if ctxSessionId, ok := ctx.Value(sessionIdKey).(string); ok {
               newLogger = newLogger.With(zap.String("sessionId", ctxSessionId))
            }
        }
        return newLogger
    }
```

Now, the program can log with the request ID, just by using this
`myappctx.Logger` function:

```go
    package somelibrary
    
    import (
        "context"
        "github.com/uber-go/zap"
        "github.com/FooStartup/myapp/myappcontext"
    )
    
    func DoSomethingAwesome(ctx context.Context, so, param string, wow int) {
        // ... awesomeness ...
        myappcontex.Logger(ctx).Info("did something awesome",
            zap.String("so", so), zap.String("param", param),
            zap.Int("wow", wow"))
    }
```

For this to work, the caller (presumably whatever calls the request
handler) just calls the `WithRqId` or `WithSessionId` functions to add
the new fields, and passes the new, specialized context down:

```go
    package yourapp
    
    import (
        "context"
        "net/http"
        "github.com/uber-go/zap"
        "github.com/FooStartup/myapp/myappcontext"
        "github.com/FooStartup/somelibrary"
        "github.com/pborman/uuid"
    )

    var httpContext = context.Background()
    
    func RequestHandler(w http.ResponseWriter, r *http.Request) {
        rqId := uuid.NewRandom()
        rqCtx := myappcontext.WithRqId(httpContext, rqId)
        logger = myappctx.Logger(rqCtx)
        logger.Info("handling /doit request", zap.String("for", user))
        somelbrary.DoSomethingAwesome(rqCtx, ...)
        logger.Info("finished handling /doit request")
    }
```

Now, the requests will come through tagged!

```
2016/12/01 14:49:12 [INFO] handling /doit request rqId="aad6dcde..." for="Bert"
2016/12/01 14:49:12 [INFO] handling /doit request rqId="7f5b859a..." for="Alex"
2016/12/01 14:49:12 [INFO] did something awesome rqId="aad6dcde..." so="totally" param="cheese" wow="3"
2016/12/01 14:49:12 [INFO] did something awesome rqId="7f5b859a..." so="so" param="param" wow="2"
2016/12/01 14:49:12 [INFO] finished handling /doit request rqId="7f5b859a..."
2016/12/01 14:49:12 [INFO] finished handling /doit request rqId="7f5b859a..."
```

It's now possible to see which line was handled by each request
handler, even though they are arriving in the log interleaved with
each other.

## Wait, so what is `context.WithValue` doing?  Does it return a copy?!

A good conceptual model of context's value storage is a map from
anything (`interface{}`) to anything else (`interface{}`).
`context.WithValue` returns a new context object which will return the
value you poked in when you look up via the key you poked it into
using `.Value`.

Similarly to a map with a key `interface{}`, lookup in a context is a
comparison where the value **and the type** of the key are matched
exactly.  Using a custom `int` type fits this bill perfectly, and
means that the value lookup is very fast: normally just comparing one
pointer (to the type structure) and one value: both single machine
words.

Each context object is also immutable, although the values in it don't
necessarily have to be.  The way to update maps with immutable objects
in functional languages like Erlang is to create a new one with the
new key added, and this is something like what `context.WithValue`
does.  It's actually a singly linked list, and each link only holds a
single key.  It looks it up, and if it doesn't find it, it passes it
to the parent.

What this immutability means is that it can be safely shared - across
goroutines, functions etc, and the state is only ever additive.
Important information that you add is rolled back as the functions
return.  It also makes it very cacheline friendly for decent
multi-core performance.

## So, what happens if the key isn't set?  Nil Pointer Panic?

Values not being set in the context is explicitly encouraged. If you
try to fetch from the context and the key is not set, you'll get `nil`
back instead.

Did I say "encouraged"?  What I actually mean is, if your function
doesn't work at all unless magic values are poked into the context
first, you are almost certainly abusing `context`.

You should always try to cast the value back to the type you expected
using the two-return form of casting, which returns a bool as a second
return value which is only `true` if the typecast was successful.

That's this part:

```go
    if ctxRqId, ok := ctx.Value(requestIdKey).(string); ok {
        newLogger = newLogger.With(zap.String("rqId", ctxRqId))
    }
```

If the request ID was not set into the context, then it doesn't add
the field to the scope logger it returns.

## Then just refactor everything

One thing I glossed over above was that the function signature changed:

```go
    func DoSomethingAwesome(so, param string, wow int) {
```

became:

```go
    func DoSomethingAwesome(ctx context.Context, so, param string, wow int) {
```

This might strike you at first as a bit ugly; having to pass an extra
argument down through callers?!  But consider how you'd have to
implement this if you didn't do this: you'd have to pass down a
*zap.Logger*.  And then, later, when you wanted to pass down some kind
of profiling object, you'd need to pass that down, too.  It's a recipe
for madness, and time-consuming/expensive.

The nice thing about using context for this is that a single extra
argument can cover a potentially unlimited number of cut-across
development patterns.

As such, a maxim I've adopted about this is: *there is no harm in*
***any*** *function taking and passing down context objects*.
Basically any function which is doing anything non-trivial:

1. calling out to external services, like a database
2. making key business logic decisions
3. producing side effects

These types of functions can accept a `ctx`, even if they do nothing
with it or merely pass it down the stack.  Some functions are "pure"
and do none of those things; fine, they don't need a context (until
they do).

## Tired of refactoring?  Take a breather with `context.TODO()`

There's a couple of blank, starting context objects:
`context.Background()` and `context.TODO()`.

The first is supposed to be used at what you consider to be the "top
level" of your program.  The ideal state is where the `main` creates a
background context, maybe tags it with the program name or something
and then passes it into every single worker and routine that it starts
for their specialization.

However, in a large established codebase, it isn't that simple.  Enter
`context.TODO()`.  If you're calling a function which has had the
`context.Context` parameter added, but the caller doesn't have it
passed down to it, you can use `context.TODO()`.  If you've been
correctly making sure that your functions degrade gracefully without a
populated context, this will help you be able to gradually cover your
codebase with contextual awareness.

And then, you'll finally have matched Blub for its logging ability.
But at 1000 times the speed!  :)

## Anti-Patterns

As `context` is still relatively new, and powerful, people are using
it in ways not intended by the original authors.  Not that this sort
of thing is always bad, but here are a few of the common abuses of the
module, along with a brief argument about why you shouldn't do this.

### Passing required arguments a long way

Sometimes, you have a function which is very deep into the call stack,
and it needs an ID or something which you know is available at a
higher level.  Resist the temptation to just throw it in the context:
this hides what should be an explicit, required parameter.

**Why this is bad**: readers of your code should be able to tell by
looking at the function arguments key factors determining the
operation of the function.  By passing required arguments via
`context`, you're writing code which is surprising to first-time
readers of it.

**What you should do instead**: you'll have to pass that function down
to all the functions which handle it in between as a formal argument.
If you're finding that many parameters are often being used together -
perhaps you can use a new `struct` type for them.

In the example above, note how the application context only returns a
primed logger, and does not provide a way to access the stored request
ID directly.  This is intentional to avoid this anti-pattern.

It's not a bad idea to unit test that your functions work when passed
`context.Background()` for their context arguments.

### Optional function arguments

"`context` is not an *ersatz* Python kwargs"
[writes](https://twitter.com/peterbourgon/status/804588570770018304)
one respected Golang community member.

*ersatz* is a lovely poetic word in English.  I had to look it up to
confirm I had the meaning right.  Borrowed from German, it means
*faux*, or *fake*, but with an implication of *improper*.  You can
read it as *imposter*, if you like.

In python, `kwargs` is a `map[string]interface{}` passed to functions
which can just grab all the extra arguments you passed on a function
call.  It commonly gets used to pass functions down several scopes, as
functions pull the parts out they want as they go.

**Why this is bad**: In python, the string keys in Python mean that
the arguments can potentially collide.  Every function has to handle
all the arguments in between, and because the keys are all strings you
don't know who is going to do what with that extra argument.  In Go,
using key types which are private avoids that problem, but again this
comes down to - this is surprising to the reader.  `context` is
*optional* context, not materially affecting the logical flow of the
program.

**What you should do instead**: consider passing pointers to values or
structs for optional arguments, and testing that they are `nil`.  Keep
passing those pointers down.

### Passing true globals through context

There's just no need to pass down `pid` through `context`, OK?  Just
use `os.Getpid()` everywhere you need to.  Similarly, you might not
want to pass down things like database pools which really have no use
in this type of localization.

**Why this is bad**: If there is no good reason for a scope, and all
of the functions called beneath it, to have an altered version of
something, then `context` is an unnecessary complication.  It very
marginally slows down valid `context` user.  If it doesn't relate to
a scope or a processing unit, it may result in confusing program
behavior if someone does change it.

**What you should do instead**: use globals for things which seem
global.  Don't worry too much about the performance of `sync.Mutex` -
it's a very fast primitive, especially if you're careful to not hold
it too long.

### Holding contexts around too long

As the context becomes required widely by a lot of functions, there is
a temptation to just "hold onto one".  You should be very careful when
moving context objects from lexical scope to objects which persist
longer than the lifetime of the function which made them.  If you
built all of your logging and other cross-cutting concerns around the
assumption that the context is valid, then this can throw maintainers
off.

**Why this is bad**: you're looking at your logs and wondering why
you're seeing activity tagged with one request ID, and it turns out
you cached a request in your business object and now all the requests
are being tagged with that one request ID!  All that work for nothing.

**What you should do instead**: don't save a `context` in your structs
just to avoid adding it as extra formal argument to method calls.  In
spawned asynchronous processes, try to keep the context objects on
data structures which relate specifically to the work of that request.

### Storing values under regular string/integer keys

This might be safe a lot of the time, but context is not a simple
string map.  Part of the convention is that your keys are private to
your model.  Sticking to this will stop people from thinking they can
treat it as a public store.

**Why this is bad**: a lot of the 'ersatz kwargs' complaints apply.
They could collide, and also a string instance is not private to the
module which called it, and people might abuse the code by pulling
them out again.

**What you should do instead**: use a non-exported type alias, and
`iota` values for your keys.  If you don't have a fixed number of
values, and need to look them up by a caller-passed string, use a type
alias and typecast it on its way in.

## Potentially Good Patterns

I'll close this post out with a list of some other good uses of context:

* **Database Transaction Handles** - typically, queries can operate in
  auto-commit or transactional mode, and it's not their concern
  whether there is an open transaction or not.  A function like this
  enables this with `context`:

    ```go
        func ApplyContextTx(ctx context.Context, stmt *sql.Stmt) *sql.Stmt {
	        if tx, ok := TxFromContext(ctx); ok {
		        return tx.Stmt(stmt)
	        }
	        return stmt
        }
    ```
  
    This has the property that it still works with an empty context.
    The presence of a database transaction could be considered a
    cross-cutting concern.  So long as your model function doesn't
    change its behavior by the presence of the transaction, this is
    fine.

* **Per-transaction Object Caches** - Along with holding the database
  transaction, you might want to consider the merits of caching all
  objects read during the current open transaction, and dumping when
  the transaction commits or rolls back.

    Again, it can work with an empty context by loading from the
    database directly.  The caller doesn't need to care that this magic
    is happening inside of context.

* **Side effect buffers** - Say your application is writing events to
  a messaging queue when a transaction completes successfully.  This
  is a good application of context.  If there is no open transaction,
  the messages should be sent immediately.  Otherwise they buffer and
  are canceled on rollback or sent on commit.

* **Application Tracing & Performance** - if you're using a library
  like Newrelic's `go-agent`, you'll need to pass its transaction
  object down scopes to functions that call external libraries.
  [Context is one way to do this](https://medium.com/@gosamv/using-gos-context-library-for-performance-monitoring-aaf25dacb0fe).

## About the author

A second-generation database-driven developer, Sam has been scratching
his head and figuring out what happened with Perl, Python and Go
applications from the logs since the late 90's.  He is currently
working for [Parsable](http://parsable.com), a San Francisco start-up
serving mostly heavy industry.
