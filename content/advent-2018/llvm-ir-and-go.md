+++
title = "LLVM IR and Go"
date = 2017-12-19T08:00:00Z
author = ["Robin Eklind"]
series = ["Advent 2018"]
+++

<!-- TODO: remember to update date to 2018 -->
<!-- TODO: add table of contents? -->

In this post, we'll look at how to build Go programs -- such as compilers and static analysis tools -- that interact with the LLVM compiler framework using the LLVM IR assembly language.

_**TL;DR** we wrote a library for interacting with LLVM IR in pure Go, see links to [code](https://github.com/llir/llvm) and [example projects](https://github.com/llir/llvm#users)._

1. [Quick primer on LLVM IR](#quick-primer-on-llvm-ir)
2. [LLVM IR library in pure Go](#llvm-ir-library-in-pure-go)
3. Building a toy compiler in Go
4. Further resources

## Quick primer on LLVM IR

_(For those already familiar with LLVM IR, feel free to [skip this section](#llvm-ir-library-in-pure-go))._

[LLVM IR](https://llvm.org/docs/LangRef.html) is a low-level intermediate representation used by the [LLVM compiler framework](http://llvm.org/). You can think of LLVM IR as a platform-independent assembly language with an infinite number of function local registers.

When developing compilers there are huge benefits with compiling your source language to an intermediate representation (IR)[^1] instead of directly compiling to a target architecture (e.g. x86). As many optimization techniques are general (e.g. dead code elimination, constant propagation), these optimization passes may be performed directly on the IR level and thus shared between all targets[^2].

Compilers are therefore often split into three components, the front-end, middle-end and back-end, each with a specific task that takes IR as input and/or produces IR as output:

* **Front-end**: Compiles source language to IR.
* **Middle-end**: Optimizes IR.
* **Back-end**: Compiles IR to machine code.

![LLVM compiler pipeline](/postimages/advent-2018/llvm-ir-and-go/llvm_compiler_pipeline.png)

### Example Program in LLVM IR Assembly

To get a glimps of what LLVM IR assembly may look like, lets consider the following C program:

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

By looking at the LLVM IR above, we may observe a few noteworthy details about LLVM IR, namely:

* LLVM IR is statically typed (i.e. 32-bit integer values are denoted with the `i32` type).
* Local variables are scoped to each function (i.e. `%1` in the `@main` function is different from `%1` in the `@f` function).
* Unnamed (temporary) registers are assigned local IDs (e.g. `%1`, `%2`) from an incrementing counter in each function.
* Each function may use an infinite number of registers (i.e. we are not limited to 32 general purpose registers).
* Global identifiers (e.g. `@f`) and local identifiers (e.g. `%a`, `%1`) are distinguished by their prefix (`@` and `%`, respectively).
* Most instructions do what you'd think, `mul` performs multiplication, `add` addition, etc.

### Control-flow in LLVM IR

To handle control-flow, LLVM IR the notion of [Basic Blocks](https://en.wikipedia.org/wiki/Basic_block) is used.

<!--A common architecture for building compilers is to devide it into three parts, the front-end, middle-end and back-end. The front-end of a compiler is responsible for parsing the source code of the language being compiled, translating this code into an Abstract Syntax Tree (AST)-->

## LLVM IR library in pure Go

There primarily exist three libraries for working with LLVM IR from Go.

* llvm.org/llvm/bindings/go/llvm
   - LLVM bindings for the Go programming language.
* github.com/aykevl/go-llvm
   - Go bindings to a system-installed LLVM.
* github.com/llir/llvm
   - Pure Go library for interacting with LLVM IR.

This article focuses on `llir/llvm`, but should generalize to working with the other libraries as well.

Fork of x/tools `strings` tool, to do the inverse: `string2enum`.

```go
package main

import "github.com/llir/llvm/asm"

func main() {
    // Parse LLVM IR assembly file.
    m, err := asm.ParseFile("foo.ll")
    if err != nil {
        panic(err)
    }
    // Print LLVM IR module.
    fmt.Println(m)
}
```

### Libraries for interacting with LLVM IR in Go

### Why develop a pure Go library for interacting with LLVM IR?

The motivation behind developing a pure Go library for interacting with LLVM IR was to make it more fun to code compilers, interpreters and static analysis tools that rely on and interact with the LLVM compiler framework through LLVM IR.

### Closing notes

The design and implementation of [llir/llvm](https://github.com/llir/llvm) has been guided by a community of people who have contributed -- not only by writing code -- but through shared discussions, pair-programming sessions, bug hunting, profiling investigations, and most of all, a curiosity for learning and taking on exciting challenges.

One particularly challenging part of the `llir/llvm` project has been to construct [an EBNF grammar for LLVM IR](https://github.com/llir/grammar) covering the *entire* LLVM IR assembly language as of LLVM v7.0. This was challenging, not because the process itself is difficult, but because there existed no official grammar covering the entire language. Several community projects have attempted to define a formal grammar for LLVM IR assembly, but these have to the best of our knowledge only covered subsets of the language.

The exciting part of having a grammar for LLVM IR is that it enables a lot of interesting projects. For instance, generating syntactically valid LLVM IR assembly to be used for fuzzing tools and libraries consuming LLVM IR (the same approach as taken by [GoSmith](https://github.com/dvyukov/gosmith)). This could be used for cross-validation efforts between LLVM projects implemented in different languages, and also help tease out potential security vulnerabilites and bugs in implementations.

The future is bright, happy hacking!

<!--(essentially, cross-reference the C++ code, the LLVM Language Reference and LLVM blog posts, where C++ would be the source of truth unless it contained language ambiguities)-->

### Further resources

There is a very well written [chapter about LLVM](http://www.aosabook.org/en/llvm.html) by Chris Lattner -- who wrote the initial design of LLVM -- in the Architecture of Open Source Applications book.

For anyone interested in writing compilers targetting LLVM IR, the [Mapping High Level Constructs to LLVM IR](https://mapping-high-level-constructs-to-llvm-ir.readthedocs.io/) gitbook is warmly recommended.

The [official Go bindings for LLVM](https://godoc.org/llvm.org/llvm/bindings/go/llvm) is a good fit for many projects, as they expose the LLVM C API which is very powerful and also quite stable.

---

[^1]: The idea of using an IR in compilers is wide spread. GCC uses [GIMPLE](https://gcc.gnu.org/onlinedocs/gcc-4.3.6/gccint/GIMPLE.html), Roslyn uses [CIL](https://www.ecma-international.org/publications/standards/Ecma-335.htm), and LLVM uses [LLVM IR](https://llvm.org/docs/LangRef.html).
[^2]: Using an IR reduces the number of compiler combinations required for _n_ source languages (front-ends) and _m_ target architectures (back-ends) from _n * m_ to _n + m_.
[^3]: Compile C to LLVM IR using: `clang -S -emit-llvm -o foo.ll foo.c`.
