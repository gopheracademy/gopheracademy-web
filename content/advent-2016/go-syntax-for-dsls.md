+++
author = ["Adam Glassman (@abglassman)"]
date = "2016-12-17T0:00:00-08:00"
title = "Abusing Go Syntax to Create a Domain-Specific Language"
series = ["Advent 2016"]
+++

Go is often the tool of choice for building the guts of a high-performance
system, but Go was also designed with some features that are great for building
high-level abstractions. You don’t need to switch gears to a dynamic language
like Ruby or Python to enjoy pleasant APIs or declarative syntax.

It’s an increasingly popular choice to express an API as a DSL - a
Domain-Specific Language. A DSL is a language-within-a-language that is
compiled or interpreted inside a host language- in our case, Go. Through clever
API design, A DSL begins to look like its own language specially suited to a
particular task. Some DSLs like CSS and SQL are built as stand-alone languages
with their own parsers, but for now we’ll focus on the ones we can build for
the Go compiler, to use within Go code.

DSLs are used for infrastructure automation, data model declaration, query
building, and tons more applications. It can be pleasant to write in a DSL for
these types of connecting-and-configuring tasks because they offer a
*declarative* syntax. Rather than *imperatively* describing all of the logic
and operations needed to get your application into a particular state, a DSL
lets you declare the desired structure and attributes of that state, and its
underlying implementation takes the steps to get there. The resulting code
tends to be easier to read, too.

We’re going to look at how to construct APIs that are valid Go that accepted by
the Go compiler, but that begin to feel like their own language. Now, this
article is called “*Abusing* Go Syntax to Create a DSL” because, well, we might
violate the spirit of Go just a bit in the interest of bending the letter of
its laws to our will. I hope you’ll be judicious in applying the more
questionable practices discussed here. I also hope that by exploring them,
you’ll be inspired to think creatively about how to write expressive Go APIs
that are fun to work with and easy to understand.

## A Motivating Example
We’re going to build a simple DSL for constructing HTTP middleware. This domain
is a great candidate for a DSL because it’s full of common, well-understood and
often reused patterns like access restrictions, rate limiting, session handling,
and more. It would be better for both readers and writers of the code that
implements those patterns if that code read more like a declarative config file
than like an imperative reinvention of the wheel.

