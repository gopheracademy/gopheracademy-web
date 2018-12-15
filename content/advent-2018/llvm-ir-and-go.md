+++
title = "LLVM IR and Go"
date = 2017-12-19T08:00:00Z
author = ["Robin Eklind"]
series = ["Advent 2018"]
+++

<!-- TODO: remember to update date to 2018 -->

In this post, we'll look at how to build Go programs -- such as compilers and static analysis tools -- that interact with the LLVM compiler framework using the LLVM IR assembly language.

_**TL;DR** we wrote a library for interacting with LLVM IR in pure Go, see links to [code](https://github.com/llir/llvm) and [example projects](https://github.com/llir/llvm#users)._

<!-- TODO: remove 3. Building a toy compiler in Go section? If time allows, add it :) -->

1. [Quick primer on LLVM IR](#quick-primer-on-llvm-ir)
2. [LLVM IR library in pure Go](#llvm-ir-library-in-pure-go)
3. Building a toy compiler in Go
4. [Closing notes](#closing-notes)
5. [Further resources](#further-resources)

## Quick primer on LLVM IR

_(For those already familiar with LLVM IR, feel free to [skip this section](#llvm-ir-library-in-pure-go))._

[LLVM IR](https://llvm.org/docs/LangRef.html) is a low-level intermediate representation used by the [LLVM compiler framework](http://llvm.org/). You can think of LLVM IR as a platform-independent assembly language with an infinite number of function local registers.

When developing compilers there are huge benefits with compiling your source language to an intermediate representation (IR)[^1] instead of compiling directly to a target architecture (e.g. x86). As many optimization techniques are general (e.g. dead code elimination, constant propagation), these optimization passes may be performed directly on the IR level and thus shared between all targets[^2].

Compilers are therefore often split into three components, the front-end, middle-end and back-end; each with a specific task that takes IR as input and/or produces IR as output.

* **Front-end**: compiles source language to IR.
* **Middle-end**: optimizes IR.
* **Back-end**: compiles IR to machine code.

![LLVM compiler pipeline](/postimages/advent-2018/llvm-ir-and-go/llvm_compiler_pipeline.png)

### Example program in LLVM IR assembly

To get a glimps of what LLVM IR assembly may look like, lets consider the following C program.

```c
int f(int a, int b) {
	return a + 2*b;
}

int main() {
	return f(10, 20);
}
```

Using [Clang](https://clang.llvm.org/)[^3], the above C code compiles to the following LLVM IR assembly.


```llvm
define i32 @f(i32 %a, i32 %b) {
; <label>:0
	%1 = mul i32 2, %b
	%2 = add i32 %a, %1
	ret i32 %2
}

define i32 @main() {
; <label>:0
	%1 = call i32 @f(i32 10, i32 20)
	ret i32 %1
}
```

By looking at the LLVM IR assembly above, we may observe a few noteworthy details about LLVM IR, namely:

* LLVM IR is statically typed (i.e. 32-bit integer values are denoted with the `i32` type).
* Local variables are scoped to each function (i.e. `%1` in the `@main` function is different from `%1` in the `@f` function).
* Unnamed (temporary) registers are assigned local IDs (e.g. `%1`, `%2`) from an incrementing counter in each function.
* Each function may use an infinite number of registers (i.e. we are not limited to 32 general purpose registers).
* Global identifiers (e.g. `@f`) and local identifiers (e.g. `%a`, `%1`) are distinguished by their prefix (`@` and `%`, respectively).
* Most instructions do what you'd think, `mul` performs multiplication, `add` addition, etc.

<!--
### Control-flow in LLVM IR

To handle control-flow, LLVM IR the notion of [Basic Blocks](https://en.wikipedia.org/wiki/Basic_block) is used.

-->

## LLVM IR library in pure Go

There primarily exist two libraries for working with LLVM IR in Go.

* [llvm.org/llvm/bindings/go/llvm](https://llvm.org/svn/llvm-project/llvm/trunk/bindings/go/README.txt): the official LLVM bindings for the Go programming language.
* [github.com/llir/llvm](https://github.com/llir/llvm): a pure Go library for interacting with LLVM IR.

The official LLVM bindings for Go uses Cgo to provide access to the rich and powerful API of the LLVM compiler framework, while the `llir/llvm` project is entirely written in Go and relies on LLVM IR to interact with the LLVM compiler framework.

This post focuses on `llir/llvm`, but should generalize to working with other libraries as well.

### Why write a new library?

The primary motivation for developing a pure Go library for interacting with LLVM IR was to make it more fun to code compilers and static analysis tools that rely on and interact with the LLVM compiler framework. In part because the compile time of projects relying on the official LLVM bindings for Go could be quite substantial (Thanks to [@aykevl](https://github.com/aykevl) there are now ways to speed up the compile time by using a system-installed version of LLVM[^4]).

Another leading motivation was to try and design an idiomatic Go API from the ground up. The main difference between the API of the LLVM bindings for Go and `llir/llvm` is how LLVM values are modelled. In the LLVM bindings for Go, LLVM values are modelled as a [concrete struct type](https://godoc.org/llvm.org/llvm/bindings/go/llvm#Value), which essentially contains every possible method of every possible LLVM value. My personal experience with using this API, is that it was difficult to know what subsets of methods you were allowed to invoke for a given value. For instance, to retreive the Opcode of an instruction, you'd invoke the [InstructionOpcode](https://godoc.org/llvm.org/llvm/bindings/go/llvm#Value.InstructionOpcode) method -- which is quite intuitive. However, if you happen to invoke the [Opcode](https://godoc.org/llvm.org/llvm/bindings/go/llvm#Value.Opcode) method instead (which is used to retreive the Opcode of constant expressions), you'd get the runtime errors _"cast&lt;Ty&gt;() argument of incompatible type!"_.

The `llir/llvm` library was therefore designed to provide compile time guarantes by relying further on the Go type system. LLVM values in `llir/llvm` are modelled as [an interface type](https://godoc.org/github.com/llir/llvm/ir/value#Value). This approach only exposes the minimum set of methods shared by all values, and if you want to access more specific methods or fields, you'd use a type switch (as illustrated in the [analysis example](#analysis-example-processing-llvm-ir) below).

### Usage examples

Now, lets consider a few concrete usage examples. Given that we have a library to work with, what may we wish to do with LLVM IR?

Firstly, we may want to *parse* LLVM IR produced by other tools, such as Clang and the LLVM optimizer `opt` (see [input example](#input-example-parsing-llvm-ir) below).

Secondly, we may want to *process* LLVM IR to perform analysis of our own (e.g. custom optimization passes) or implement interpreters and Just-in-Time compilers (see [analysis example](#analysis-example-processing-llvm-ir) below).

Thirdly, we may want to *produce* LLVM IR to be consumed by other tools. This is the approach taken when developing a front-end for a new programming language. In this post, we shall look at each of these through the following examples (see [output example](#output-example-producing-llvm-ir) below).

#### Input example - Parsing LLVM IR

```go
package main

import "github.com/llir/llvm/asm"

func main() {
    // Parse LLVM IR assembly file.
    m, err := asm.ParseFile("foo.ll")
    if err != nil {
        panic(err)
    }
    // process, interpret or optimize LLVM IR.

    // Print LLVM IR module.
    fmt.Println(m)
}
```

#### Analysis example - Processing LLVM IR

<!-- TODO: write analysis section -->
TODO

#### Output example - Producing LLVM IR

<!-- TODO: write output section -->
TODO

## Closing notes

The design and implementation of [llir/llvm](https://github.com/llir/llvm) has been guided by a community of people who have contributed -- not only by writing code -- but through shared discussions, pair-programming sessions, bug hunting, profiling investigations, and most of all, a curiosity for learning and taking on exciting challenges.

One particularly challenging part of the `llir/llvm` project has been to construct [an EBNF grammar for LLVM IR](https://github.com/llir/grammar) covering the *entire* LLVM IR assembly language as of LLVM v7.0. This was challenging, not because the process itself is difficult, but because there existed no official grammar covering the entire language. Several community projects have attempted to define a formal grammar for LLVM IR assembly, but these have to the best of our knowledge only covered subsets of the language.

<!--(essentially, cross-reference the C++ code, the LLVM Language Reference and LLVM blog posts, where C++ would be the source of truth unless it contained language ambiguities)-->

The exciting part of having a grammar for LLVM IR is that it enables a lot of interesting projects. For instance, generating syntactically valid LLVM IR assembly to be used for fuzzing tools and libraries consuming LLVM IR (the same approach as taken by [GoSmith](https://github.com/dvyukov/gosmith)). This could be used for cross-validation efforts between LLVM projects implemented in different languages, and also help tease out potential security vulnerabilites and bugs in implementations.

The future is bright, happy hacking!

## Further resources

There is a very well written [chapter about LLVM](http://www.aosabook.org/en/llvm.html) by Chris Lattner -- who wrote the initial design of LLVM -- in the Architecture of Open Source Applications book.

For anyone interested in writing compilers targetting LLVM IR, the [Mapping High Level Constructs to LLVM IR](https://mapping-high-level-constructs-to-llvm-ir.readthedocs.io/) gitbook is warmly recommended.

The [official Go bindings for LLVM](https://godoc.org/llvm.org/llvm/bindings/go/llvm) is a good fit for many projects, as they expose the LLVM C API which is very powerful and also quite stable.

A good complement to this post is the article [An introduction to LLVM in Go](https://blog.felixangell.com/an-introduction-to-llvm-in-go/).




[^1]: The idea of using an IR in compilers is wide spread. GCC uses [GIMPLE](https://gcc.gnu.org/onlinedocs/gcc-4.3.6/gccint/GIMPLE.html), Roslyn uses [CIL](https://www.ecma-international.org/publications/standards/Ecma-335.htm), and LLVM uses [LLVM IR](https://llvm.org/docs/LangRef.html).
[^2]: Using an IR thus reduces the number of compiler combinations required for _n_ source languages (front-ends) and _m_ target architectures (back-ends) from _n * m_ to _n + m_.
[^3]: Compile C to LLVM IR using: `clang -S -emit-llvm -o foo.ll foo.c`.
[^4]: The [github.com/aykevl/go-llvm](https://github.com/aykevl/go-llvm) project provides Go bindings to a system-installed LLVM, which may help speed up compile time a lot.
