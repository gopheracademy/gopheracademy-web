+++
author = ["Brad Peabody"]
#title = "User Interfaces in Go with Vugu: Intro and Example"
#title = "Writing User Interfaces in Go with Vugu"
title = "HTML, CSS and Go, Together at Long Last: Vugu Premise and Example"
#linktitle = "Short Title (use when necessary)"
date = 2019-12-25T00:00:00Z
+++

## Huh, what's a Vugu?

[Vugu](https://www.vugu.org/) is a [Go library](https://github.com/vugu/vugu) that makes it easy to write HTML markup and Go code which is compiled and run in the browser using [WebAssembly](https://webassembly.org/).

To put it differently: Vugu lets you write single-page web applications (SPAs) starting with HTML layout and using Go to handle program logic.

The name was originally a quasi-portmanteau of "Vue" and "Go", although the further along the project has progressed the less it attempts to follow Vue and instead by necessity finds solutions which are more appropriate to Go as a language.

After all, Go is not JavaScript.

## How it Works

A typical Vugu program follows this sequence:

* You write HTML in a .vugu file.
* Event handlers and other runtime logic are added as Go code to your .vugu file.
* The vugugen tool (or directly, see the [Vugu docs](https://godoc.org/github.com/vugu/vugu) and the `gen` sub-package) then generates Go code that corresponds to your .vugu program.  (There is also a development server which can do the following steps for you automatically upon page refresh.)
* The program is compiled to WebAssembly (.wasm file).
* And then run in the browser, where the HTML you wrote in your .vugu file is synchronized to browser DOM and the Go code you wrote is executed in response to events and so on.

The result is essentially an SPA written in Go.

## Hello, World

A minimal Vugu program (e.g. root.vugu) looks like so:

```html
<div>
    <div class="counter">Hello count: <span vg-html="c.Cnt"></span></div>
    <button @click="c.HandleAddOne()">Say Hello</button>
</div>

<style>
.counter { font-weight: bold; }
</style>

<script type="application/x-go">
type Root struct {
    Cnt int `vugu:"data"`
}

func (c *Root) HandleAddOne() {
    c.Cnt++
}
</script>
```

As you can see, this file contains markup at the top, sprinkled with some properties that have special behavior - `vg-html` prints the specified value into the contents of a tag, `@click` indicates a click handler.

It also has a style block, for styles which should accompany this markup.

And of course the strangest thing here is a `<script>` tag with Go code in it.  This code is copied more or less verbatim into the code-generated output file (root.go, corresponding to root.vugu).  You are of course free to place your Go code anywhere in the appropriate package folder, but including it in a .vugu can be more convenient when organizing your UI components.

See the [Getting Started guide](https://www.vugu.org/doc/start) for more info.

## But Why?

I will make an earnest attempt to not bash JavaScript here.  The truth is, one can do some amazing things with JS these days and its ecosystem abounds with libraries for nearly every task.  However, when you consider the fact that a loosely-type, interpreted language initially written in 10 days (so the [story](https://thenewstack.io/brendan-eich-on-creating-javascript-in-10-days-and-what-hed-do-differently-today/) goes) is essentially the only option for writing modern web user interfaces, it really makes you think... Why can't I use a different language for this task?

Now that of course is what WebAssembly (wasm) is all about.  Assuming it survives the test of time (not guaranteed, but appears likely), it will be commonplace for browsers of the near future to run sophisticated applications written in many languages.  Language choice will be actuality rather than dream when it comes to web development.

So why not use Go for this task?  Well, there are a few challenges, but chief among them is the lack of productive tooling.  Which brings us to Vugu.

## What Makes Web Development Productive

This is probably a deeper question than I am prepared to answer in-depth in this article, but I'll attempt to highlight what I consider to be salient points:

#### Layout should be declarative

***Use HTML and CSS for what the are good at: Declaring document structure and assigning style, respectively.***

It is generally easier (and sometimes much much easier) to describe what is on a page with HTML and how it should look with CSS, than it is to write the equivalent code using a traditional widget library or by emitting such things with regular code.

HTML and CSS are ***declarative***.  HTML gives a structure of elements.  CSS associates styles with them to describe how they are displayed.

Many not-necessarily-obvious-at-first aspects of these languages lend themselves to these tasks:

* HTML tags are nested - they are good at describing both sequence and hierarchy.
* The close of an HTML tag contains the tag name - so deeply nested structures tend to be easier to read (`</div></section></body>` is a lot more descriptive than `} } }`).
* CSS ids and classes are used to match styles.  If something doesn't match, it simply doesn't apply. There is no such thing as an "undefined CSS class" or "unused CSS class" error - the behavior is intentionally kept simple.
* Because the intended use is limited, names can be brief and domain-specific.  `<p>` for paragraph, `<br>` for break, `<div>` and so on.  Names in regular programming languages tend to be longer and require namespaces and prefixes - because there is just a lot more going on and more differentiation needed.
* The simplicity of HTML and CSS also make them much easier to learn.  It is commonplace for people interested in web technology to learn HTML+CSS as a first step, sometimes people can learn the basics in a matter of a few weeks or even days and be somewhat productive even with very little experience.
* There are more points, but you get the idea.

HTML and CSS are productive when used for that which they are intended.  Just because we want to be able to do awesome things with Go in the UI does not mean we should lose the advantages of HTML and CSS.  Even after countless thousands of hours of writing code in "real" programming languages, I still find it personally much more productive to perform page layout in HTML, not in Go, not in JS, not Java, not in PHP - in HTML.  Because HTML is designed for that task.

#### But, real program logic needs a real programming language

***For many years JavaScript has been the only "real" programming language available in-browser, alas 'tis no more.***

HTML and CSS are not "real" programming languages - and that is by design and a good thing.

To clarify, by "real programming language" I mean one with sequence, with flow control, with functions, with data. The stuff that HTML and CSS intentionally lack. You can't write sophisticated program logic without addressing these concerns.

With the advent of WebAssembly, we now have options, Go being among them.  Since this is a post on a blog devoted to Go programming I won't delve into why Go vs alternatives like Rust or C++.  I'm going to assuming you, dear reader, are already sold on the merits of Go as a language.

One thing, often said by many, which I will repeat here is that the larger your program, the more you want the compiler to help you out.  Large programs in languages without type safety and fewer compile-time guarantees are notoriously difficult to maintain.  Many errors show up late, only after testing, and can be difficult to track down and debug.  Solutions like TypeScript are interesting and help, but they are still bolted onto the same system: it's duct tape rather than a rebuild.  Wasm gives us an opportunity to start over with the language of our choice.

#### Reactive web programming has proven its effectiveness

In the olden days web developers used things like jQuery to manipulate the DOM based on user input, data retrieved from a server, etc.

Writing sophisticated applications was tedious and error prone because the page's DOM (HTML) was just another piece of state you had to manage with your application some how some way.  Want to hide a div?  Sure, no problem, just use jQuery's `hide()` method.  Want to show it again, call `show()`.  Want to show/hide it based on some boolean that is elsewhere - sure just update every place that variable is modifed.  Too messy?  Make some wrapper functions to encapsulate your data.  Oh wait, you still have to put show/hide calls in all kinds of places?  And it's not just show/hide it's actually way more complicated?  This sort of mess eventually led to reactive web programming.

If you've read up on Angular, React or even Vue, you'll discover there is a lot to know.  In my opinion these frameworks (heck, frameworks in general) can easily get over-complicated.  But, the core idea is pretty simple and valuable:

***Your page layout is a function of application state.***

In other words: You write markup that describes page layout in a declarative way based on application state.

You don't create DOM elements (tags), you declare them to render in a certain way based on a certain condition.  The framework/library you are using handles deciding when this condition should be re-evaluated and the appropriate DOM updated.

This means your HTML changes from being manipulated imperatively (show/hide) to being declared conditionally (in Vugu you would do `<div vg-if="condition">...</div>`).  Instead of manipulating the page, you manipulate variables (which can be much more concise and compact and without unnecessary duplication), and your page is essentially declared as a function of that, through simple conditions, loops, text output, etc.

#### Vugu is a Library, not a Framework

Here's the difference between a library and framework:

- Framework: Don't call us, we'll call you.
- Library: Call me when you need me.

In other words, in a framework you write something and some other third party code calls it.  With a library, you may need to conform to an interface or otherwise do things a certain way, but ultimately the code path through the application is within your control.

This might sound like a strange statement in regards to a project that has a special file format (.vugu) which is used to code-generate Go code.  After all, something is calling this generated code.  However, care has been taken with Vugu (and will continue to be as the project grows) to ensure you still have control.  Vugu emits a `main_wasm.go` file which is the entry point for the application, but you can modify this file as needed and it does not contain a large amount of setup.  It simply initializes your root component and runs the render loop.  If you need to do something a bit different, that's not a problem.

Large frameworks are a double-edged sword.  Structure is good when it matches the problem at hand.  But it's bad when you have to break out of that shell and do something different.  Vugu tries to walk the line by providing a lot of functionality out of the box, while still maintaining the fact that a the end of the day it's just a library of methods and interfaces that can be used like any other Go library.

## Vugu Wishes You a Happy Holiday Season by Example: The Mathematics of Santa

In the spirit of the Holiday Season, let's have a look at an example application written in Vugu
that calculates a few numbers related to estimated velocity of Santa Claus required to deliver
Christmas presents.

Here is the [running example](http://gh.peabody.io/santamath/), [source code is here](https://github.com/bradleypeabody/santamath).

The minimal setup for a Vugu project is quite simple, requiring only a go.mod file, a root component file (root.vugu) and a development server (devserver.go)

If you check out the project and simply do `go run devserver.go`, that is enough to run the project locally and make changes.

### The Root Component

The root component file ([root.vugu](https://github.com/bradleypeabody/santamath/blob/master/root.vugu)) is the top level visual element.  While components normally start with a tag like `<div>` or `<span>`, the root component can start with an `<html>` tag and specify CSS files to include, e.g.:

```html
<html>

<head>
    <title>The Mathematics of Santa</title>
    <link href="https://stackpath.bootstrapcdn.com/bootstrap/4.4.1/css/bootstrap.min.css" rel="stylesheet"/>
</head>

<body class="bodybg">
    <div>
        <div class="container">
            <h1 class="mt-2">Could Santa Be Real?</h1>
            <h2>Do the Math</h2>

            ...

        </div>
    </div>
</body>

</html>
```

The static parts of the HTML will look familiar.  However, as briefly mentioned earlier in this article, you will also notice things like this:

```html
<span vg-html='fmt.Sprintf("%.0f%%", c.NiceRatioPct)'></span>
```

In this case the `vg-html` attribute is used to emit Go code that evaluates to a string to be injected as the innerHTML of the element in question.  As you can see, a regular `fmt.Sprintf` call is used to perform the formatting.

There are several of these `vg-` directives, including `vg-if` for conditions and `vg-for` for loops.  See the [markup section](https://www.vugu.org/doc/files/markup) of the docs for more info.

This component file can also contain a `<style>` block for CSS and a `<script type="application/x-go">` for Go code.  Since `.vugu` files are code-generated to corresponding `.go` files, the this Go code is simply copied to the output file.

### Component Structs

Each component file corresponds to a single struct.  (Note that this changed from the initial v0.1.0 of Vugu where each component had two structs, this has since been simplified.)

The file `root.vugu` is code generated to `root.go` and corresponds to a struct definition of `type Root struct { //...`

A component struct can contain the necessary data to hold form field values or whatever other state. In this case, it's our metrics we're using to calculate Santa's required velocity:

```go
type Root struct {
    WorldPopulation float64 `vugu:"data"`
    FamilySize float64 `vugu:"data"`
    AverageDistanceBetweenDeliveriesKm float64 `vugu:"data"`
    NiceRatioPct float64 `vugu:"data"`

    vesselMaxVelocityKmh float64
    averageVelocityKmh float64
}
```

The exported fields which are tagged with `vugu:"data"` are properties which correspond to the state of the component.  The other unexported fields are populated in a `BeforeBuild()` method and are not considered part of the state of the component.  This idea of what is part of the "state" of the component is used to determine when to re-render the page - whenever any of those `vugu:"data"` fields change.

Note that `c` is used generally used as the method receiver in many places and loosely means "this component".

### Connecting Attributes and Properties

Aside from `vg-` directives, other syntactic elements like attributes prefixed with `.` or `:` are used in order to dynamically emit HTML markup.

The `:` is used to indicate a dynamic attribute - an attribute which has its value derived from an expression.  Example:

```html
<img :src='c.vtype+".jpg"' />
```

The above outputs a regular `img` tag with a `src` attribute corresponding to the evaluated Go expression.

The `.` is used to indicate a dynamic DOM property.  It is similar to `:` but instead of corresponding to HTML markup, it corresponds to a JavaScript property on that DOM element. For example, this:

```html
<input class="form-control" type="number" .value='c.WorldPopulation'/>
```

Is functionally equivalent to:

```
<input id="temp_id" class="form-control" type="number" />
<script>
document.
    getElementById('temp_id')
    .value = /* value of c.WorldPopulation converted from Go to a JS value*/;
</script>
```

### Responding to Events

DOM events can be registered using `@`.  The name of the event corresponds exactly to the named of the event in the DOM, and the handler can be any Go code.  `c` is used to indicate the current component and `event` is of type [vugu.DOMEvent](https://godoc.org/github.com/vugu/vugu#DOMEvent) and helps adapt the JS DOM event so it's usable in Go.  Example:

```html
<input class="form-control" type="number" .value='c.WorldPopulation'
    @change='c.WorldPopulation, _ = strconv.ParseFloat(event.PropString("target", "value"), 64)' />
```

### Nested Components

In the file `vessel-display.vugu`, you will find another Vugu component which is "nested" inside `root.vugu` with the following syntax:

```html
<main:VesselDisplay :TargetVelocityKmh="c.averageVelocityKmh"/>
```

In this case `main` is the package name, `VesselDisplay` is the name of the component struct, and `TargetVelocityKmh` is a struct field, for which the expression `c.averageVelocityKmh` is used to assign a value.

More complex use cases are still being tested with Vugu to determine how best to layer more complex techniques on top of nested components, but as you can see there is already some decent utility immediately available.

### And More

To learn more about the Vugu project, have a look at the [docs](https://www.vugu.org/doc).

Please note: Vugu is continuing to evolve and as such documentation is a moving target, efforts are made to keep it as up to date as possible.  However as of this writing you probably also want to [look at these notes](https://github.com/vugu/vugu/wiki/Refactor-Notes---Sep-2019) for recent updates.

## Vugu Today

As it stands today, Vugu is in its infancy and yet supports many features one would expect in a complete web UI library or framework:

* Single-file components
* Event Handling
* Conditions and loops with vg-if and vg-for
* Nested components with properties
* Static HTML generation
* A caching system to reduce unnecessary updates
* A development server to handle the edit/build/reload cycle upon page refresh.
* A basic playground

And yet there is still much work to do.

## Vugu Tomorrow

The road yet to be travelled for Vugu includes things like:

* **Component-to-component events.**  Complex applications need a well-defined way to easily pass data between them.
* **Slots.**  A common pattern seen in UI frameworks is to have a component which is passed another component to render a specific part, e.g. a page can call a data table component and provide a custom way to render and individual cell or row.
* **Binary Size Reduction**  Work is under way to support compilation with [TinyGo](https://tinygo.org), which supports a subset of the Go language but outputs much smaller and more efficient binaries.  Some ideas for better compression and caching are also on the table for Vugu programs which need to use the default Go compiler.
* **URL Router** Any big web application needs good support for dealing with its routes.  [A Vugu router is work-in-progress.](https://github.com/vugu/vgrouter)
* **Full Server-Side Rendering Support** A static HTML output mechanism exists and the plan is to build on this to provide an effective static site generation tool as well as server-side rendering more or less out of the box so Vugu apps have a rapid startup time.
* **Component Library(s)** Much work has been done by people much better at website layout than myself to create CSS libraries which follow Material Design or other layout frameworks like Bootstrap.  A convenient way to rapidly assembly Vugu programs using these libraries will go a long way toward improving the overall usefulness of the project.

Hopefully this article has helped shed some light on the motivation behind this project and some of the details of how to use it.

Have a happy holiday season!
