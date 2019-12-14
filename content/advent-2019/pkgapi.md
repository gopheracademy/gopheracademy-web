+++
author = ["Chewxy"]
title = "Some Thoughts on Library Design"
linktitle = "Short Title (use when necessary)"
date = 2019-12-09T06:40:42Z
+++

There are many ways to think of a programming. Of late, I have been a fan of thinking of the act of programming as a conversation between the programmer and the computer. Furthering the analogy, in the most basic act of modern programming, the programmer is having two conversations with two different parties. These two parties are often conflated with one another, resulting in cofusion when people discuss programming languages. Given this is Gopheracademy, I will use Go to illustrate my points.

The two conversations the programmer has with the computer is a conversation with the compiler and a conversation with the runtime system. I will explain the difference after the examples that follow.

The first conversation the programmer is having is a conversation with the compiler. When we see code like this:

```
type Foo struct {
     A, B int
}
```

It's the programmer telling the compiler, "hey, next time you see `f := Foo{}`, know that we're talking about some memory space enough for two `int`".

The second conversation the programmer has is telling the computer what to do. So when we see code that looks like this:

```
type plus(a Foo) int { return a.A+a.B }

...
two := plus(Foo{1,1})
```

It's the programmer telling the runtime, "When you see some memory space that we have agreed to call `Foo`, return me the `A+B` value in that memory".

Now, it may seem a bit odd for me to talk about the separate conversations as if the are unrelated with one another. Go is a compiled language so the only thing that the programmer is talking to is the compile. That's true. Ultimately all programs are translated into binary code which the processor executes. However, it is still good to separate the notion of a runtime system and a compile time system.

If you look at a snippet of code, and it doesn't do anything by itself, then it's a piece of code that is part of the conversation with the compiler. If you look at a snippet of code and it appears to tell the computer to do something, then it's a conversation with the runtime system via the compiler.

My introduction of the act of programming as a conversatioon with the computer on two fronts is to facilitate a larger discussion on library design.

# Some Definitions and a Raison d'Être #

But first, let's go back to basics and address "why libraries"? Why do we write software libraries? What benefits do we get from software libraries?

First, note that I am using the term "libraries" instead of "packages", "modules" or "repository". Despite being used interchangably in my mind there are very subtle differences. Allow me to explain.

## A Repository ##

A repository is a collection of files containing soure code. They are typically arranged within a directory in the file system.

## A Library ##

A library a collection of resources - usually source code - that is shared. Source code sharing can come in many forms. The most common way of doing this in Go is through packages, which the language supports [by specification](https://golang.org/ref/spec#Packages).

There are other ways of sharing code as well. What follows is a contrived example to illustrate my point.

Let's say I have a file (let's call it `lib.go`) in a directory called `common`.

```
func MaxInt(a, b int) int {
     if a > b {
     	return a
     }
     return b
}
```

I start a new Go package (called `foo`), and I place it in a directory called `github.com/myusername/bar`. I copy `lib.go` from  `common` to `bar`, and rename the file in `bar` as `lib_bar.go`. Now I edit `lib_bar.go` and prepend the declaration `package foo` at the top so that the complete `lib_bar.go` is as follows:

```
package foo

func MaxInt(a, b int) int {
     if a > b {
     	return a
     }
     return b
}
```

Let's say I now start another new Go package (called 'baz'), and I place it in a directory called `bitbucket.org/myusername/quux`. I copy `lib.go` from `common` to `quux`, and rename the file in `quux` as `lib_baz.go`. I prepende the declaration `package baz` at the top so the complete `lib_baz.go` reads as follows:

```
package baz

func MaxInt(a, b int) int {
     if a > b {
     	return a
     }
     return b
}
```

Now, to take stock:

* in `common` I have a malformed `.go` file called `lib.go`
* in `github.com/myusername/bar` I have a file called `lib_foo.go`. The repository holds the package `foo`
* in `github.com/myusername/quux` I have a file called `lib_baz.go`. The repository holds the package `baz`.

