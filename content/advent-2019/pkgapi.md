+++
author = ["Chewxy"]
title = "Some Thoughts on Library Design"
linktitle = "Short Title (use when necessary)"
date = 2014-02-06T06:40:42Z
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
```

It's the programmer telling the runtime, "When you see some memory space that we have agreed to call `Foo`, return me the `A+B` value in that memory".

Now, it may seem a bit odd for me to talk about the separate conversations as if the are unrelated with one another. Go is a compiled language so the only thing that the programmer is talking to is the compile. That's true. Ultimately all programs are translated into binary code which the processor executes. However, it is still good to separate the notion of a runtime system and a compile time system.

If you look at a snippet of code, and it doesn't do anything by itself, then it's a piece of code that is part of the conversation with the compiler. If you look at a snippet of code and it appears to tell the computer to do something, then it's a conversation with the runtime system via the compiler.

My introduction of the act of programming as a conversatioon with the computer on two fronts is to facilitate a larger discussion on library design.

# Why Libraries #

But first, let's go back to basics and address "why libraries"? Why do we write software libraries? What benefits do we get from software libraries?

First, note that I am using the term "libraries" instead of "packages", "modules" or "repository". Despite being used interchangably in my mind there are very subtle differences. Allow me to explain.

## A Repository ##

A repository is a collection of files containing soure code. They are typically arranged within a directory in the file system.

## A Library ##

A library is a an abstract notion of shared source code. Let's say I have a file (let's call it `lib.go`) somewhere in my repositories that look like this in its entirety, then this is a library:

```
func MaxInt(a, b int) int {
     if a > b {
     	return a
     }
     return b
}
```

The astute reader will note that having `lib.go` will cause any Go project to have a compilation failure. All `.go` files must declare at the very top, what package it is used for. The declaration `package foo` is a conversation to the compiler, telling the compiler to include the file. We will return shortly to packages.

The example above may give the reader a mistaken definition of a library - that a library is a file in a package. It is not. I show this by the following contrived example.

Assume now that `lib.go` is in a repository `common`. I start a new Go package (called `foo`), and I place it in a directory called `github.com/myusername/bar`. I copy `lib.go` from  `common` to `bar`, and rename the file in `bar` as `lib_bar.go`. Now I edit `lib_bar.go` and prepend the declaration `package foo` at the top so that the complete `lib_bar.go` is as follows:

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

It is in this sense that I use the word "library",

## A Package ##

In general, libraries of source code in Go are arranged in packages and modules. A package may span multiple `.go` files.
