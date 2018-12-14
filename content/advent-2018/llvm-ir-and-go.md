+++
title = "LLVM IR and Go"
date = 2017-12-19T08:00:00Z
author = ["Robin Eklind"]
series = ["Advent 2018"]
+++


1. Quick primer on LLVM IR
2. LLVM IR libraries in Go
3. Building a toy compiler in Go
4. Inspirational projects
5. Further resources


<!-- TODO: remember to update date to 2018 -->
<!-- TODO: add table of contents? -->

In this post, we'll look at how to build Go programs -- such as compilers and static analysis tools -- that interact with the LLVM compiler framework using the LLVM IR assembly language.

# Quick primer to LLVM IR

_(For those already familiar with LLVM IR, feel free to skip to section [Foobar](#foobar))._

[LLVM IR](https://llvm.org/docs/LangRef.html) is a low-level intermediate representation used by the [LLVM compiler framework](http://llvm.org/). You can think of LLVM IR as a platform-independent assembly language with access to an infinite number of function local registers. The benefit of developing a compiler which targets an intermediate representation (IR) instead of a specific hardware instruction set -- such as x86 or ARM -- is that the tools and algorithms used for optimizations and analysis may be developed once for the IR, instead of once per hardware architecture and source language. _n + m_ instead fo _n * m_.

To get a glimps of what LLVM IR assembly may look like, lets consider the following C program:

```c
int f(int a, int b) {
	return a + 2*b;
}

int main(int argc, char **argv) {
	return f(10, 20);
}
```

Using [Clang](https://clang.llvm.org/)[^1], the above C code compiles to the following LLVM IR assembly.


```llvm
define i32 @f(i32 %a, i32 %b) {
; <label>:0
	%1 = mul i32 2, %b
	%2 = add i32 %a, %1
	ret i32 %2
}

define i32 @main(i32 %argc, i8** %argv) {
; <label>:0
	%1 = call i32 @f(i32 10, i32 20)
	ret i32 %1
}
```

By looking at the LLVM IR above, we may observe a few noteworthy details about LLVM IR. It is statically typed (e.g. 32-bit integer values are denoted with the type `i32`), and uses in each function an incrementing counter of local IDs (e.g. `%1`) to assign temporary values to unnamed registers.

, is that tools may be written to optimize and analyze this IR instead of a specific hardware architecture -- such as Intel or ARM --

LLVM IR is often used to build compilers, and this will be the focus of this post, specifically how one may go about developing a compiler in Go.

 A common architecture for building compilers is to devide it into three parts, the front-end, middle-end and back-end. The front-end of a compiler is responsible for parsing the source code of the language being compiled, translating this code into an Abstract Syntax Tree (AST)

## Further reading

There is a very well written [chapter about LLVM](http://www.aosabook.org/en/llvm.html) in the Architecture of Open Source Applications book, by Chris Lattner who wrote the initial design of LLVM.

<!-- ![noice reduction](/postimages/advent-2018/llvm-ir-and-go/headphones.jpg) -->

## The Evolution of an LLVM IR library in pure Go

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

## Libraries for interacting with LLVM IR in Go

## Why develop a pure Go library for interacting with LLVM IR?

The motivation behind developing a pure Go library for interacting with LLVM IR was to make it more fun to code compilers, interpreters and static analysis tools that rely on and interact with the LLVM compiler framework through LLVM IR.

The [official Go bindings for LLVM](https://godoc.org/llvm.org/llvm/bindings/go/llvm) is a good fit for many projects, as they expose the LLVM C API which is very powerful and also quite stable.

## Further resources

Anyone interested in writing compilers targetting LLVM IR, I can warmly recommend checkout out the gitbook [Mapping High Level Constructs to LLVM IR](https://mapping-high-level-constructs-to-llvm-ir.readthedocs.io/).

## Inspiration

Inspiration for the API was taken from github.com/bongo227/goory.

[^1]: Compile C to LLVM IR using: `clang -S -emit-llvm -o foo.ll foo.c`.