Observe that I have shared the source code from `common/lib.go` into two different packages, `foo` and `baz`. Yes, the source code is shared by copying, instead of having a single source of truth, but the source code, at this point in time, is shared.

It is in this sense that I use the word "library" - a library is source code that is shared.

## A Package ##

In general, libraries of source code in Go are arranged in packages and modules. A package is a collection of `.go` files. Usually a package does one thing. A package may depend on another package.

The astute reader will note that having `lib.go` in the above example will cause any Go project to have a compilation failure. All `.go` files must declare at the very top, what package it is used for. The declaration `package foo` is a conversation to the compiler, telling the compiler to include the file in a package.

## A Module ##

If a package is a collection of files containing source code, a module is a collection of packages. Go modules were designed to solve package dependency issues. Modules in Go are defined by a `go.mod` file, which lists all the packages the module depends on.

## Why Libraries ##

Having introduced all the terms, we can now go back to answer the question: why libraries?

We use libraries because they make our lives easier - reusing source code allows for less duplication of effort. Furthermore, using libraries minimizes errors. While I can quite confidently write linear regressions by hand, I would rather not have to redo it everytime. I am not a particularly good programmer and I often make mistakes. Relying on a library (i.e a package or module) would mean I only need to get it right once.

# What Can Go Into a Library? #

I used a very liberal definition of "library" - that a library is a collection of resources that are shared. These resources are usually source code. Although this is not necessarily always true. I will provide two examples of external resources being put into libraries.

The first example concerns using CUDA for processing.