## Type Identities
One subtly powerful feature of Go for writing code with a descriptive feel is its
[type identities](https://golang.org/ref/spec#Type_identity).  Most of the time
when declaring our own types in Go, we’re declaring a struct or interface. We
can also declare new type identities for existing types, referring to them by a
new name we choose.

This can be something simple like giving a new name to a basic type, such as a
`string` type for host names:
```
type Host string
```

We can also create type identies for collections:
```
type HostList []Host
type HostSet map[Host]interface{}
```
Now anywhere in our code, a variable of type `HostList` will really be a `[]Host`,
or really a `[]string` under the hood, but with a more descriptive name.

A benefit of these type identities, apart from cosmetics and saved keystrokes,
is that these new types can be enriched with their own methods. For example:
```
func (s HostSet) Add(n Host) {
	s[n] = struct{}{}
}

func (s HostSet) Remove(n Host) {
	delete(s, n)
}

func (s HostSet) Contains(n Host) bool {
	_, found := s[n]
	return found
}
```
Now we can use a `HostSet` as though it were a more complex container struct
accessed via methods:
```
func main() {
	s := make(HostSet)
	s.Add("golang.org")
	s.Add("google.com")
	s.Add("gopheracademy.org")
	s.Remove("google.com")

	hostnames := HostList{
		"golang.org",
		"google.com",
		"gopheracademy.org",
	}
	for _, n := range hostnames {
		fmt.Printf("%s? %v\n", n, s.Contains(n))
	}
}
```
This gives output:
```
golang.org? true
google.com? false
gopheracademy.org? true
```
[Try it out here.](https://play.golang.org/p/ME93NteGCf)

What have we gained here? We’ve created an abstraction over a simple `map`; we
can use it like a set - it has `Add` and `Remove` operations, and a `Contains`
check - we’ve created idioms that can be reused throughout our code. It’s
better-encapsulated than passing around a `map[string]interface{}` and hoping
the “set of hostnames” semantics are honored when accessing the map. It’s also
more fluent and descriptive than, say
```
func SetContains(s map[string]interface{}, hostname string) bool {
	_, found := s[hostname]
	return found
}

func main() {
	s := make(map[string]interface{})
	if SetContains(s, hostname) {
        // do stuff
	}
}
```
Experiment a bit with creating new type identities, particularly for different
slice and map types, and even for channels. What idioms can you create to make
working with these types simpler and clearer?

## Higher-Order Functions
Go incorporates some concepts from functional programming that are invaluable
for creating expressive, declarative APIs. Go offers the ability to assign
functions to variables, to pass a function as an argument to another function,
and to create anonymous functions and
[closures](https://en.wikipedia.org/wiki/Closure_%28computer_programming%29). Using
*higher-order functions* that create, modify, or compose the behavior of other
functions, you can easily combine pieces of logic and functionality into a more
sophisticated whole with just a few statements, rather than by duplicating code
or creating a tangle of conditional logic.

Let’s build on our example from above. Let’s add a method to `HostList` that
takes a function as an input and returns a new `HostList`:
```
func (l HostList) Select(f func(Host) bool) HostList {
	result := make(HostList, 0, len(l))
	for _, h := range l {
		if f(h) {
			result = append(result, h)
		}
	}
	return result
}
```
This method of `HostList` has the effect of creating a new `HostList` for which
the provided condition (func `f`) is `true`. Let’s make a simple condition
function to plug in for `f`:
```
// import “strings”
func IsDotOrg(h Host) bool {
	return strings.HasSuffix(string(h), ".org")
}
```
and use it in our new method of `HostList`:
```
myHosts := HostList{"golang.org", "google.com", "gopheracademy.org"}
fmt.Printf("%v\n", myHosts.Select(IsDotOrg))
```
we see output:
```
[golang.org gopheracademy.org]
```
`Select` returned only those elements of `myHosts` for which the function we passed
into it, `IsDotOrg`, was `true`, the hostnames that contained `.org.`

`func(Host) bool` is a bit gnarly as a parameter type and makes the method signature
of `Select` difficult to read, so let’s use our type identity trick to make it a bit
neater:
```
type HostFilter func(Host) bool
```
This makes the signature of `Select` a bit more readable:
```
func (l HostList) Select(f HostFilter) HostList {
        //...
}
```
and has the added benefit that we can declare some methods of `HostFilter`s:
```
func (f HostFilter) Or(g HostFilter) HostFilter {
    return func(h Host) bool {
        return f(h) || g(h)
    }
}

func (f HostFilter) And(g HostFilter) HostFilter {
    return func(h Host) bool {
        return f(h) && g(h)
    }
}
```
If we want to declare a function that can use these `HostFilter` methods,
unfortunately we will need to go a bit out of our way to do so. For a function
to be a valid receiver of `HostFilter` methods, it’s not sufficient to match
the signature of a `HostFilter`, we need to declare the function as a
`HostFilter` explicitly:
```
var IsDotOrg HostFilter = func(h Host) bool {
	return strings.HasSuffix(string(h), ".org")
}
```
Unfortunately here it becomes clear that we have begun to make good on our
threat to “abuse” Go’s syntax. Declaring a function by assigning an anonymous
function to a variable gives an unclean feeling. Note that this isn’t required
to use higher-order functions, or to take advantage of a type identity for a
function signature - any `func(Host) bool` can be *assigned* to a `HostFilter`
variable or parameter. This hare-brained declaration is only needed to be able
to use functions like `IsDotOrg` as the *receiver* of the `HostFilter` methods.

The payoff, though, is that going to these lengths to allow  our `HostFilter`
functions to use these methods enables an interesting syntax:
```
var HasGo HostFilter = func (h Host) bool {
    return strings.Contains(string(h), "go")
}

var IsAcademic HostFilter = func(h Host) bool {
    return strings.Contains(string(h), "academy")
}

func main() {
    myHosts := HostList{"golang.org", "google.com", "gopheracademy.org"}
    goHosts := myHosts.Select(IsDotOrg.Or(HasGo))
    academies := myHosts.Select(IsDotOrg.And(IsAcademic))

    fmt.Printf("Go sites: %v\n", goHosts)
    fmt.Printf("Academies: %v\n", academies)
}
```
Running this gets:
```
Go sites: [golang.org google.com gopheracademy.org]
Academies: [gopheracademy.org]
```
We can see a language of our own taking shape in an expression like
`myHosts.Select(IsDotOrg.Or(HasGo))`. It reads a bit like English, if a little
like something you might hear in the swamps of Dagobah. The declarative syntax
has begun to emerge - the expression says more about the desired result
(“select the elements of `myHosts` that are .orgs or contain ‘Go’) than it does
about the specific steps required to get there. We used higher-order functions,
`Select`, `And`, and `Or`, to compose behaviors from three different pieces of
code in an entirely dynamic way.

This is a powerful way of expressing behavior, but all of this method chaining
can start to get muddled:
```
// etc.
myHosts.Select(IsDotOrg.Or(HasGo).Or(IsAcademic).Or(WelcomesGophers).And(UsesSSL)
```
So perhaps we can clean things up by using a variadic method:
```
var HostFilter Or = func (clauses ...HostFilter) HostFilter {
    var f HostFilter = nil
    for _, c := range clauses {
        f = f.Or(c)
    }
    return f
}
```
and then rewrite the chained invocation above as:
```
myHosts.Select(Or(IsDotOrg, HasGo, IsAcademic, WelcomesGophers).And(UsesSSL))
```

Another warning: these functional-style constructs are some of the most
dangerously powerful features of Go - all of the truly unreadable Go I’ve ever
read and most of the truly unreadable Go I’ve ever written got to be that way
by abusing these features, creating anonymous functions and passing them
through layer after layer of indirection.

Nevertheless, the dynamism of functional-style programming is invaluable when
building our own language inside of Go. Higher-order functions, functions that
operate on other functions and return whole new functions, afford us the
ability to compose or parameterize behaviors. The purpose of creating a DSL is
to simplify the solutions to a class of problems by exposing to the DSL’s
user a few concepts useful for solving those problems and empowering them
configure and combine those concepts in meaningful ways. Composable dynamic
behaviors created through higher-order programming are one way deliver that
functionality.


## A More Useful Example
We’ll build on our work on hostnames to make something a little closer to what
we might use in a real application. Importing the `net/http` package, let’s
create another type identity:
```
type RequestFilter func(*http.Request) bool
```
We can use a `RequestFilter` in a simple HTTP server to evaluate whether a
given `http.Request` satisfies a particular condition, as we did with
`HostFilter` above. We can use those conditions to determine whether to handle
or reject the request.

We’ll shift from working with hostnames as above to working with ranges of IP
addresses. We’ll use
[CIDR](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing) blocks,
e.g. `"192.168.0.0/16"`, which identifies a range of IPs from `192.168.0.0`
through `192.168.255.255`. We’ll create a `RequestFilter` that filters requests
based on IP.

From the `net` package, we’ll use the
[ParseCIDR](https://godoc.org/net#ParseCIDR) function to parse the CIDRs, and
the [ParseIP](https://godoc.org/net#ParseIP) function to parse IP addresses
from incoming requests. One of the return values from `ParseCIDR` is an
[IPNet](https://godoc.org/net#IPNet) which conveniently has a `Contains` method
that will do the work of telling us whether the incoming IP matches the range
in our CIDR block.

So let’s also import the `net` package and write a `RequestFilter` that takes a
variadic input of CIDR blocks in string form:
```
func CIDR(cidrs ...string) RequestFilter {
	nets := make([]*net.IPNet, len(cidrs))
	for i, cidr := range cidrs {
        // TODO: handle err
		_, nets[i], _ = net.ParseCIDR(cidr)
	}
	return func(r *http.Request) bool {
        // TODO: handle err
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		ip := net.ParseIP(host)
		for _, net := range nets {
			if net.Contains(ip) {
				return true
			}
		}
		return false
	}
}
```

Note that the `net/http` package already contains a type for HTTP handlers,
`HandlerFunc`:

```
type HandlerFunc func(ResponseWriter, *Request)
```

and we’ll be using higher-order functions and our `RequestFilter`s to modify
`http.HandlerFunc`s, so let’s declare a type for functions that operate on
`http.HandlerFunc`s:
```
type Middleware func(http.HandlerFunc) http.HandlerFunc
```

and let’s make some functions to build `Middleware` that uses the `RequestFilter`:
```
func Allow(f RequestFilter) Middleware {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if f(r) {
				h(w, r)
			} else {
				// TODO
				w.WriteHeader(http.StatusForbidden)
			}
		}
	}
}
```
So now, for example, you could modify an HTTP handler `MyHandler` to only accept requests
from `127.0.0.1` with something like:
```
filteredHandler := Allow(CIDR("127.0.0.1/32"))(MyHandler)
```

Let’s try it by running a simple server:
```
func hello(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello\n")
}

func main() {
    http.HandleFunc("/hello", Allow(CIDR("127.0.0.1/32")(hello))
	log.Fatal(http.ListenAndServe(":1217", nil))
}
```
If you hit your new endpoint from your local machine at
[http://0.0.0.0:1217/hello](http://0.0.0.0:1217/hello), you should see “Hello”
in response; if you hit it from another IP address, you should see a `403
Forbidden` error.

For fun, let’s add another kind of `RequestFilter` that implements a really
naive authentication mechanism:

```
func PasswordHeader(password string) RequestFilter {
	return func(r *http.Request) bool {
		return r.Header.Get("X-Password") == password
	}
}
```
and one based on HTTP method:
```
func Method(methods ...string) RequestFilter {
	return func(r *http.Request) bool {
		for _, m := range methods {
			if r.Method == m {
				return true
			}
		}
		return false
	}
}
```
and a `Middleware` that performs some simple logging:
```
func Logging(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[%v] - %s %s\n", time.Now(), r.Method, r.RequestURI)
		f(w, r)
	}
}
```

which we can try with an update to our server:
```
func main() {
    http.HandleFunc("/hello", Logging(Allow(CIDR("127.0.0.1/32")(hello)))
	log.Fatal(http.ListenAndServe(":1217", nil))
}
```

Run this and visit [http://localhost:1217/hello](http://localhost:1217/hello) a
few times in your browser and in the console where the server is running you
should see:
```
[2016-12-14 07:42:12.022266374 -0500 EST] - GET /hello
[2016-12-14 07:42:14.537985456 -0500 EST] - GET /hello
[2016-12-14 07:42:24.220089221 -0500 EST] - GET /hello
```

This syntax is fairly declarative as is, but the method chaining can get a
little awkward. Methods have to be chained in the right order to behave
correctly, and the result can be difficult to read.

We can use a struct to further flesh out our DSL and give our users an even cleaner
way to declare their middleware configuration:
```
type Filters []RequestFilters
type Stack []Middleware
type Endpoint struct {
	Handler    http.HandlerFunc
	Allow      Filters
	Middleware Stack
}
```
then we could express the endpoint above with the same restrictions as:
```
var MyEndpoint = Endpoint{
	Handler: hello,
	Allow: Filters{
		CIDR("127.0.0.1/32"),
	},
	Middleware: Stack{
		Logging,
	},
}
```
Which is much easier to write, read, and modify. We just need to add a few
methods to our struct and type identities turn this declarative description
into a usable `http.HandlerFunc`:
```
// Combine creates a RequestFilter that is the conjunction
// of all the RequestFilters in f.
func (f Filters) Combine() RequestFilter {
	return func(r *http.Request) bool {
		for _, filter := range f {
			if !filter(r) {
				return false
			}
		}
		return true
	}
}

// Apply returns an http.Handlerfunc that has had all of the
// Middleware functions in s, if any, to f.
func (s Stack) Apply(f http.HandlerFunc) http.HandlerFunc {
	g := f
	for _, middleware := range s {
		g = middleware(g)
	}
	return g
}

// Builds the endpoint described by e, by applying
// access restrictions and other middleware.
func (e Endpoint) Build() http.HandlerFunc {
	allowFilter := e.Allow.Combine()
	restricted := Allow(allowFilter)(e.Handler)

	return e.Middleware.Apply(restricted)
}
```
and, finally, modify the server to use the endpoint built this way:
```
func main() {
	http.HandleFunc("/hello", mw.MyEndpoint.Build())
	log.Fatal(http.ListenAndServe(":1217", nil))
}
```

To see the benefit of this mini-DSL we’ve created, let’s add one more kind of
middleware:
```
func SetHeader(key, value string) Middleware {
    return func(f http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *RequestFilter) {
            w.Header().Set(key, value)
            f(w, r)
        }
    }
}
```
And then add it, along with another `RequestFilter`, to our endpoint:
```
var MyEndpoint = Endpoint{
	Handler: hello,
	Allow: Filters{
		CIDR("127.0.0.1/32"),
		PasswordHeader("opensesame"), // added
		Method("GET"), // added
	},
	Middleware: Stack{
		Logging,
		SetHeader("X-Foo", "Bar"), // added
	},
}
```
We’ve added significantly to the complexity of `MyEndpoint` without adding much
complexity to its declaration.

This is a useful DSL for building single HTTP endpoints, but frequently we’ll
want more than just one on a service. We’ll add one last element to our demo
DSL, a way to create several routes and their endpoints at once:
```
type Routes map[string]Endpoint

func (r Routes) Serve(addr string) error {
	mux := http.NewServeMux()
	for pattern, endpoint := range r {
		mux.Handle(pattern, endpoint.Build())
	}

	return http.ListenAndServe(addr, mux)
}
```
and then our service becomes:
```
func main() {
	routes := Routes{
		"/hello": {
			Handler: hello,
			Middleware: Stack{
				Logging,
			},
		},
		"/private": {
			Handler: hello,
			Allow: Filters{
				CIDR("127.0.0.1/32"),
				PasswordHeader("opensesame"),
			},
			Middleware: Stack{
				Logging,
			},
		},
		"/test": {
			Handler: hello,
			Middleware: Stack{
				Logging,
				SetHeader("X-Foo", "Bar"),
			},
		},
	}
	log.Fatal(routes.Serve(":1217"))
}
```
Note that Go automatically infers the type of the `Endpoint` struct literals in
the `Routes` map, saving us even more typing and clutter.

This HTTP middleware DSL shows how much can be accomplished in a relatively small
amount of Go, but it’s a toy example. Here are some ideas for exercises to extend
it and to make the DSL even more powerful:

* Implement additional `RequestFilter`s, like a rate-limiter, perhaps using
  [golang.org/x/time/rate](https://godoc.org/golang.org/x/time/rate) or
  [juju/ratelimit](https://github.com/juju/ratelimit), or a more robust
  authentication mechanism
* Implement another `Middleware`
* Modify the `Endpoint` struct to include a `Deny` field of type `Filters`, that
  rejects the request if any of its `RequestFilter`s is `true`
* Each of the endpoints in the final sample included `Logging` in its
  middleware; add to the DSL a facility to apply a set of common restrictions
  or middleware to all of the endpoints.
* Create a way for this middleware stack to create a `context.Context` and to
  work with handlers that accept them.


To recap, we used type identities to create abstractions over simple collection
types and functions of particular signatures, and we took advantage of Go
syntax features like variadic functions and inferred types to write a smooth,
uncluttered syntax. The heavy lifting in creating our DSL was performed by
higher-order functions that let us create parameterized behaviors that could be
combined and configured at runtime. We employed a few dangerous coding practices to
do it, but as long as we apply them only when reducing complexity for end-users is
the right tradeoff, we can all sleep at night.

The Go you get out of the box is detail-oriented, minimalistic, and can become
verbose. Go gives you the tools, however, to build up your own abstractions-
your own high-level language- to write code that is as pithy, elegant, and
expressive as any you’ll find in a dynamic or purely functional language, but
that still gives us access to all of the features we love about Go.

