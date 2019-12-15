+++
author = ["Chewxy"]
title = "Some Thoughts on Library Design"
linktitle = "Short Title (use when necessary)"
date = 2019-12-09T06:40:42Z
+++

As programmers we use libraries a lot. But library design is hard. In this article, I will walk through some considerations in designing a library.


Consider the act of programming. The main purpose of programming is to produce programs that do useful things. Everything that follows is simply bureaucracy. Modern programming can be split into two main activities: writing the application and writing the support libraries that the application uses.

# Some Definitions and a Raison d'Être #

But first, let's go back to basics and address "why libraries"? Why do we write software libraries? What benefits do we get from software libraries?

First, note that I am using the term "libraries" instead of "packages", "modules" or "repository". Despite being used interchangably in my mind there are very subtle differences. Allow me to explain.

#### A Repository ####

A repository is a collection of files containing soure code. They are typically arranged within a directory in the file system.

#### A Library ####

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

#### A Package ####

In general, libraries of source code in Go are arranged in packages and modules. A package is a collection of `.go` files. Usually a package does one thing. A package may depend on another package.

The astute reader will note that having `lib.go` in the above example will cause any Go project to have a compilation failure. All `.go` files must declare at the very top, what package it is used for. The declaration `package foo` is a conversation to the compiler, telling the compiler to include the file in a package.

#### A Module ####

If a package is a collection of files containing source code, a module is a collection of packages. Go modules were designed to solve package dependency issues. Modules in Go are defined by a `go.mod` file, which lists all the packages the module depends on.

## Why Libraries ##

Having introduced all the terms, we can now go back to answer the question: why libraries?

We can write an application (with `package main` as the magical declaration at the top), and put all our data structures and code within the main package.

We try not to do that because we know our code can be reused. So we package up the source code and resources that can be reused into libraries.

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

I have a few principles that form the basic principles of what makes a good library:

1. Reliable
2. Easy to use/build
3. Generic

## Reliable ##

First and foremost, a library must be reliable. What is the point of using a library if it cannot reliably do what it claims to do? There are many features of a  reliable library, enumerated below.

### Does One Thing ###

A good library does one thing or provides one resource. What constitute "one thing" is usually the point of contention.