If you have an nVidia graphics card and you would like to use the graphics card for [GPGPU](https://en.wikipedia.org/wiki/General-purpose_computing_on_graphics_processing_units purposes) purposes, using CUDA. The GPU is a resource you need to access. The access can acquired using the CUDA drivers. In Go, the [`cu`](https://gorgonia.org/cu) library that is a part of the [Gorgonia](https://gorgonia.org) suite of libraries manages the driver and accesses the device.

You can get access by means of the following code

```
d := cu.CurrentDevice()
ctx := cu.NewContext(d,	cu.SchedAuto|cu.MapHost)
mem, err := ctx.MemAlloc(1024)

```

`ctx` is a handle to the GPU. Once you have that handle, you can send work to the GPU to do (like reserving 1MB of graphics card memory in the example). But ultimately `ctx` is a resource.

The second example concerns using files as a resource in a library.

Say I have a list of Shakespeare's plays in ASCII format and I want to put them in a library. I can store the plays as `.txt` files, and put them in some central location to be acccessed.

Or to maximize compatibility with the Go programming language, I can create a new package and have the content be something like this:

```go
package willshakes

const AllsWellThatEndsWell = `Act 1 Scene 1

COUNTESS.
	In delivering my son from me, I bury a second husband.

BERTRAM.
	And I in going, madam, weep o'er my father's death
	anew: but I must attend his majesty's command, to
	whom I am now in ward, evermore in subjection.
...
`

const MacBeth = `Act 1 Scene 1

FIRST WITCH.
	When shall we three meet again
	In thunder, lightning, or in rain?

SECOND WITCH.
	When the hurlyburly's done,
	When the battle's lost and won.

THIRD WITCH.
	That will be ere the set of sun.
...
`
```

Put thus, we can just import `willshakes` and to access the text of MacBeth we simply use `willshakes.MacBeth`.

The difference between the CUDA example and the Shakespeare example is that the CUDA example is an example where the resource is dynamic, while in the Shakespeare example, the resource is static.

The use of the terms "static" and "dynamic" is not good, but are in standard use. To make it clearer, allow me to further explain:

A resource is static if its state is known at compile time. A resource is dynamic if its state unknown at compile time.

Thus the Shakespeare resource is static because the entire corpus is known and available at compile time. The state of a graphics card availability may change and needs to be determined at runtime, therefore it is a dynamic resource.

At the time of writing, there is [a proposal to allow static resources to be embedded in the final binary of a Go program](https://github.com/golang/go/issues/35950), so the story is still to be told on the Go end.

# The Types of Libraries #

Barring discussions on the concrete details of libraries and what form they take (packages, drivers, modules, etc), let's consider the various types of libraries we have discussed so far.

Broadly speaking there are two general classes of libraries:

* Libraries where source code is the primary resource that is being shared.
* Libraries where some resource other than source code is being shared.

The latter of these can be split into two:

* Driver libraries
* Resource libraries

A driver library is typically a package that wrap access to a driver. For example, the aforementioned  [`cu`](https://gorgonia.org/cu) package wraps the CUDA drivers to enable CUDA programming in Go. Similarly the [go-gl](https://github.com/go-gl) package which mediates access to OpenGL). Both packages expose extra helper functions to make the transition between worlds easier, but are fundamentally drvier packages.

The `willshakes` library is an example of a resource library. For a real life use case, I offer the [mnist](gorgonia.org/gorgonia/examples/mnist) library for consideration. Due to the nature of the source data, as well as the inability for Go to handle static resources, the design of the package is limited to source code that loads data from an external file into a data structure. In the Python world, this is a different case as one may use [`keras.datasets.mnist`](https://keras.io/datasets/) immediately as a resource.


# What Makes A Good Library? #

Now that I have defined the things that can go into a library, let's take a look at what makes a good library. We will start with the simpler, more obvious statements, before moving on to more nuanced considerations. Despite this, the simpler statements often come with caveats, which will be briefly explored.

I have a few principles that form the axioms of what makes a good library:

1. Reliable
2. Easy to use/build
3. Generic

## Reliable ##

First and foremost, a library must be reliable. What is the point of using a library if it cannot reliably do what it claims to do? There are many features of a  reliable library, enumerated below.

### Do One Thing ###

A good library does one thing or provides one resource. What constitute "one thing" is usually the point of contention.

For example, consider the [grpc](https://godoc.org/google.golang.org/grpc) library. It does one thing - gRPC. But gRPC has many subcomponents to it - server and client are the two primary subcomponents.

An example on the other extreme can be seen in the packages that pervade npm. `left-pad` was a package that provided one function that padded a string. It did one thing, and many packages depended upon it. Thus when the `left-pad` package was unpublished, it broke the internet.


### Well-Tested ###

A good library is well-tested. Users who use the library must be able to feel confident about the library they're using. I would go so far to prefer only libraries that have been tested using property-based testing (I had previously written [an article about property-based testing on GopherAcademy](https://blog.gopheracademy.com/advent-2017/property-based-testing/)) or have been fuzz-tested.

A good thing to check on a well-tested library is whether the tests test for general cases or only specific cases. This is why I prefer libraries that are fuzz-tested and have PBTs in them. Fuzz-testing checks that the libary functions can handle unforeseen input, while property-based testing requires a deep understanding of the domain space.

Having said that, if you develop driver libaraies, it might be a bit difficult to test such libraries. Perhaps this is an indictment on my poor ability to reason about testing, but I have found no good general-purpose testing patterns in the case of driver libraries.

### Don't Manage Resources For Users ###

A good library does not manage resources for its user. Instead, it provides resource management utilties to the user.

For example: If you're writing a library that uses an OpenGL context to do something with OpenGL, don't create the OpenGL context in the library. Instead, require the user to pass in a OpenGL context.

This is also true for allocations. Where possible, don't create allocations on behalf of the user.

Dave Cheney recently wrote [a most excellent article on the topic of forcing allocations](https://dave.cheney.net/2019/09/05/dont-force-allocations-on-the-callers-of-your-api). The title's a bit confusing but the main point is similar to what I am espousing here.

When a library doesn't manage resources for the user, it becomes clear that the user has to manage resources by themselves. The brunt of the responsibility falls onto the user, but the library becomes more reliable.

Last but not least, don't spawn goroutines on behalf of the user.


## Easy to Use ##

A good library is easy to use. There are a number of ways that a library can be easy to use.

### Good Documentation and Examples ###

A good library has good documentation. And to readers who think "tests are documentation", yes! Go has good support for examples, which are both documentation and tests. I enjoy using libraries that have examples when I go to their godoc.

### Don't Panic ###

Panics should only happen in a case when there are no better options. Usually returning errors are a better thing to do.

### Minimal Dependencies ###

This is fairly contentious especially in the Big Picture view of this article (see the Tension section below). But in my opinion a good library has minimal dependencies.

This is especially true of libraries where source code are the primary resource being shared. If a library whose purpose is to share source code were to depend on some resource library, I would be quite suspicious.

Additional dependencies also increase the difficulty to use. I often check what each library imports in order to know that my imports are not going to suddenly call home to some server somewhere. I am not fastidious over it, only because there is so much to check.

### Make the Zero Value Useful ###

One of the Go proverbs, the zero value of any data type should be useful. This avoids the need for complicated constructor functions. A very good example I enjoy is Gonum's `mat.Dense` type.

The `mat.Dense` data type has a method `Mul` which performs matrix multiplication. It has the following type signature:

```
func (m *Dense) Mul(a, b Matrix)
```

The result of `a × b` is placed in `m`. Thus, if `a` is a (2,3) matrix and `b` is a (3,2) matrix, then `m` will be a (2,2) matrix. The documentation is not clear, so most people will try something like this:

```
c := mat.NewDense(2, 2, make([]float64, 4))
c.Mul(a, b)
```


In actuality, this would work as well:

```
var c mat.Dense
c.Mul(a, b)
```

## Generic ##

A good library is also generic - in that it can be used under a number of different scenarios. This has almost nothing to do with generics. Generics may help, but as of now Go provides enough for a library to be generic.

### Accept Interfaces, Return Structs ###

This has been [said](https://medium.com/@cep21/what-accept-interfaces-return-structs-means-in-go-2fe879e25ee8) [to](https://mycodesmells.com/post/accept-interfaces-return-struct-in-go) [death](https://www.integralist.co.uk/posts/go-interfaces/) (even I had a post of [how to use interfaces in Go](https://blog.chewxy.com/2018/03/18/golang-interfaces/)), so allow me to say it once more: Accept interfaces, Return Structs.

A function that can accept anything within limits is by definition generic.

### Play Nice. Compose ###

The key to a library that is generic is that it is composable. Yes, libraries compose. If we are to consider only packages (i.e. libraries whose main purpose is to share source code), then the logical endpoint would be [MLton-style modules](https://mlton.org/Features) (not to be confused with Go modules).

I hesitate to go further with the logical extreme as introducing MLton (and SML) would require overloading several terms such as `struct` and confusing terms from other languages like `functor`. However, due to the way MLton designed their modules system, the "Do One Thing" ethos is naturally arising - libraries are usually very small. The module system also defines a composing helper that allows modules to be composed.

So how does one compose Go packages? Let us imagine an alternative Go with MLton-style modules. The closest equivalent with what we have right now would be to imagine if packages only ever exported interfaces and functions. You can stil write corresponding data types in a package but you cannot export them. What would the end result be?

Such an alternative programming language would produce objects that are highly composable with one another. "Accept Interfaces, Return Structs" becomes less of a maxim and is essentially enforced by the language. `struct`s embed interfaces instead of concrete types.

If all the concrete data types of package A are accepted by functions of package B, then we say package A and package B are composable.

This happens in Go as is. The following graph shows packages that are composable with one another in my GOPATH.

[IMAGE]

The arrow points at an interface defined outside the package (i.e. this is a dependency)
