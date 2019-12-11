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
type sum(a Foo) int { return a.A+a.B }
```

It's the programmer telling the runtime, "When you see some memory space that we have agreed to call `Foo`, return me the `A+B` value in that memory".

Now, it may seem a bit odd for me to talk about the separate conversations as if the are unrelated with one another. Go is a compiled language so the only thing that the programmer is talking to is the compile. That's true. Ultimately all programs are translated into binary code which the processor executes. However, it is still good to separate the notion of a runtime system and a compile time system.

If you look at a snippet of code, and it doesn't do anything by itself, then it's a piece of code that is part of the conversation with the compiler. If you look at a snippet of code and it appears to tell the computer to do something, then it's a conversation with the runtime system via the compiler.

My introduction of the act of programming as a conversatioon with the computer on two fronts is to facilitate a larger discussion on library design.

# Why Libraries #

But first, let's go back to basics and address "why libraries"? Why do we write software libraries? What benefits do we get from software libraries?