For example, consider the [grpc](https://godoc.org/google.golang.org/grpc) library. It does one thing - gRPC. But gRPC has many subcomponents to it - server and client are the two primary subcomponents.

An example on the other extreme can be seen in the packages that pervade npm. `left-pad` was a package that provided one function that padded a string. It did one thing, and many packages depended upon it. Thus when the `left-pad` package was unpublished, it broke the internet.


### Is Well-Tested ###

A good library is well-tested. Users who use the library must be able to feel confident about the library they're using. I would go so far to prefer only libraries that have been tested using property-based testing (I had previously written [an article about property-based testing on GopherAcademy](https://blog.gopheracademy.com/advent-2017/property-based-testing/)) or have been fuzz-tested.

A good thing to check on a well-tested library is whether the tests test for general cases or only specific cases. This is why I prefer libraries that are fuzz-tested and have PBTs in them. Fuzz-testing checks that the libary functions can handle unforeseen input, while property-based testing requires a deep understanding of the domain space.

Having said that, if you develop driver libaraies, it might be a bit difficult to test such libraries. Perhaps this is an indictment on my poor ability to reason about testing, but I have found no good general-purpose testing patterns in the case of driver libraries.

### Doesn't Manage Resources For Users ###

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

### Doesn't Panic ###

Panics should only happen in a case when there are no better options. Usually returning errors are a better thing to do.

### Has Minimal Dependencies ###

This is fairly contentious especially in the Big Picture view of this article (see the Tension section below). But in my opinion a good library has minimal dependencies.

This is especially true of libraries where source code are the primary resource being shared. If a library whose purpose is to share source code were to depend on some resource library, I would be quite suspicious.

Additional dependencies also increase the difficulty to use. I often check what each library imports in order to know that my imports are not going to suddenly call home to some server somewhere. I am not fastidious over it, only because there is so much to check.

### Makes the Zero Value Useful ###

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

### Accepts Interfaces, Returns Structs ###

This has been [said](https://medium.com/@cep21/what-accept-interfaces-return-structs-means-in-go-2fe879e25ee8) [to](https://mycodesmells.com/post/accept-interfaces-return-struct-in-go) [death](https://www.integralist.co.uk/posts/go-interfaces/) (even I had a post of [how to use interfaces in Go](https://blog.chewxy.com/2018/03/18/golang-interfaces/)), so allow me to say it once more: Accept interfaces, Return Structs.

A function that can accept anything within limits is by definition generic.

### Is Extensible ###

Allow users of your library to extend the functions and behaviour of the objects in your library. The main method to do so in Go would be to take advantage of the composability of data types.

Which brings me to my next point -

### Plays Nice ###

The key to a library that is generic is that it is composable. Yes, _libraries_ compose. If we are to consider only packages (i.e. libraries whose main purpose is to share source code), then the logical endpoint would be [MLton-style modules](https://mlton.org/Features) (not to be confused with Go modules).

 Due to the way MLton (and SML) designed their modules system, the "Do One Thing" ethos is naturally arising - libraries are usually very small. The module system of those languages also defines a helper that allows modules to be composed.

Now, I put it to you, dear readers, that something like that is doable in Go, albeit in a less pure manner.

So how does one compose Go packages? Let us imagine an alternative Go with MLton-style modules. The closest equivalent with what we have right now would be to imagine if packages only ever exported interfaces and functions. You can stil write corresponding data types in a package but you cannot export them. What would the end result be?

Such an alternative programming language would produce objects that are highly composable with one another. "Accept Interfaces, Return Structs" becomes less of a maxim and is essentially enforced by the language. `struct`s embed interfaces instead of concrete types.

If all the concrete data types of package A are accepted by functions of package B, then we say package A and package B are composable.

This happens in Go as is. The following graph shows packages that are composable with one another in my GOPATH.

<div style="margin-left:auto; margin-right:auto; width:90%">
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN"
    "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd" >
<svg contentScriptType="text/ecmascript" width="100%"
     xmlns:xlink="http://www.w3.org/1999/xlink" zoomAndPan="magnify"
     contentStyleType="text/css"
     viewBox="-1797.000000 -1817.000000 3602.000000 3633.000000" height="auto"
     preserveAspectRatio="xMidYMid meet" xmlns="http://www.w3.org/2000/svg"
     version="1.1">
    <g id="edges">
        <path fill="none" stroke-width="3.0"
              d="M 492.459229,-1042.464722 L 273.699768,-1171.437378"
              class="id_gopkg.in/alecthomas/kingpin.v3-unstable id_github.com/spf13/pflag"
              stroke-opacity="1.0" stroke="#e98669"/>
        <path fill="none" stroke-width="3.0"
              d="M -1095.225952,-426.313232 L -1347.951416,-467.191345"
              class="id_github.com/cznic/strutil id_modernc.org/strutil"
              stroke-opacity="1.0" stroke="#dd3f37"/>
        <path fill="none" stroke-width="3.0"
              d="M -1099.713501,-1001.816284 L -1254.047607,-807.443726"
              class="id_github.com/chzyer/readline id_github.com/abiosoft/readline"
              stroke-opacity="1.0" stroke="#df473d"/>
        <path fill="none" stroke-width="3.0"
              d="M -1098.312378,-1019.201904 L -1230.115723,-1254.699585"
              class="id_github.com/chzyer/readline id_gopkg.in/readline.v1"
              stroke-opacity="1.0" stroke="#df473d"/>
        <path fill="none" stroke-width="3.0"
              d="M -1467.032471,-185.713135 L -1623.468506,-693.744385"
              class="id_rsc.io/c2go/cc id_rsc.io/c2go" stroke-opacity="1.0"
              stroke="#e04f42"/>
        <path fill="none" stroke-width="3.0"
              d="M -813.994629,-215.702454 L -590.457764,-55.180206"
              class="id_honnef.co/go/tools/lint id_github.com/cznic/cc/v2"
              stroke-opacity="1.0" stroke="#e15748"/>
        <path fill="none" stroke-width="3.0"
              d="M -823.839355,-211.409760 L -858.128113,60.035862"
              class="id_honnef.co/go/tools/lint id_github.com/cznic/xc"
              stroke-opacity="1.0" stroke="#e15748"/>
        <path fill="none" stroke-width="3.0"
              d="M -832.346436,-218.117630 L -1106.299805,-114.667526"
              class="id_honnef.co/go/tools/lint id_github.com/cznic/cc"
              stroke-opacity="1.0" stroke="#e15748"/>
        <path fill="none" stroke-width="3.0"
              d="M -819.516296,-231.887146 L -743.033997,-487.752869"
              class="id_honnef.co/go/tools/lint id_github.com/cznic/golex/lex"
              stroke-opacity="1.0" stroke="#e15748"/>
        <path fill="none" stroke-width="3.0"
              d="M -812.801941,-225.794693 L -226.418243,-465.120483"
              class="id_honnef.co/go/tools/lint id_honnef.co/go/tools/callgraph"
              stroke-opacity="1.0" stroke="#e15748"/>
        <path fill="none" stroke-width="3.0"
              d="M 1198.144165,-9.482651 L 1444.112183,147.014999"
              class="id_gorgonia.org/gorgonia id_gorgonia.org/golgi"
              stroke-opacity="1.0" stroke="#e7765e"/>
        <path fill="none" stroke-width="3.0"
              d="M 1191.328369,-38.105896 L 1421.328247,-424.771301"
              class="id_gorgonia.org/gorgonia id_github.com/gorgonia/bindgen"
              stroke-opacity="1.0" stroke="#e7765e"/>
        <path fill="none" stroke-width="3.0"
              d="M 1198.988647,-30.035795 L 1725.964966,-307.420227"
              class="id_gorgonia.org/gorgonia id_golang.org/x/time/rate"
              stroke-opacity="1.0" stroke="#e7765e"/>
        <path fill="none" stroke-width="3.0"
              d="M 1179.294556,-40.928261 L 1158.989014,-308.067596"
              class="id_gorgonia.org/gorgonia id_github.com/jonas-p/go-shp"
              stroke-opacity="1.0" stroke="#e7765e"/>
        <path fill="none" stroke-width="3.0"
              d="M 457.047760,-711.435425 L 1029.446167,-957.094543"
              class="id_github.com/gonuts/flag id_gopkg.in/alecthomas/kingpin.v2"
              stroke-opacity="1.0" stroke="#e98669"/>
        <path fill="none" stroke-width="3.0"
              d="M 437.723969,-703.214294 L 165.676849,-588.486328"
              class="id_github.com/gonuts/flag id_rsc.io/c2go/add/obj"
              stroke-opacity="1.0" stroke="#e98669"/>
        <path fill="none" stroke-width="3.0"
              d="M 449.389893,-717.603882 L 501.680389,-988.360718"
              class="id_github.com/gonuts/flag id_gopkg.in/alecthomas/kingpin.v3-unstable"
              stroke-opacity="1.0" stroke="#e98669"/>
        <path fill="none" stroke-width="3.0"
              d="M -827.326904,463.260681 L -822.660767,760.969360"
              class="id_gopkg.in/doug-martin/goqu.v3 id_gopkg.in/abiosoft/ishell.v2"
              stroke-opacity="1.0" stroke="#e87e63"/>
        <path fill="none" stroke-width="3.0"
              d="M -809.334473,433.551392 L -597.383789,326.940369"
              class="id_gopkg.in/doug-martin/goqu.v3 id_github.com/kr/pretty"
              stroke-opacity="1.0" stroke="#e87e63"/>
        <path fill="none" stroke-width="3.0"
              d="M -311.439758,1124.603760 L -537.599548,1009.802673"
              class="id_github.com/icza/bitio id_github.com/nsf/gocode"
              stroke-opacity="1.0" stroke="#d92827"/>
        <path fill="none" stroke-width="3.0"
              d="M -319.074005,1437.485229 L -296.944519,1178.221558"
              class="id_github.com/xwb1989/sqlparser id_github.com/icza/bitio"
              stroke-opacity="1.0" stroke="#d92827"/>
        <path fill="none" stroke-width="3.0"
              d="M -984.833618,-1492.607056 L -759.939392,-1614.057861"
              class="id_gopkg.in/check.v1 id_gopkg.in/yaml.v2"
              stroke-opacity="1.0" stroke="#fbebb1"/>
        <path fill="none" stroke-width="3.0"
              d="M -1141.821411,247.710007 L -865.454529,419.291260"
              class="id_github.com/jmoiron/sqlx id_gopkg.in/doug-martin/goqu.v3"
              stroke-opacity="1.0" stroke="#e87e63"/>
        <path fill="none" stroke-width="3.0"
              d="M -1161.182617,241.056259 L -1724.408447,180.883530"
              class="id_github.com/jmoiron/sqlx id_gopkg.in/gorp.v1"
              stroke-opacity="1.0" stroke="#e87e63"/>
        <path fill="none" stroke-width="3.0"
              d="M -223.687210,-145.554642 L -436.697357,-315.563477"
              class="id_github.com/BurntSushi/toml id_golang.org/x/debug/server"
              stroke-opacity="1.0" stroke="#e56e58"/>
        <path fill="none" stroke-width="3.0"
              d="M 330.410126,1433.586304 L 82.696106,1551.715454"
              class="id_golang.org/x/debug/macho id_github.com/gomidi/midi"
              stroke-opacity="1.0" stroke="#accbbb"/>
        <path fill="none" stroke-width="3.0"
              d="M -933.309265,-734.010376 L -496.006134,-704.186768"
              class="id_github.com/pelletier/go-toml id_github.com/golang/dep"
              stroke-opacity="1.0" stroke="#5e9ab8"/>
        <path fill="none" stroke-width="3.0"
              d="M -1310.566895,755.233826 L -1367.194580,478.784851"
              class="id_github.com/nickng/bibtex id_rsc.io/pdf"
              stroke-opacity="1.0" stroke="#ec9574"/>
        <path fill="none" stroke-width="3.0"
              d="M 329.666840,344.146729 L 841.329346,-188.061157"
              class="id_github.com/gizak/termui id_github.com/pkg/errors"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 322.695801,362.211548 L 332.465668,697.269409"
              class="id_github.com/gizak/termui id_modernc.org/ql"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 318.254089,342.064789 L 93.644440,-182.093216"
              class="id_github.com/gizak/termui id_github.com/spf13/viper"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 332.133636,347.803558 L 561.730408,255.613724"
              class="id_github.com/gizak/termui id_golang.org/x/debug/dwarf"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 321.549561,341.249664 L 298.593658,55.288830"
              class="id_github.com/gizak/termui id_gopkg.in/urfave/cli.v1"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 312.457886,355.123016 L -487.413574,629.508240"
              class="id_github.com/gizak/termui id_gorgonia.org/tensor"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 332.023254,347.539185 L 1140.020630,-2.785503"
              class="id_github.com/gizak/termui id_gorgonia.org/gorgonia"
              stroke-opacity="1.0" stroke="#97818a"/>
        <path fill="none" stroke-width="3.0"
              d="M 312.397522,348.490295 L -241.700165,169.615723"
              class="id_github.com/gizak/termui id_github.com/pkg/sftp"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 330.307007,358.612915 L 555.701233,554.959717"
              class="id_github.com/gizak/termui id_gorgonia.org/cu"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 323.101166,341.240143 L 367.326691,-310.030884"
              class="id_github.com/gizak/termui id_github.com/gorgonia/agogo"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 312.184784,354.187469 L -189.483276,475.682220"
              class="id_github.com/gizak/termui id_github.com/golang/freetype/truetype"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 314.189575,358.273956 L -218.975616,784.663574"
              class="id_github.com/gizak/termui id_google.golang.org/grpc"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 317.455475,360.984375 L 66.621330,832.140198"
              class="id_github.com/gizak/termui id_github.com/korandiz/v4l"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 326.479370,342.045166 L 652.149292,-428.079895"
              class="id_github.com/gizak/termui id_github.com/BurntSushi/xgb"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 322.478912,362.215637 L 328.308929,1048.802612"
              class="id_github.com/gizak/termui id_golang.org/x/xerrors"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 314.639465,344.632019 L -174.818207,-102.744156"
              class="id_github.com/gizak/termui id_github.com/BurntSushi/toml"
              stroke-opacity="1.0" stroke="#967d87"/>
        <path fill="none" stroke-width="3.0"
              d="M 313.296997,346.465149 L 85.176575,214.731171"
              class="id_github.com/gizak/termui id_github.com/lynic/gorgonnx"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 326.816254,361.237366 L 586.901672,920.680054"
              class="id_github.com/gizak/termui id_github.com/cznic/fileutil/falloc"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 332.058899,347.622375 L 843.681641,131.017212"
              class="id_github.com/gizak/termui id_golang.org/x/debug/gosym"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 330.548553,358.325409 L 828.716003,761.889587"
              class="id_github.com/gizak/termui id_modernc.org/fileutil/falloc"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 327.652313,342.630005 L 561.840210,-61.705639"
              class="id_github.com/gizak/termui id_honnef.co/go/tools/unused"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 313.866699,357.848480 L 102.918045,509.629303"
              class="id_github.com/gizak/termui id_github.com/gonum/matrix"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 332.699005,353.708374 L 842.525696,452.237854"
              class="id_github.com/gizak/termui id_github.com/lucasb-eyer/go-colorful"
              stroke-opacity="1.0" stroke="#488cb7"/>
        <path fill="none" stroke-width="3.0"
              d="M 978.675049,-613.771606 L 1235.477905,-716.549500"
              class="id_github.com/fortytw2/leaktest id_github.com/sirupsen/logrus"
              stroke-opacity="1.0" stroke="#3a83b6"/>
        <path fill="none" stroke-width="3.0"
              d="M 961.863525,-617.639343 L 790.454651,-806.181396"
              class="id_github.com/fortytw2/leaktest id_github.com/Sirupsen/logrus"
              stroke-opacity="1.0" stroke="#3a83b6"/>
        <path fill="none" stroke-width="3.0"
              d="M -386.594147,-1069.187012 L -134.017197,-1138.334717"
              class="id_golang.org/x/debug id_modernc.org/db"
              stroke-opacity="1.0" stroke="#337fb6"/>
        <path fill="none" stroke-width="3.0"
              d="M -406.219940,-1061.938965 L -646.430847,-948.757080"
              class="id_golang.org/x/debug id_modernc.org/internal/file"
              stroke-opacity="1.0" stroke="#337fb6"/>
        <path fill="none" stroke-width="3.0"
              d="M -406.793030,-1069.383301 L -807.748718,-1187.578735"
              class="id_golang.org/x/debug id_modernc.org/fileutil/storage"
              stroke-opacity="1.0" stroke="#337fb6"/>
        <path fill="none" stroke-width="3.0"
              d="M -393.918640,-1076.533447 L -307.794067,-1387.464233"
              class="id_golang.org/x/debug id_github.com/cznic/fileutil/storage"
              stroke-opacity="1.0" stroke="#337fb6"/>
        <path fill="none" stroke-width="3.0"
              d="M -389.411163,-1058.877319 L -185.842300,-848.992310"
              class="id_golang.org/x/debug id_github.com/spf13/afero"
              stroke-opacity="1.0" stroke="#337fb6"/>
        <path fill="none" stroke-width="3.0"
              d="M -402.643555,-1075.084961 L -558.469788,-1303.232422"
              class="id_golang.org/x/debug id_modernc.org/lldb"
              stroke-opacity="1.0" stroke="#337fb6"/>
        <path fill="none" stroke-width="3.0"
              d="M -386.719696,-1063.218506 L 118.770241,-901.695129"
              class="id_golang.org/x/debug id_modernc.org/file"
              stroke-opacity="1.0" stroke="#337fb6"/>
    </g>
    <g id="arrows">
        <polyline fill="#e98669" fill-opacity="1.0"
                  class="id_gopkg.in/alecthomas/kingpin.v3-unstable id_github.com/spf13/pflag"
                  points="253.025360,-1183.626221 279.794220,-1181.774536 267.605316,-1161.100220"
                  stroke="none"/>
        <polyline fill="#dd3f37" fill-opacity="1.0"
                  class="id_github.com/cznic/strutil id_modernc.org/strutil"
                  points="-1371.643555,-471.023529 -1346.035278,-479.037384 -1349.867554,-455.345306"
                  stroke="none"/>
        <polyline fill="#df473d" fill-opacity="1.0"
                  class="id_github.com/chzyer/readline id_github.com/abiosoft/readline"
                  points="-1268.971558,-788.648132 -1263.445435,-814.905701 -1244.649780,-799.981750"
                  stroke="none"/>
        <polyline fill="#df473d" fill-opacity="1.0"
                  class="id_github.com/chzyer/readline id_gopkg.in/readline.v1"
                  points="-1241.837158,-1275.642578 -1219.644165,-1260.560303 -1240.587280,-1248.838867"
                  stroke="none"/>
        <polyline fill="#e04f42" fill-opacity="1.0"
                  class="id_rsc.io/c2go/cc id_rsc.io/c2go"
                  points="-1630.531494,-716.681580 -1611.999878,-697.275879 -1634.937134,-690.212891"
                  stroke="none"/>
        <polyline fill="#e15748" fill-opacity="1.0"
                  class="id_honnef.co/go/tools/lint id_github.com/cznic/cc/v2"
                  points="-570.963379,-41.181278 -597.457214,-45.433029 -583.458313,-64.927383"
                  stroke="none"/>
        <polyline fill="#e15748" fill-opacity="1.0"
                  class="id_honnef.co/go/tools/lint id_github.com/cznic/xc"
                  points="-861.135864,83.846642 -870.033508,58.531982 -846.222717,61.539742"
                  stroke="none"/>
        <polyline fill="#e15748" fill-opacity="1.0"
                  class="id_honnef.co/go/tools/lint id_github.com/cznic/cc"
                  points="-1128.752319,-106.189026 -1110.539063,-125.893784 -1102.060547,-103.441269"
                  stroke="none"/>
        <polyline fill="#e15748" fill-opacity="1.0"
                  class="id_honnef.co/go/tools/lint id_github.com/cznic/golex/lex"
                  points="-736.160522,-510.747559 -731.536682,-484.316132 -754.531311,-491.189606"
                  stroke="none"/>
        <polyline fill="#e15748" fill-opacity="1.0"
                  class="id_honnef.co/go/tools/lint id_honnef.co/go/tools/callgraph"
                  points="-204.197708,-474.189545 -221.883713,-454.010223 -230.952774,-476.230743"
                  stroke="none"/>
        <polyline fill="#e7765e" fill-opacity="1.0"
                  class="id_gorgonia.org/gorgonia id_gorgonia.org/golgi"
                  points="1464.361084,159.898407 1437.670532,157.139450 1450.553833,136.890549"
                  stroke="none"/>
        <polyline fill="#e7765e" fill-opacity="1.0"
                  class="id_gorgonia.org/gorgonia id_github.com/gorgonia/bindgen"
                  points="1433.597656,-445.398010 1431.641602,-418.636597 1411.014893,-430.906006"
                  stroke="none"/>
        <polyline fill="#e7765e" fill-opacity="1.0"
                  class="id_gorgonia.org/gorgonia id_golang.org/x/time/rate"
                  points="1747.202515,-318.599030 1731.554321,-296.801453 1720.375610,-318.039001"
                  stroke="none"/>
        <polyline fill="#e7765e" fill-opacity="1.0"
                  class="id_gorgonia.org/gorgonia id_github.com/jonas-p/go-shp"
                  points="1157.169922,-331.998566 1170.954468,-308.977112 1147.023560,-307.158081"
                  stroke="none"/>
        <polyline fill="#e98669" fill-opacity="1.0"
                  class="id_github.com/gonuts/flag id_gopkg.in/alecthomas/kingpin.v2"
                  points="1051.500854,-966.559814 1034.178833,-946.067200 1024.713501,-968.121887"
                  stroke="none"/>
        <polyline fill="#e98669" fill-opacity="1.0"
                  class="id_github.com/gonuts/flag id_rsc.io/c2go/add/obj"
                  points="143.562881,-579.160400 161.013885,-599.543335 170.339813,-577.429321"
                  stroke="none"/>
        <polyline fill="#e98669" fill-opacity="1.0"
                  class="id_github.com/gonuts/flag id_gopkg.in/alecthomas/kingpin.v3-unstable"
                  points="506.231354,-1011.925293 513.462646,-986.085266 489.898102,-990.636169"
                  stroke="none"/>
        <polyline fill="#e87e63" fill-opacity="1.0"
                  class="id_gopkg.in/doug-martin/goqu.v3 id_gopkg.in/abiosoft/ishell.v2"
                  points="-822.284668,784.966431 -834.659302,761.157410 -810.662231,760.781311"
                  stroke="none"/>
        <polyline fill="#e87e63" fill-opacity="1.0"
                  class="id_gopkg.in/doug-martin/goqu.v3 id_github.com/kr/pretty"
                  points="-575.943298,316.155823 -591.991516,337.660614 -602.776062,316.220123"
                  stroke="none"/>
        <polyline fill="#d92827" fill-opacity="1.0"
                  class="id_github.com/icza/bitio id_github.com/nsf/gocode"
                  points="-559.000244,998.939453 -532.167908,999.102295 -543.031189,1020.503052"
                  stroke="none"/>
        <polyline fill="#d92827" fill-opacity="1.0"
                  class="id_github.com/xwb1989/sqlparser id_github.com/icza/bitio"
                  points="-294.903412,1154.308594 -284.988007,1179.242065 -308.901031,1177.201050"
                  stroke="none"/>
        <polyline fill="#fbebb1" fill-opacity="1.0"
                  class="id_gopkg.in/check.v1 id_gopkg.in/yaml.v2"
                  points="-738.821960,-1625.461914 -754.237305,-1603.499146 -765.641479,-1624.616577"
                  stroke="none"/>
        <polyline fill="#e87e63" fill-opacity="1.0"
                  class="id_github.com/jmoiron/sqlx id_gopkg.in/doug-martin/goqu.v3"
                  points="-845.064575,431.950287 -871.784058,429.486237 -859.125000,409.096283"
                  stroke="none"/>
        <polyline fill="#e87e63" fill-opacity="1.0"
                  class="id_github.com/jmoiron/sqlx id_gopkg.in/gorp.v1"
                  points="-1748.272583,178.333984 -1723.133667,168.951431 -1725.683228,192.815628"
                  stroke="none"/>
        <polyline fill="#e56e58" fill-opacity="1.0"
                  class="id_github.com/BurntSushi/toml id_golang.org/x/debug/server"
                  points="-455.455353,-330.534698 -429.211731,-324.942474 -444.182983,-306.184479"
                  stroke="none"/>
        <polyline fill="#accbbb" fill-opacity="1.0"
                  class="id_golang.org/x/debug/macho id_github.com/gomidi/midi"
                  points="61.033234,1562.046021 77.530838,1540.884033 87.861374,1562.546875"
                  stroke="none"/>
        <polyline fill="#5e9ab8" fill-opacity="1.0"
                  class="id_github.com/pelletier/go-toml id_github.com/golang/dep"
                  points="-472.061737,-702.553772 -496.822632,-692.214600 -495.189636,-716.158936"
                  stroke="none"/>
        <polyline fill="#ec9574" fill-opacity="1.0"
                  class="id_github.com/nickng/bibtex id_rsc.io/pdf"
                  points="-1372.010742,455.273071 -1355.438721,476.376770 -1378.950439,481.192932"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_github.com/pkg/errors"
                  points="857.962646,-205.362366 849.979980,-179.744492 832.678711,-196.377823"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_modernc.org/ql"
                  points="333.165192,721.259216 320.470764,697.619141 344.460571,696.919678"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_github.com/spf13/viper"
                  points="84.191429,-204.153137 104.674408,-186.819717 82.614471,-177.366714"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_golang.org/x/debug/dwarf"
                  points="584.002075,246.670990 566.201782,266.749573 557.259033,244.477890"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_gopkg.in/urfave/cli.v1"
                  points="296.673187,31.365788 310.555176,54.328602 286.632141,56.249058"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_gorgonia.org/tensor"
                  points="-510.115051,637.295715 -491.307281,618.157532 -483.519867,640.858948"
                  stroke="none"/>
        <polyline fill="#97818a" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_gorgonia.org/gorgonia"
                  points="1162.040039,-12.332500 1144.794067,8.224207 1135.247192,-13.795214"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_github.com/pkg/sftp"
                  points="-264.539551,162.242676 -238.013641,158.196030 -245.386688,181.035416"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_gorgonia.org/cu"
                  points="573.797791,570.724060 547.819031,564.007996 563.583435,545.911438"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_github.com/gorgonia/agogo"
                  points="368.952698,-333.975769 379.299133,-309.217865 355.354248,-310.843903"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_github.com/golang/freetype/truetype"
                  points="-212.808990,481.331268 -192.307800,464.019379 -186.658752,487.345062"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_google.golang.org/grpc"
                  points="-237.718903,799.653198 -226.470428,775.291931 -211.480804,794.035217"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_github.com/korandiz/v4l"
                  points="55.342930,853.325012 56.028908,826.500977 77.213753,837.779419"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_github.com/BurntSushi/xgb"
                  points="661.496948,-450.184692 663.201660,-423.406067 641.096924,-432.753723"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_golang.org/x/xerrors"
                  points="328.512695,1072.801758 316.309357,1048.904541 340.308502,1048.700684"
                  stroke="none"/>
        <polyline fill="#967d87" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_github.com/BurntSushi/toml"
                  points="-192.533203,-118.936096 -166.722244,-111.601654 -182.914169,-93.886658"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_github.com/lynic/gorgonnx"
                  points="64.393097,202.729202 91.177551,204.339432 79.175598,225.122910"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_github.com/cznic/fileutil/falloc"
                  points="597.019348,942.443176 576.020142,925.738892 597.783203,915.621216"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_golang.org/x/debug/gosym"
                  points="865.782532,121.660385 848.360046,142.067657 839.003235,119.966766"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_modernc.org/fileutil/falloc"
                  points="847.364624,776.996826 821.162415,771.213928 836.269592,752.565247"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_honnef.co/go/tools/unused"
                  points="573.868896,-82.473671 572.224243,-55.691303 551.456177,-67.719978"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_github.com/gonum/matrix"
                  points="83.436737,523.646362 95.909500,499.888641 109.926590,519.369934"
                  stroke="none"/>
        <polyline fill="#488cb7" fill-opacity="1.0"
                  class="id_github.com/gizak/termui id_github.com/lucasb-eyer/go-colorful"
                  points="866.089661,456.791870 840.248718,464.019836 844.802673,440.455872"
                  stroke="none"/>
        <polyline fill="#3a83b6" fill-opacity="1.0"
                  class="id_github.com/fortytw2/leaktest id_github.com/sirupsen/logrus"
                  points="1257.759644,-725.467163 1239.936768,-705.408630 1231.019043,-727.690369"
                  stroke="none"/>
        <polyline fill="#3a83b6" fill-opacity="1.0"
                  class="id_github.com/fortytw2/leaktest id_github.com/Sirupsen/logrus"
                  points="774.310120,-823.939575 799.333740,-814.253662 781.575562,-798.109131"
                  stroke="none"/>
        <polyline fill="#337fb6" fill-opacity="1.0"
                  class="id_golang.org/x/debug id_modernc.org/db"
                  points="-110.869003,-1144.671997 -130.848572,-1126.760620 -137.185822,-1149.908813"
                  stroke="none"/>
        <polyline fill="#337fb6" fill-opacity="1.0"
                  class="id_golang.org/x/debug id_modernc.org/internal/file"
                  points="-668.141602,-938.527466 -651.545654,-959.612427 -641.316040,-937.901733"
                  stroke="none"/>
        <polyline fill="#337fb6" fill-opacity="1.0"
                  class="id_golang.org/x/debug id_modernc.org/fileutil/storage"
                  points="-830.769348,-1194.364746 -804.355652,-1199.088989 -811.141785,-1176.068481"
                  stroke="none"/>
        <polyline fill="#337fb6" fill-opacity="1.0"
                  class="id_golang.org/x/debug id_github.com/cznic/fileutil/storage"
                  points="-301.387543,-1410.593384 -296.229492,-1384.260986 -319.358643,-1390.667480"
                  stroke="none"/>
        <polyline fill="#337fb6" fill-opacity="1.0"
                  class="id_golang.org/x/debug id_github.com/spf13/afero"
                  points="-169.132935,-831.764465 -194.456207,-840.637634 -177.228394,-857.346985"
                  stroke="none"/>
        <polyline fill="#337fb6" fill-opacity="1.0"
                  class="id_golang.org/x/debug id_modernc.org/lldb"
                  points="-572.005920,-1323.050903 -548.560547,-1310.000488 -568.379028,-1296.464355"
                  stroke="none"/>
        <polyline fill="#337fb6" fill-opacity="1.0"
                  class="id_golang.org/x/debug id_modernc.org/file"
                  points="141.631485,-894.390137 115.117722,-890.264526 122.422760,-913.125732"
                  stroke="none"/>
    </g>
    <g id="nodes">
        <circle fill-opacity="1.0" fill="#d7191c" r="10.0" cx="1170.1672"
                class="id_modernc.org/golex" cy="311.45386" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="872.1703"
                class="id_github.com/pkg/errors" cy="-220.1405" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e98669" r="20.0" cx="235.36597"
                class="id_github.com/spf13/pflag" cy="-1194.0376"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e98669" r="20.0" cx="1070.3392"
                class="id_gopkg.in/alecthomas/kingpin.v2" cy="-974.6448"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e98669" r="20.0" cx="124.673874"
                class="id_rsc.io/c2go/add/obj" cy="-571.1945" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e98669" r="20.0" cx="510.11862"
                class="id_gopkg.in/alecthomas/kingpin.v3-unstable"
                cy="-1032.0533" stroke="#000000" stroke-opacity="1.0"
                stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#337fb6" r="20.0" cx="-91.09658"
                class="id_modernc.org/db" cy="-1150.0851" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#d82021" r="10.0" cx="-1734.781"
                class="id_honnef.co/go/tools/lint/lintutil" cy="488.23413"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="333.7627"
                class="id_modernc.org/ql" cy="741.7505" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#d92827" r="20.0" cx="-577.28"
                class="id_github.com/nsf/gocode" cy="989.66046" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#337fb6" r="20.0" cx="-686.68616"
                class="id_modernc.org/internal/file" cy="-929.7897"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#db302c" r="10.0" cx="-1742.5403"
                class="id_github.com/onnx-go" cy="-437.7559" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#dc3832" r="10.0" cx="-958.4486"
                class="id_github.com/jakub-m/gearley" cy="1505.3033"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="76.11698"
                class="id_github.com/spf13/viper" cy="-222.996" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#dd3f37" r="20.0" cx="-1391.8805"
                class="id_modernc.org/strutil" cy="-474.29684" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#dd3f37" r="10.0" cx="-1084.8607"
                class="id_github.com/cznic/strutil" cy="-424.63666"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#df473d" r="20.0" cx="-1281.7191"
                class="id_github.com/abiosoft/readline" cy="-772.5935"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#df473d" r="20.0" cx="-1251.8491"
                class="id_gopkg.in/readline.v1" cy="-1293.5314" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e87e63" r="20.0" cx="-821.9634"
                class="id_gopkg.in/abiosoft/ishell.v2" cy="805.4639"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#df473d" r="10.0" cx="-1093.1843"
                class="id_github.com/chzyer/readline" cy="-1010.03937"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#337fb6" r="20.0" cx="-850.4328"
                class="id_modernc.org/fileutil/storage" cy="-1200.1613"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#337fb6" r="20.0" cx="-295.9153"
                class="id_github.com/cznic/fileutil/storage" cy="-1430.3495"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="603.0258"
                class="id_golang.org/x/debug/dwarf" cy="239.03241"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e04f42" r="20.0" cx="-1636.5645"
                class="id_rsc.io/c2go" cy="-736.27374" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e04f42" r="10.0" cx="-1463.9424"
                class="id_rsc.io/c2go/cc" cy="-175.67812" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e15748" r="20.0" cx="-554.31195"
                class="id_github.com/cznic/cc/v2" cy="-29.223858"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e15748" r="20.0" cx="-863.705"
                class="id_github.com/cznic/xc" cy="104.18502" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e15748" r="20.0" cx="-1147.9305"
                class="id_github.com/cznic/cc" cy="-98.946976" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e15748" r="20.0" cx="-730.2894"
                class="id_github.com/cznic/golex/lex" cy="-530.38885"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e15748" r="20.0" cx="-185.21767"
                class="id_honnef.co/go/tools/callgraph" cy="-481.93604"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e15748" r="10.0" cx="-822.52344"
                class="id_honnef.co/go/tools/lint" cy="-221.82698"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e35f4d" r="10.0" cx="-1478.66"
                class="id_modernc.org/golex/lex" cy="-1046.3503"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e46653" r="10.0" cx="20.473284"
                class="id_honnef.co/go/tools/ssa" cy="1190.9165"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="295.0328"
                class="id_gopkg.in/urfave/cli.v1" cy="10.931522"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e56e58" r="20.0" cx="-471.4778"
                class="id_golang.org/x/debug/server" cy="-343.32263"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="-529.50586"
                class="id_gorgonia.org/tensor" cy="643.94745" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e7765e" r="20.0" cx="1180.8483"
                class="id_gorgonia.org/gorgonia" cy="-20.487225"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e87e63" r="20.0" cx="-557.6296"
                class="id_github.com/kr/pretty" cy="306.94403" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e98669" r="10.0" cx="447.39883"
                class="id_github.com/gonuts/flag" cy="-707.2944"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#eb8d6e" r="10.0" cx="1685.4954"
                class="id_github.com/jung-kurt/gofpdf" cy="673.2726"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e7765e" r="20.0" cx="1481.657"
                class="id_gorgonia.org/golgi" cy="170.90298" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#337fb6" r="20.0" cx="-154.86035"
                class="id_github.com/spf13/afero" cy="-817.0491"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#ec9574" r="20.0" cx="-1376.1245"
                class="id_rsc.io/pdf" cy="435.19006" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="-284.04822"
                class="id_github.com/pkg/sftp" cy="155.94487" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e7765e" r="20.0" cx="1444.0778"
                class="id_github.com/gorgonia/bindgen" cy="-463.0167"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#337fb6" r="20.0" cx="-583.56805"
                class="id_modernc.org/lldb" cy="-1339.9792" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e87e63" r="20.0" cx="-827.6482"
                class="id_gopkg.in/doug-martin/goqu.v3" cy="442.76318"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#ee9d79" r="10.0" cx="1677.7238"
                class="id_github.com/clipperhouse/typewriter" cy="-665.295"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#efa57f" r="10.0" cx="559.6304"
                class="id_github.com/awalterschulze/gographviz" cy="-1715.6193"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#d92827" r="20.0" cx="-293.15997"
                class="id_github.com/icza/bitio" cy="1133.8828" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#d92827" r="10.0" cx="-319.96698"
                class="id_github.com/xwb1989/sqlparser" cy="1447.9471"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#f0ad84" r="10.0" cx="1490.121"
                class="id_github.com/gonum/blas" cy="906.0024" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e7765e" r="20.0" cx="1765.3429"
                class="id_golang.org/x/time/rate" cy="-328.1476"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#f2b48a" r="10.0" cx="-3.146088"
                class="id_github.com/kr/text/cmd/agg" cy="-1470.251"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#f3bc90" r="10.0" cx="885.2668"
                class="id_golang.org/x/oauth2" cy="1158.0277" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#3a83b6" r="20.0" cx="1276.792"
                class="id_github.com/sirupsen/logrus" cy="-733.0843"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#3a83b6" r="20.0" cx="760.52"
                class="id_github.com/Sirupsen/logrus" cy="-839.1081"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#fbebb1" r="10.0" cx="-994.0725"
                class="id_gopkg.in/check.v1" cy="-1487.6178" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="589.25525"
                class="id_gorgonia.org/cu" cy="584.18945" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="370.34158"
                class="id_github.com/gorgonia/agogo" cy="-354.42865"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#f4c495" r="10.0" cx="853.35645"
                class="id_github.com/google/btree" cy="-1172.0885"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#f6cc9b" r="10.0" cx="-1432.6803"
                class="id_github.com/golang/freetype/raster" cy="1091.412"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="-232.73302"
                class="id_github.com/golang/freetype/truetype" cy="486.1565"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#f7d4a0" r="10.0" cx="626.84283"
                class="id_github.com/disintegration/imaging" cy="1321.8031"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#f8dba6" r="10.0" cx="1336.5656"
                class="id_github.com/go-resty/resty" cy="-1193.8885"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#337fb6" r="20.0" cx="161.1588"
                class="id_modernc.org/file" cy="-888.1504" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#fae3ab" r="10.0" cx="222.20563"
                class="id_honnef.co/go/tools/staticcheck/vrp" cy="-1806.0138"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#fbebb1" r="20.0" cx="-720.7842"
                class="id_gopkg.in/yaml.v2" cy="-1635.203" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#fcf3b6" r="10.0" cx="-1786.5137"
                class="id_github.com/golang/glog" cy="-131.49065"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#fefbbc" r="10.0" cx="1365.3402"
                class="id_github.com/kelseyhightower/envconfig" cy="1194.5026"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#accbbb" r="20.0" cx="42.529533"
                class="id_github.com/gomidi/midi" cy="1570.87" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="-253.7288"
                class="id_google.golang.org/grpc" cy="812.45685"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#fbfcbe" r="10.0" cx="-134.40611"
                class="id_github.com/cznic/golex" cy="-1770.2079"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#f4f8be" r="10.0" cx="-1635.0719"
                class="id_github.com/BurntSushi/xgbutil" cy="802.47754"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#edf3be" r="10.0" cx="1442.1349"
                class="id_gonum.org/v1/plot" cy="501.98563" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e7765e" r="20.0" cx="1155.6162"
                class="id_github.com/jonas-p/go-shp" cy="-352.4396"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="45.709297"
                class="id_github.com/korandiz/v4l" cy="871.4204"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e5efbd" r="10.0" cx="-1204.0979"
                class="id_github.com/cznic/ebnf2y/demo" cy="1316.9688"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#deeabd" r="10.0" cx="272.52335"
                class="id_github.com/llgcode/draw2d" cy="1769.66"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="669.4814"
                class="id_github.com/BurntSushi/xgb" cy="-469.06586"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#d7e6bd" r="10.0" cx="606.55023"
                class="id_github.com/faiface/beep" cy="-1347.045"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e87e63" r="10.0" cx="-1150.7421"
                class="id_github.com/jmoiron/sqlx" cy="242.17169"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="328.68677"
                class="id_golang.org/x/xerrors" cy="1093.301" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#d0e1bd" r="10.0" cx="-842.0872"
                class="id_github.com/sanity-io/litter" cy="1195.4553"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e56e58" r="20.0" cx="-207.66476"
                class="id_github.com/BurntSushi/toml" cy="-132.76671"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#c9ddbc" r="10.0" cx="-1091.7227"
                class="id_github.com/nfnt/resize" cy="580.9561" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#c2d8bc" r="10.0" cx="1524.1318"
                class="id_modernc.org/sortutil" cy="-938.86084" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#bbd4bc" r="10.0" cx="-571.9497"
                class="id_github.com/fogleman/gg" cy="1350.2668"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#b3d0bb" r="10.0" cx="-687.063"
                class="id_github.com/peterh/liner" cy="1648.998"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#accbbb" r="10.0" cx="339.88763"
                class="id_golang.org/x/debug/macho" cy="1429.0667"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#a5c7bb" r="10.0" cx="-392.33655"
                class="id_github.com/eugene-eeo/rope" cy="1747.7272"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#e87e63" r="20.0" cx="-1768.6566"
                class="id_gopkg.in/gorp.v1" cy="176.15623" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="46.640537"
                class="id_github.com/lynic/gorgonnx" cy="192.47754"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#9ec2ba" r="10.0" cx="1113.7426"
                class="id_github.com/ulikunitz/xz" cy="1393.0139"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#97beba" r="10.0" cx="1764.984"
                class="id_github.com/alecthomas/template/parse" cy="334.86185"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#90b9ba" r="10.0" cx="573.791"
                class="id_github.com/stretchr/objx" cy="1689.6414"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#88b5b9" r="10.0" cx="1794.9169"
                class="id_rsc.io/x86/x86asm" cy="3.5310864" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#81b0b9" r="10.0" cx="-427.69373"
                class="id_github.com/delaneyj/cogent" cy="-1733.1655"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#7aacb9" r="10.0" cx="340.3909"
                class="id_github.com/gonum/graph" cy="-1505.9939"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#73a7b9" r="10.0" cx="1143.4064"
                class="id_gopkg.in/gizak/termui.v1" cy="973.2217"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#6ca3b8" r="10.0" cx="1111.1829"
                class="id_github.com/kr/fs" cy="-1397.6838" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#659eb8" r="10.0" cx="858.16986"
                class="id_golang.org/x/debug/gocore" cy="1565.4421"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#5e9ab8" r="20.0" cx="-451.60925"
                class="id_github.com/golang/dep" cy="-701.15894"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#5e9ab8" r="10.0" cx="-943.7849"
                class="id_github.com/pelletier/go-toml" cy="-734.7248"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#5695b7" r="10.0" cx="-1453.314"
                class="id_github.com/cznic/lexer" cy="133.56116"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#4f91b7" r="10.0" cx="-75.35827"
                class="id_github.com/googleapis/gax-go" cy="1805.8636"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#ec9574" r="10.0" cx="-1308.4598"
                class="id_github.com/nickng/bibtex" cy="765.52026"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="10.0" cx="322.38977"
                class="id_github.com/gizak/termui" cy="351.716" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="605.66156"
                class="id_github.com/cznic/fileutil/falloc" cy="961.0325"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="884.6604"
                class="id_golang.org/x/debug/gosym" cy="113.66809"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="863.2937"
                class="id_modernc.org/fileutil/falloc" cy="789.9009"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="584.1434"
                class="id_honnef.co/go/tools/unused" cy="-100.21303"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="66.79646"
                class="id_github.com/gonum/matrix" cy="535.6193"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#488cb7" r="20.0" cx="886.2172"
                class="id_github.com/lucasb-eyer/go-colorful" cy="460.68173"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#4188b6" r="10.0" cx="852.6636"
                class="id_github.com/cznic/lex" cy="-1570.9755" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#3a83b6" r="10.0" cx="968.92676"
                class="id_github.com/fortytw2/leaktest" cy="-609.8701"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#337fb6" r="10.0" cx="-396.7215"
                class="id_golang.org/x/debug" cy="-1066.4144" stroke="#000000"
                stroke-opacity="1.0" stroke-width="1.0"/>
        <circle fill-opacity="1.0" fill="#2c7bb6" r="10.0" cx="-1078.9707"
                class="id_github.com/gonum/lapack" cy="1001.1908"
                stroke="#000000" stroke-opacity="1.0" stroke-width="1.0"/>
    </g>
    <g id="node-labels">
        <text font-size="24" x="1170.1672" y="317.2605" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_modernc.org/golex">
            modernc.org/golex
        </text>
        <text font-size="48" x="872.1703" y="-208.52722" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/pkg/errors">
            github.com/pkg/errors
        </text>
        <text font-size="48" x="235.36597" y="-1182.4243" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/spf13/pflag">
            github.com/spf13/pflag
        </text>
        <text font-size="48" x="1070.3392" y="-963.0315" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_gopkg.in/alecthomas/kingpin.v2">
            gopkg.in/alecthomas/kingpin.v2
        </text>
        <text font-size="48" x="124.673874" y="-559.58124" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_rsc.io/c2go/add/obj">
            rsc.io/c2go/add/obj
        </text>
        <text font-size="48" x="510.11862" y="-1020.44006" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog"
              class="id_gopkg.in/alecthomas/kingpin.v3-unstable">
            gopkg.in/alecthomas/kingpin.v3-unstable
        </text>
        <text font-size="48" x="-91.09658" y="-1138.4718" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_modernc.org/db">
            modernc.org/db
        </text>
        <text font-size="24" x="-1734.781" y="494.04077" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_honnef.co/go/tools/lint/lintutil">
            honnef.co/go/tools/lint/lintutil
        </text>
        <text font-size="48" x="333.7627" y="753.3638" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_modernc.org/ql">
            modernc.org/ql
        </text>
        <text font-size="48" x="-577.28" y="1001.27374" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/nsf/gocode">
            github.com/nsf/gocode
        </text>
        <text font-size="48" x="-686.68616" y="-918.1764" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_modernc.org/internal/file">
            modernc.org/internal/file
        </text>
        <text font-size="24" x="-1742.5403" y="-431.94925" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/onnx-go">
            github.com/onnx-go
        </text>
        <text font-size="24" x="-958.4486" y="1511.11" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/jakub-m/gearley">
            github.com/jakub-m/gearley
        </text>
        <text font-size="48" x="76.11698" y="-211.38272" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/spf13/viper">
            github.com/spf13/viper
        </text>
        <text font-size="48" x="-1391.8805" y="-462.68356" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_modernc.org/strutil">
            modernc.org/strutil
        </text>
        <text font-size="24" x="-1084.8607" y="-418.83002" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/cznic/strutil">
            github.com/cznic/strutil
        </text>
        <text font-size="48" x="-1281.7191" y="-760.9802" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/abiosoft/readline">
            github.com/abiosoft/readline
        </text>
        <text font-size="48" x="-1251.8491" y="-1281.9181" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_gopkg.in/readline.v1">
            gopkg.in/readline.v1
        </text>
        <text font-size="48" x="-821.9634" y="817.0772" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_gopkg.in/abiosoft/ishell.v2">
            gopkg.in/abiosoft/ishell.v2
        </text>
        <text font-size="24" x="-1093.1843" y="-1004.2327" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/chzyer/readline">
            github.com/chzyer/readline
        </text>
        <text font-size="48" x="-850.4328" y="-1188.548" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_modernc.org/fileutil/storage">
            modernc.org/fileutil/storage
        </text>
        <text font-size="48" x="-295.9153" y="-1418.7362" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/cznic/fileutil/storage">
            github.com/cznic/fileutil/storage
        </text>
        <text font-size="48" x="603.0258" y="250.64569" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_golang.org/x/debug/dwarf">
            golang.org/x/debug/dwarf
        </text>
        <text font-size="48" x="-1636.5645" y="-724.66046" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_rsc.io/c2go">
            rsc.io/c2go
        </text>
        <text font-size="24" x="-1463.9424" y="-169.87148" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_rsc.io/c2go/cc">
            rsc.io/c2go/cc
        </text>
        <text font-size="48" x="-554.31195" y="-17.610577" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/cznic/cc/v2">
            github.com/cznic/cc/v2
        </text>
        <text font-size="48" x="-863.705" y="115.7983" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/cznic/xc">
            github.com/cznic/xc
        </text>
        <text font-size="48" x="-1147.9305" y="-87.333694" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/cznic/cc">
            github.com/cznic/cc
        </text>
        <text font-size="48" x="-730.2894" y="-518.7756" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/cznic/golex/lex">
            github.com/cznic/golex/lex
        </text>
        <text font-size="48" x="-185.21767" y="-470.32275" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_honnef.co/go/tools/callgraph">
            honnef.co/go/tools/callgraph
        </text>
        <text font-size="24" x="-822.52344" y="-216.02034" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_honnef.co/go/tools/lint">
            honnef.co/go/tools/lint
        </text>
        <text font-size="24" x="-1478.66" y="-1040.5437" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_modernc.org/golex/lex">
            modernc.org/golex/lex
        </text>
        <text font-size="24" x="20.473284" y="1196.7231" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_honnef.co/go/tools/ssa">
            honnef.co/go/tools/ssa
        </text>
        <text font-size="48" x="295.0328" y="22.544804" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_gopkg.in/urfave/cli.v1">
            gopkg.in/urfave/cli.v1
        </text>
        <text font-size="48" x="-471.4778" y="-331.70935" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_golang.org/x/debug/server">
            golang.org/x/debug/server
        </text>
        <text font-size="48" x="-529.50586" y="655.5607" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_gorgonia.org/tensor">
            gorgonia.org/tensor
        </text>
        <text font-size="48" x="1180.8483" y="-8.873943" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_gorgonia.org/gorgonia">
            gorgonia.org/gorgonia
        </text>
        <text font-size="48" x="-557.6296" y="318.5573" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/kr/pretty">
            github.com/kr/pretty
        </text>
        <text font-size="24" x="447.39883" y="-701.48773" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/gonuts/flag">
            github.com/gonuts/flag
        </text>
        <text font-size="24" x="1685.4954" y="679.0792" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/jung-kurt/gofpdf">
            github.com/jung-kurt/gofpdf
        </text>
        <text font-size="48" x="1481.657" y="182.51627" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_gorgonia.org/golgi">
            gorgonia.org/golgi
        </text>
        <text font-size="48" x="-154.86035" y="-805.4358" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/spf13/afero">
            github.com/spf13/afero
        </text>
        <text font-size="48" x="-1376.1245" y="446.80334" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_rsc.io/pdf">
            rsc.io/pdf
        </text>
        <text font-size="48" x="-284.04822" y="167.55815" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/pkg/sftp">
            github.com/pkg/sftp
        </text>
        <text font-size="48" x="1444.0778" y="-451.4034" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/gorgonia/bindgen">
            github.com/gorgonia/bindgen
        </text>
        <text font-size="48" x="-583.56805" y="-1328.366" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_modernc.org/lldb">
            modernc.org/lldb
        </text>
        <text font-size="48" x="-827.6482" y="454.37646" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_gopkg.in/doug-martin/goqu.v3">
            gopkg.in/doug-martin/goqu.v3
        </text>
        <text font-size="24" x="1677.7238" y="-659.48834" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/clipperhouse/typewriter">
            github.com/clipperhouse/typewriter
        </text>
        <text font-size="24" x="559.6304" y="-1709.8126" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog"
              class="id_github.com/awalterschulze/gographviz">
            github.com/awalterschulze/gographviz
        </text>
        <text font-size="48" x="-293.15997" y="1145.4961" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/icza/bitio">
            github.com/icza/bitio
        </text>
        <text font-size="24" x="-319.96698" y="1453.7538" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/xwb1989/sqlparser">
            github.com/xwb1989/sqlparser
        </text>
        <text font-size="24" x="1490.121" y="911.809" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/gonum/blas">
            github.com/gonum/blas
        </text>
        <text font-size="48" x="1765.3429" y="-316.53433" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_golang.org/x/time/rate">
            golang.org/x/time/rate
        </text>
        <text font-size="24" x="-3.146088" y="-1464.4443" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/kr/text/cmd/agg">
            github.com/kr/text/cmd/agg
        </text>
        <text font-size="24" x="885.2668" y="1163.8344" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_golang.org/x/oauth2">
            golang.org/x/oauth2
        </text>
        <text font-size="48" x="1276.792" y="-721.471" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/sirupsen/logrus">
            github.com/sirupsen/logrus
        </text>
        <text font-size="48" x="760.52" y="-827.4948" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/Sirupsen/logrus">
            github.com/Sirupsen/logrus
        </text>
        <text font-size="24" x="-994.0725" y="-1481.8112" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_gopkg.in/check.v1">
            gopkg.in/check.v1
        </text>
        <text font-size="48" x="589.25525" y="595.80273" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_gorgonia.org/cu">
            gorgonia.org/cu
        </text>
        <text font-size="48" x="370.34158" y="-342.81537" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/gorgonia/agogo">
            github.com/gorgonia/agogo
        </text>
        <text font-size="24" x="853.35645" y="-1166.2819" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/google/btree">
            github.com/google/btree
        </text>
        <text font-size="24" x="-1432.6803" y="1097.2186" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/golang/freetype/raster">
            github.com/golang/freetype/raster
        </text>
        <text font-size="48" x="-232.73302" y="497.76978" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog"
              class="id_github.com/golang/freetype/truetype">
            github.com/golang/freetype/truetype
        </text>
        <text font-size="24" x="626.84283" y="1327.6097" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/disintegration/imaging">
            github.com/disintegration/imaging
        </text>
        <text font-size="24" x="1336.5656" y="-1188.0819" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/go-resty/resty">
            github.com/go-resty/resty
        </text>
        <text font-size="48" x="161.1588" y="-876.5371" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_modernc.org/file">
            modernc.org/file
        </text>
        <text font-size="24" x="222.20563" y="-1800.2072" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_honnef.co/go/tools/staticcheck/vrp">
            honnef.co/go/tools/staticcheck/vrp
        </text>
        <text font-size="48" x="-720.7842" y="-1623.5897" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_gopkg.in/yaml.v2">
            gopkg.in/yaml.v2
        </text>
        <text font-size="24" x="-1786.5137" y="-125.684006" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/golang/glog">
            github.com/golang/glog
        </text>
        <text font-size="24" x="1365.3402" y="1200.3092" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog"
              class="id_github.com/kelseyhightower/envconfig">
            github.com/kelseyhightower/envconfig
        </text>
        <text font-size="48" x="42.529533" y="1582.4833" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/gomidi/midi">
            github.com/gomidi/midi
        </text>
        <text font-size="48" x="-253.7288" y="824.0701" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_google.golang.org/grpc">
            google.golang.org/grpc
        </text>
        <text font-size="24" x="-134.40611" y="-1764.4012" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/cznic/golex">
            github.com/cznic/golex
        </text>
        <text font-size="24" x="-1635.0719" y="808.2842" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/BurntSushi/xgbutil">
            github.com/BurntSushi/xgbutil
        </text>
        <text font-size="24" x="1442.1349" y="507.79227" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_gonum.org/v1/plot">
            gonum.org/v1/plot
        </text>
        <text font-size="48" x="1155.6162" y="-340.82632" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/jonas-p/go-shp">
            github.com/jonas-p/go-shp
        </text>
        <text font-size="48" x="45.709297" y="883.0337" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/korandiz/v4l">
            github.com/korandiz/v4l
        </text>
        <text font-size="24" x="-1204.0979" y="1322.7754" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/cznic/ebnf2y/demo">
            github.com/cznic/ebnf2y/demo
        </text>
        <text font-size="24" x="272.52335" y="1775.4667" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/llgcode/draw2d">
            github.com/llgcode/draw2d
        </text>
        <text font-size="48" x="669.4814" y="-457.45258" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/BurntSushi/xgb">
            github.com/BurntSushi/xgb
        </text>
        <text font-size="24" x="606.55023" y="-1341.2384" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/faiface/beep">
            github.com/faiface/beep
        </text>
        <text font-size="24" x="-1150.7421" y="247.97833" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/jmoiron/sqlx">
            github.com/jmoiron/sqlx
        </text>
        <text font-size="48" x="328.68677" y="1104.9143" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_golang.org/x/xerrors">
            golang.org/x/xerrors
        </text>
        <text font-size="24" x="-842.0872" y="1201.262" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/sanity-io/litter">
            github.com/sanity-io/litter
        </text>
        <text font-size="48" x="-207.66476" y="-121.15343" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/BurntSushi/toml">
            github.com/BurntSushi/toml
        </text>
        <text font-size="24" x="-1091.7227" y="586.76276" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/nfnt/resize">
            github.com/nfnt/resize
        </text>
        <text font-size="24" x="1524.1318" y="-933.0542" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_modernc.org/sortutil">
            modernc.org/sortutil
        </text>
        <text font-size="24" x="-571.9497" y="1356.0735" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/fogleman/gg">
            github.com/fogleman/gg
        </text>
        <text font-size="24" x="-687.063" y="1654.8047" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/peterh/liner">
            github.com/peterh/liner
        </text>
        <text font-size="24" x="339.88763" y="1434.8733" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_golang.org/x/debug/macho">
            golang.org/x/debug/macho
        </text>
        <text font-size="24" x="-392.33655" y="1753.5338" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/eugene-eeo/rope">
            github.com/eugene-eeo/rope
        </text>
        <text font-size="48" x="-1768.6566" y="187.76952" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_gopkg.in/gorp.v1">
            gopkg.in/gorp.v1
        </text>
        <text font-size="48" x="46.640537" y="204.09082" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/lynic/gorgonnx">
            github.com/lynic/gorgonnx
        </text>
        <text font-size="24" x="1113.7426" y="1398.8206" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/ulikunitz/xz">
            github.com/ulikunitz/xz
        </text>
        <text font-size="24" x="1764.984" y="340.6685" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog"
              class="id_github.com/alecthomas/template/parse">
            github.com/alecthomas/template/parse
        </text>
        <text font-size="24" x="573.791" y="1695.448" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/stretchr/objx">
            github.com/stretchr/objx
        </text>
        <text font-size="24" x="1794.9169" y="8.64632" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_rsc.io/x86/x86asm">
            rsc.io/x86/x86asm
        </text>
        <text font-size="24" x="-427.69373" y="-1727.3589" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/delaneyj/cogent">
            github.com/delaneyj/cogent
        </text>
        <text font-size="24" x="340.3909" y="-1500.1873" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/gonum/graph">
            github.com/gonum/graph
        </text>
        <text font-size="24" x="1143.4064" y="979.0283" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_gopkg.in/gizak/termui.v1">
            gopkg.in/gizak/termui.v1
        </text>
        <text font-size="24" x="1111.1829" y="-1391.8772" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/kr/fs">
            github.com/kr/fs
        </text>
        <text font-size="24" x="858.16986" y="1571.2488" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_golang.org/x/debug/gocore">
            golang.org/x/debug/gocore
        </text>
        <text font-size="48" x="-451.60925" y="-689.54565" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/golang/dep">
            github.com/golang/dep
        </text>
        <text font-size="24" x="-943.7849" y="-728.91815" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/pelletier/go-toml">
            github.com/pelletier/go-toml
        </text>
        <text font-size="24" x="-1453.314" y="139.3678" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/cznic/lexer">
            github.com/cznic/lexer
        </text>
        <text font-size="24" x="-75.35827" y="1811.6703" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/googleapis/gax-go">
            github.com/googleapis/gax-go
        </text>
        <text font-size="24" x="-1308.4598" y="771.3269" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/nickng/bibtex">
            github.com/nickng/bibtex
        </text>
        <text font-size="24" x="322.38977" y="357.52264" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/gizak/termui">
            github.com/gizak/termui
        </text>
        <text font-size="48" x="605.66156" y="972.64575" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/cznic/fileutil/falloc">
            github.com/cznic/fileutil/falloc
        </text>
        <text font-size="48" x="884.6604" y="125.28137" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_golang.org/x/debug/gosym">
            golang.org/x/debug/gosym
        </text>
        <text font-size="48" x="863.2937" y="801.51416" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_modernc.org/fileutil/falloc">
            modernc.org/fileutil/falloc
        </text>
        <text font-size="48" x="584.1434" y="-88.59975" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_honnef.co/go/tools/unused">
            honnef.co/go/tools/unused
        </text>
        <text font-size="48" x="66.79646" y="547.2326" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/gonum/matrix">
            github.com/gonum/matrix
        </text>
        <text font-size="48" x="886.2172" y="472.295" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/lucasb-eyer/go-colorful">
            github.com/lucasb-eyer/go-colorful
        </text>
        <text font-size="24" x="852.6636" y="-1565.1688" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/cznic/lex">
            github.com/cznic/lex
        </text>
        <text font-size="24" x="968.92676" y="-604.0635" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/fortytw2/leaktest">
            github.com/fortytw2/leaktest
        </text>
        <text font-size="24" x="-396.7215" y="-1060.6078" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_golang.org/x/debug">
            golang.org/x/debug
        </text>
        <text font-size="24" x="-1078.9707" y="1006.99744" fill="#000000"
              style="text-anchor: middle; dominant-baseline: central;"
              font-family="Dialog" class="id_github.com/gonum/lapack">
            github.com/gonum/lapack
        </text>
    </g>
</svg>
</div>

The arrow points at an interface defined outside the package (i.e. this is a dependency). From this we can derive a metric of how composable the entire Go ecosystem is. The size indicates the in-degree - how many packages implement the interfaces of a given package. This is a measure of the Postel-law-ness of the functions in a library. The colours indicate a [modularity class](https://en.wikipedia.org/wiki/Modularity_(networks)).

Out of 762 packages in my GOPATH, only 120 are included here with 62 edges, forming 56 modules. The remaining 600+ packages are excluded for a lack of composability. Mind, for this analysis I excluded interfaces defined in the standard library because I didn't know how to load them for analysis (probably something to do with `types.Universe`).

Thinking about grouping software libraries by its compositionality appears to be a weird idea at first, but it isn't really that weird. In languages that prize abstraction over everything else (i.e. Haskell), this sort of thinking is the norm. I'm not arguing that we should do that. Instead, I am offering a different view on designing libraries.

The metric is important. For example, in building this graph, I also noticed that the Gorgonia libraries are not as composable with each other as I had originally assumed. This will be fixed in the coming summer (winter in the Northern Hemisphere).


# The Tension in Designing Libraries #

If you are a careful reader, you would have immediately spotted the tension that exists between the principles that I list for what qualifies as a good library.

A good reliable library does not manage resources for the user. However, this usually makes the library difficult to use.

Let us revisit the Gonum example from the section Make The Zero Value Useful.

In the first part of the example, repeated here:

```
c := mat.NewDense(2, 2, make([]float64, 4))
c.Mul(a, b)
```

We see that this follows very much the "Don't manage resources for your user". Instead, the user has to create the `*Dense`, and allocate the value (that's what `make([]float64, 4)` is there for).

Gonum provides a user friendly alternative, as shown in the second part of the example:

```
var c mat.Dense
c.Mul(a, b)
```

However, this violates the "Don't manage resources for your user".

A more egregious example can be found in my own `tensor` library. A `*tensor.Dense` has a method `Mul`, defined with a signature as follows:

```
func (t *Dense) Mul(other *Dense, opts ...FuncOpt) (*Dense, error)
```

By default the `tensor` library manages memory allocation for the user. But the functional options allow for modification to the behaviour of `Mul`.

So for example, one may manually manage the allocations:

```
a := tensor.New(tensor.WithShape(2,3), tensor.WithBacking([]float64{...}))
b := tensor.New(tensor.WithShape(3,2), tensor.WithBacking([]float64{...}))
foo := tensor.New(tensor.WithShape(2,2), tensor.Of(tensor.Float64))
c, err := a.Mul(b, T.WithReuse(foo))
```

The result, `c` is exactly the same as `foo`.

So why did I bring up the tension, and showed off two "bad" examples?

Because to resolve the tension, one must consider the bigger picture.

## The Bigger Picture ##

Start with the big picture in mind. The big picture for the `tensor` package is so that it  works generically across data types and generically across computation. This is useful for the kinds of deep learning workload that Gorgonia handles.

For example, the same example from above may be used on `float32` types, computed in a GPU:

```
type Engine struct {
	tensor.StdEng
	ctx cu.Context
	*cublas.Standard
}

// Engine implemnents `tensor.Engine`

e := newEngine()
a := tensor.New(tensor.WithShape(2, 3), tensor.WithEngine(e), tensor.Of(tensor.Float32))
b := tensor.New(tensor.WithShape(3, 2), tensor.WithEngine(e), tensor.Of(tensor.Float32))
c := tensor.New(tensor.WithShape(2, 2), tensor.WithEngine(e), tensorlOf(tensor.Float32))

// fill up the values of a and b
// ...

_, err := a.Mul(b, tensor.WithReuse(c))

```

Now that the big picture is clear, we can choose to make some compromises. To take stock:

* The `tensor` library is designed to be generic across data types and generic across computation.
* We should not manage resources for the user.
* The library must be easy to use.
* The library must be extensible.

If we prioritise "not managing resources for the user", then we immediately lose "easy to use" and "extensible".

However, if we do not prioritize "not managing resources for the user", then a user who wants to use the `tensor` package using CUDA might fall into the trap of thinking that the default behaviour for `Mul` would work on CUDA as well (it will not - the program panics because GPU memory access is quite finnicky)

## Consider the Use Cases ##

To resolve the tension, I considered the different use cases. The most common use case, I reasoned, would be to use the `tensor` package on the CPU, with well known data types like `float64` and `float32`.

I built a heirarchy of needs, with GPU usage at the top. This sacrifices some of the ease-of-use, but I reckoned if you want to use GPU, you'd be an expert user.

## Heirarchical Use of Libraries ##

Sometimes, it's not quite possible to resolve the tension. In cases like that, I would opt to build a heirarchical family of libraries.

In the course of programming across different languages, I have noticed some good patterns. Good libraries are somewhat heirarchically organized - they are built on top of structures from libraries in a compositional manner.

<div style="margin-left:auto;margin-right:auto; width: 80%; margin-bottom:20px">
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<svg xmlns="http://www.w3.org/2000/svg" style="background-color: rgb(255, 255, 255);" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" width="471px" height="201px" viewBox="-0.5 -0.5 471 201" content="&lt;mxfile host=&quot;www.draw.io&quot; modified=&quot;2019-12-14T23:53:26.287Z&quot; agent=&quot;Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:71.0) Gecko/20100101 Firefox/71.0&quot; etag=&quot;1uTiXhRgZH6dchvWc1f9&quot; version=&quot;12.4.2&quot; type=&quot;device&quot; pages=&quot;1&quot;&gt;&lt;diagram id=&quot;wyTi_E9LNr-jFQr5dkLu&quot; name=&quot;Page-1&quot;&gt;7VhRU6MwEP41POqQALY+aut5L47e+HCebxmyhYwhqSHY9n79BQkFQrG2Vz1Hb6bTZr8km+T7dslSL5hky0tF5umVpMA97NOlF0w9jFGIsVd+fLqqkNEoqIBEMWoHNcAt+w0W9C1aMAp5Z6CWkms274KxFAJi3cGIUnLRHTaTvLvqnCTQA25jwvvoT0Z1WqHjyG/w78CStF4Z+bYnI/VgC+QpoXLRgoILL5goKXXVypYT4CV5NS/VvG8DveuNKRD6NRMuZ0/54/X1GD3cT8X95d3j4w9yFFVenggv7IHtZvWqZkArRkRSWueLlGm4nZO47FoYwQ2W6owbC5mmkppoJoUxj059A1jnoDQsB3eN1lyYIAKZgVYrM8ROCE4sfYuGfVRjaYt5HFqQWMWTta+GFNOwvOzA0cl2jkDQszLYjCWkgC4tsGT6zrT9Y4wja/96thEaWXtaHtevjVXLuAHFzDlAWaxaG2gvbLcy2mIw2kBgjSngRsWnrvtNpNoVbiQzCzeC1X5Wjli1i1wWKgY7qx2vjqMo2OJIE5WA7jl61nh97P1lHx1M9o7oLwr+WnHNLp5Z3JbZHyYI0OlxtGcY+I4r/L5hMP6f/TsIH6KuWuM9kz8cveznjUU/3Un0mJM8Z7GRJzf70n24FQ4fXcDASTfs76mg68i9nt9YwToOWxJeSQWlhnOI2czoUlaS5qvISzQm5Y9b+sBSd+XLtZIPMJFcqibbZ4xzByKcJWUdFBtVy+Q9L4sgZsrKM9uRMUoHayolC0GBdnL+r6ooHLiqer2ialNQuc/ag9VUCA3Jk4Awz7z4E2uBBzLs34mBh8TIiCgIrxrmTScDm6yfUhi34PwAwgRf9h46OdQ95Dp673soHMotUmiZGYrir5JeaPsdNH7X7Oq/2GeEic+rQOjcPFHUUyA8jADGbP7YqlKp+XswuPgD&lt;/diagram&gt;&lt;/mxfile&gt;"><defs/><g><path d="M 140 -20 L 300 100 L 140 220 Z" fill="#ffffff" stroke="#000000" stroke-miterlimit="10" transform="rotate(-90,220,100)" pointer-events="all"/><path d="M 130 140 L 310 140" fill="none" stroke="#000000" stroke-miterlimit="10" pointer-events="stroke"/><path d="M 160 100 L 280 100" fill="none" stroke="#000000" stroke-miterlimit="10" pointer-events="stroke"/><path d="M 190 60 L 250 60" fill="none" stroke="#000000" stroke-miterlimit="10" pointer-events="stroke"/><path d="M 80 173.63 L 80 26.37" fill="none" stroke="#000000" stroke-miterlimit="10" pointer-events="stroke"/><path d="M 80 178.88 L 76.5 171.88 L 80 173.63 L 83.5 171.88 Z" fill="#000000" stroke="#000000" stroke-miterlimit="10" pointer-events="all"/><path d="M 80 21.12 L 83.5 28.12 L 80 26.37 L 76.5 28.12 Z" fill="#000000" stroke="#000000" stroke-miterlimit="10" pointer-events="all"/><rect x="10" y="0" width="150" height="20" fill="none" stroke="none" pointer-events="all"/><g transform="translate(17.5,3.5)"><switch><foreignObject style="overflow:visible;" pointer-events="all" width="135" height="12" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility"><div xmlns="http://www.w3.org/1999/xhtml" style="display: inline-block; font-size: 12px; font-family: Helvetica; color: rgb(0, 0, 0); line-height: 1.2; vertical-align: top; width: 136px; white-space: nowrap; overflow-wrap: normal; text-align: center;"><div xmlns="http://www.w3.org/1999/xhtml" style="display:inline-block;text-align:inherit;text-decoration:inherit;white-space:normal;">More specific to use case</div></div></foreignObject><text x="68" y="12" fill="#000000" text-anchor="middle" font-size="12px" font-family="Helvetica">More specific to use case</text></switch></g><rect x="0" y="180" width="150" height="20" fill="none" stroke="none" pointer-events="all"/><g transform="translate(39.5,183.5)"><switch><foreignObject style="overflow:visible;" pointer-events="all" width="70" height="12" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility"><div xmlns="http://www.w3.org/1999/xhtml" style="display: inline-block; font-size: 12px; font-family: Helvetica; color: rgb(0, 0, 0); line-height: 1.2; vertical-align: top; width: 71px; white-space: nowrap; overflow-wrap: normal; text-align: center;"><div xmlns="http://www.w3.org/1999/xhtml" style="display:inline-block;text-align:inherit;text-decoration:inherit;white-space:normal;">More generic</div></div></foreignObject><text x="35" y="12" fill="#000000" text-anchor="middle" font-size="12px" font-family="Helvetica">More generic</text></switch></g><rect x="310" y="180" width="150" height="20" fill="none" stroke="none" pointer-events="all"/><g transform="translate(313.5,183.5)"><switch><foreignObject style="overflow:visible;" pointer-events="all" width="143" height="12" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility"><div xmlns="http://www.w3.org/1999/xhtml" style="display: inline-block; font-size: 12px; font-family: Helvetica; color: rgb(0, 0, 0); line-height: 1.2; vertical-align: top; width: 144px; white-space: nowrap; overflow-wrap: normal; text-align: center;"><div xmlns="http://www.w3.org/1999/xhtml" style="display:inline-block;text-align:inherit;text-decoration:inherit;white-space:normal;">More manual management</div></div></foreignObject><text x="72" y="12" fill="#000000" text-anchor="middle" font-size="12px" font-family="Helvetica">More manual management</text></switch></g><path d="M 380 173.63 L 380 26.37" fill="none" stroke="#000000" stroke-miterlimit="10" pointer-events="stroke"/><path d="M 380 178.88 L 376.5 171.88 L 380 173.63 L 383.5 171.88 Z" fill="#000000" stroke="#000000" stroke-miterlimit="10" pointer-events="all"/><path d="M 380 21.12 L 383.5 28.12 L 380 26.37 L 376.5 28.12 Z" fill="#000000" stroke="#000000" stroke-miterlimit="10" pointer-events="all"/><rect x="290" y="0" width="180" height="20" fill="none" stroke="none" pointer-events="all"/><g transform="translate(301.5,3.5)"><switch><foreignObject style="overflow:visible;" pointer-events="all" width="156" height="12" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility"><div xmlns="http://www.w3.org/1999/xhtml" style="display: inline-block; font-size: 12px; font-family: Helvetica; color: rgb(0, 0, 0); line-height: 1.2; vertical-align: top; width: 157px; white-space: nowrap; overflow-wrap: normal; text-align: center;"><div xmlns="http://www.w3.org/1999/xhtml" style="display:inline-block;text-align:inherit;text-decoration:inherit;white-space:normal;">More automatic management</div></div></foreignObject><text x="78" y="12" fill="#000000" text-anchor="middle" font-size="12px" font-family="Helvetica">More automatic management</text></switch></g><rect x="200" y="35" width="40" height="20" fill="none" stroke="none" pointer-events="all"/><g transform="translate(206.5,38.5)"><switch><foreignObject style="overflow:visible;" pointer-events="all" width="26" height="12" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility"><div xmlns="http://www.w3.org/1999/xhtml" style="display: inline-block; font-size: 12px; font-family: Helvetica; color: rgb(0, 0, 0); line-height: 1.2; vertical-align: top; width: 27px; white-space: nowrap; overflow-wrap: normal; text-align: center;"><div xmlns="http://www.w3.org/1999/xhtml" style="display:inline-block;text-align:inherit;text-decoration:inherit;white-space:normal;">main</div></div></foreignObject><text x="13" y="12" fill="#000000" text-anchor="middle" font-size="12px" font-family="Helvetica">main</text></switch></g></g></svg>
</div>

Recall that the act of building libraries is in service of building a useful program. It would be very nice to build a user-friendly library so that it may be used in the final program.

The solution is hence to build a family of libraries.  Each library builds atop a more fundamental library. As we traverse up the heirarchy of libraries, the use case becomes more narrower. In narrowing the possible use cases, it becomes more feasible to make decisions to perform automatic management for the user.

Note that this is orthogonal to the notion of abstracting. While it is true that as we traverse upwards along the heirarchy of libraries, the libraries become more abstract in general. But this doesn't necessarily have to be the case. Abstracting away the details is an orthogonal issue to be discussed on another day.

There aren't very many "families" of libraries out there in the Go ecosystem. Here are a few:

* [Gonum](https://gonum.org/)
* [Gorgonia](https://gorgonia.org)
* [Go-HEP](https://go-hep.org/)
* [go-gl](https://github.com/go-gl)
* [fyne](https://fyne.io)

I think it's generally a good thing that there aren't many "families" of libraries. Observe the class of problems that these families of libraries solve - Gorgonia solves the deep learning problem. Gonum solves the numeric libraries problem. Go-HEP solves problems in highe energy physics. Fyne solves GUI problems. These are hard problems. I have no doubt that if we look into the Docker or Kubernetes subecosystems we will find families there too.

There is a danger of overengineering when designing families of libraries. That's an article for another day.

## Make the Tradeoff Clear ##

After the decision has been made, the tradeoff should be well documented. Every library at every level should have the tradeoffs listed.

This is especially true of mid-level

# Conclusion #

This article is quite long, at close to 4000 words (if `count-words` is to be trusted). Perhaps it's time to end.

The main point of this article is that library design is hard. There are many considerations to take into account. I list them here:

* What goes into a library?
* What types of library?
* A good library is reliable, having the following features:
    * Does one thing/Provides one resource.
    * Is well-tested
    * Doesn't manage resources for users.
* A good library is easy to use
    * Has good documentation and examples
    * Does not panic
    * Has minimal dependencies
    * Makes the zero value useful
* A good library is generic
    * Functions that accept interfaces and return structs
    * Is extensible.
    * Plays nice with the environment.
* Consider the big picture reason for designing a library.
* Consider the use cases.
* Consider making a family of libraries.
* Make the tradeoff clear.
