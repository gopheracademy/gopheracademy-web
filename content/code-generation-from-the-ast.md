+++
author = ["Ian Eyberg"]
date = "2015-03-13T12:50:47-07:00"
linktitle = "Code Generation From the AST"
title = "Code Generation From the AST"
+++

[Deprehend](https://github.com/deferpanic/deprehend "Deprehend") is a tool
that auto-generates go src to wrap goroutines with
panic handlers and wrap http handlers for metrics tracking.

![](/postimages/code-generation-from-the-ast/deprehend.png)

Over at [DeferPanic](https://deferpanic.com "DeferPanic") we do a lot of
application monitoring specifically for you, Go engineers.

Go as a language is very opinionated and more importantly, Go engineers
are an anti-magic crowd. To this end, we have exposed most of our
functionality as wrappers around the stdlib. What this means is that you
have to manually add code to use the client library effectively.

Engineers are a lazy bunch - it's why we are engineers. We like to
automate as much as possible.

As more and more companies started integrating with us we kept hearing
more and more of - “You mean I have to wrap every single http handler
and every single goroutine?”

This reminded gophers of their initial reaction to error handling in Go
- "You mean I have to wrap every function with error handling?” :)

For smaller codebases or new projects this isn't a big deal but for
existing codebases this can be a non-trivial amount of work and what
happens if you forget something?

We admit, this is not only a complete pain but it is hard to
scale as you have no way of knowing if new code has been handled correctly
or not - it kinda defeats the purpose of handling unexpected
errors/panics.

So we started hacking out Deprehend.

This tool looks at your source code via the AST and auto-generates the
code for you.

## What Exactly is an AST Anyways?

AST stands for 'abstract syntax tree'. This is a data structure used
by the compiler. It performs a number of functions such as looking at
the correctness of your program and generating the symbol table.

What do we mean by the correctness? Have you ever tried to build your
program via go build and it fails to compile? Maybe like me you use
[Fresh](https://github.com/pilu/fresh "Fresh") and you have a OCD of
hitting save every 2-3 seconds as you are thinking - you'll probably get
a few failed builds here and there as you save if you have only written
half an expression. This is the compiler failing to produce a correct
AST of which it complains and then bails out.

What about the symbol table? The symbol table is another data structure
used by the compiler where every variable you declare is complemented
with meta-data such as its type (int vs string). In some languages you
might see its scope or location in source. In Go we have access to the
receiver, the package name, whether or not it's visible or not, etc.

In many languages the lexer will first come along and tokenize everything in your code.
Then the parser comes along and adds the relationships between the
tokens creating our AST.

For example:

```go
  i := 2 + 3
```

This might be lex'd as:

  identifier (i) operator (:=) operand (2), operator (+), operand (3).

Then the parser might conclude that this forms an assignment expression
where i evaluates to 5 with a resulting tree looking similar to:

![](/postimages/code-generation-from-the-ast/ast.png)

To be clear this is just a general example - not necessarily what you'll
find in any given language.

## The Purpose of it All:

With the structure of an AST we can now determine what exactly the
expression is versus what we (humans) read it to be. This is a very important
distinction to make.

One of the main functions of Deprehend is to find and auto-wrap errors
so we can handle them properly. There's a difference between catching an
error, dealing with an error and observing the (possibly unknown) error.
Our focus is on observability.

So the reader may ask - why don't we just do a regex to find errors?

Well - let's try that.

You might have seen this line before:

```go
if err != nil {
  doBlah()
}
```

So we just need a regex that looks like this right?

```go
if err != nil
```

### Problem 1:

Well, unfortunately, not everyone uses "err" as the variable name. It could
be err1 or errorz or notAnError.

### Problem 2:

What if err is not an error at all? What if it's a string? We simply
don't know by parsing the source with a regex. There is no affirmative
way to understand that without context.

### Problem 3:

Since our code replacement is potentially adding more lines, how do we do
this without accidentally modifying things we should not be modifying?

The answer:

## We dump the AST.

From the top we'll recursively walk the file tree specified on the
command line via [filepath.Walk](http://golang.org/pkg/path/filepath/#Walk).

From there we parse the source code of each file to grab the AST via
[parser.ParseFile](http://golang.org/pkg/go/parser/#ParseFile).

We also grab the type information for our expressions and stuff it
into a map via [types.Info](https://godoc.org/golang.org/x/tools/go/types#Info).

We walk the AST in depth-first order via
[ast.Inspect](http://golang.org/pkg/go/ast/#Inspect). (remember, we are dealing
with trees)

For each node we look for a few things:

  * does this node contain a goroutine?

  * does this node make an assignment? if so is the type an error?


## Errors

If an error is found we do a couple of things.

First, we want to mark the position in the file where we found it as
we'll be modifying the source later. To do that we use
http://golang.org/pkg/go/token/#FileSet.Position .

Then we grab the actual error name to have a reference to it.

We also check to see if a blank identifier, the underscore is being
used. There are lots of times when errors get thrown away because people
don't want to deal with them. If it's a return from strconv.Atoi maybe you don't
care but maybe you should.

For example say you are ingesting a form request. One of the fields
has always been a whole dollar amount so your code looks like:

```go
  s = "10"
  i, _ := strconv.Atoi(s)
```

However, recently, unbeknownst to you, one of the front-end people
with code in a completely different repository on a completely different
team dressed it up with the USD symbol of $.

  Now it looks like:

```go
  s = "$10"
  i, _ := strconv.Atoi(s)
```

Your tests might still pass because your assumptions were that no one
would ever send you anything but a whole dollar amount. However, if you
look at the error you are throwing away you would know why this code is
not working:

  <u>strconv.ParseInt: parsing "$10": invalid syntax</u>

Guess what the value of i is now? That's right - [the zero value](https://golang.org/ref/spec#The_zero_value "the zero value") of an int - 0.

By discovering the places where we purposely or inadvertently throw
away errors we can uncover potential bugs lurking on our production
systems and prevent a rash of angry customers wondering why the service
was free for so long when we should have been charging them $10.

## GoRoutines:

For goroutines we simply insert code to defer any potential panics that might
happen. For better or for worse not everyone has a "let it crash"
mentality and sometimes it's simply not acceptable to kill the entire
daemon because someone missed a divide by zero.

By consulting the AST we can confirm what our code is doing rather than just by
guessing via some regexen.

There's a famous [stackoverflow post](http://stackoverflow.com/questions/1732348/regex-match-open-tags-except-xhtml-self-contained-tags "stackoverflow post")
where someone asks about using regexen to parse html. Similarly, if you are trying to understand source
- don't guess - just look it up in the AST.

For an example of how we rewrite the source grab Deprehend and the
client (to build):

```bash
go get -u github.com/deferpanic/deprehend
go get -u github.com/deferpanic/deferclient
```

Then run it via:

```bash
deprehend /path/to/myproj
```

```go
package main

import (
        "fmt"
)

func bob() {
        fmt.Println("panicing")
        panic("AAAAH")
}

func main() {
        go bob()
}
```

Then we’ll get this output:

```go
package main

import (
        "github.com/deferpanic/deferclient/deferclient"
        "fmt"
)

func bob() {
        fmt.Println("panicing")
        panic("AAAAH")
}

func main() {
        go func(){
                defer deferclient.Persist()
                bob()
        }()
}
```

This shows that we automatically detect goroutines and rewrite the
source to handle any potential panics that might occur within that
goroutine.

This tool is new and ***experimental*** and yes, it might eat your cat - so if
you find any bugs or want to extend functionality please create a pull
request.

We hope this makes developing in Go much easier and makes you, the end
developer, more productive. You can stop guessing whether or not you are
covered and just pump out code instead.

## Looking Towards the Future
I think in the future we might look at making this more general
purpose and including some vim/emacs plugins.
