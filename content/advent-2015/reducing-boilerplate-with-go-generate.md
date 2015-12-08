+++
author = ["Ernesto Jiménez"]
date = "2015-12-08T08:00:00+00:00"
title = "Reducing boilerplate with go generate"
series = ["Advent 2015"]
+++

Go is an awesome language. It's simple, powerful, has great tooling and
many of really enjoy using it every day. However, as it's common with
strongly typed languages, we write a good deal of boilerplate to
connect things around.

In this post we'll have cover mostly three points:

1. Why can we build tools with Go that will help reduce boilerplate using code generation.
2. What are the building blocks for code generation in Go.
3. Where can we find examples of code generation tools to learn more.

# Why use code generation to reduce boilerplate?

Sometimes we try to reduce boilerplate by using reflection and filling
our projects with methods accepting `interface{}`. However, whenever a
method takes an `interface{}` we are throwing our type safety out of the
window. When using type assertions and reflection the compiler is unable
to check we are passing the right types and we are more open runtime
panics.

Some of the boilerplate code we've got can be mostly inferred from the
code we alerady have in our project. For that, we can write tools that
will read our project's source code and generate the relevant code.

# The building blocks to code generation

## Reading code

The standard library has a wonderful set of packages ready to do most of
the heavy lifting when it comes to reading and parsing code.

* `go/build`: gathers information about a go package. Given a package
  name, it'll return information such as what's the directory containing
the source code for the package, what are the code and test files in the
directory, what other packages it's dependent on, etc.
* `go/scanner` and `go/parser`: read source code and parse it to
  generate an [Abstract Syntax Tree][ast] (AST).
* `go/ast`: declares the types used to represent the AST and includes
  some methods to help walking and modifying the tree.
* `go/types`: declares the data types and implements the algorithms used
  for type-checking Go packages. While `go/ast` contains the raw tree,
  this package does all the heavy lifting to process the AST so you can
  get information about types directly.

## Generating code

When generating code, most projecst just rely on the good old
`text/template` to generate the code.

I recommend starting generated files with a comment clarifying the code
is automatically generated, which program generated it and mentioning it
should not be editign by hand.

```go
/*
* CODE GENERATED AUTOMATICALLY WITH github.com/ernesto-jimenez/gogen/unmarshalmap
* THIS FILE SHOULD NOT BE EDITED BY HAND
*/
```

We can also use the `go/format` package to format our code before
writing it. This package contains the logic used by `go fmt`.

## go generate

Once we start writing tools that generate source code for our programs
two questions appear quickly: at what point in our development process do we
generate the code? how do we keep the generated code up to date?


Since 1.4, the go tool comes with the `generate` command. It allows us
to run the tools we use for code generation with the go tool itself. We
can specify what commands need to be run using special comments within
our source code and `go generate` will do the work for us.

We just need to add a comment with the following format:

```
//go:generate shell command
```

Once you have that, `go generate` will automatically call `command`
whenever it's run.

There's two points that are important to remember:

* `go generate` is meant to be run by the developer authoring the
  program or package. It's never called automatically by `go get`
* You need to have all the tools invoked by `go generate` already
  installed and setup in your system. Make sure you document what tools
  you are going to use and where those tools can be downloaded.

Also, if your code generation tool is within the same repository, I
would recommend calling `go run` from `go:generate`. That way, you can run
`generate` without building and installing the tool manually each time
you change the tool.

# How do you start building your own tools?

The stdlib packages to parse and generate code are great, but their
documentation is huge and it can be quite daunting to make a sense of
how to use the packages just from the docs.

The best thing I did when I got into code generation was to learn about
some of the existing tools. It serves three purposes:

1. You'll get some inspiration about the kind of tools you can build.
2. You'll have the chance to learn from the tools' source code.
3. You can find some of these tools really useful by themselves.

# Projects to learn from

## Generating stubs to implement an interface

Have you ever found yourself copying and pasting the list of methods
defined in an interface you've got to implement?

You can use [`impl`][impl] to generate the stubs automatically. It'll
use the packages in the stdlib to look for the interface and output
methods you must implement.

```bash
$ impl 'f *File' io.ReadWriteCloser
func (f *File) Read(p []byte) (n int, err error) {
    panic("not implemented")
}

func (f *File) Write(p []byte) (n int, err error) {
    panic("not implemented")
}

func (f *File) Close() error {
    panic("not implemented")
}
```

## Generating mocks automatically with mockery

[testify][testify] has a nice [mock][testify-mock] package to

Here's a very simplified example about how to mock a service:

```go
package main

import (
  "testing"

  "github.com/stretchr/testify/mock"
)

type downcaser interface {
  Downcase(string) (string, error)
}

func TestMock(t *testing.T) {
  m := &mockDowncaser{}
  m.On("Downcase", "FOO").Return("foo", nil)
  m.Downcase("FOO")
  m.AssertNumberOfCalls(t, "Downcase", 1)
}
```

The implementation of the mock is pretty straightforward:

```go
type mockDowncaser struct {
  mock.Mock
}

func (m *mockDowncaser) Downcase(a0 string) (string, error) {
  ret := m.Called(a0)
  return ret.Get(0).(string), ret.Error(1)
}
```

However, as we can see from the implementation, it's so straightforward
that the interface definition itself has all the information we need to
generate a mock automatically.

That's what [`mockery`][mockery] does:

```
$ mockery -inpkg -testonly -name=downcaser
Generating mock for: downcaser
```

I always use it with `go generate` to automatically create the mocks for
my interfaces. We just have to add one line of code to our previous
example to have a mock up and running.

```go
package main

import (
  "testing"
)

type downcaser interface {
  Downcase(string) (string, error)
}

//go:generate mockery -inpkg -testonly -name=downcaser

func TestMock(t *testing.T) {
  m := &mockDowncaser{}
  m.On("Downcase", "FOO").Return("foo", nil)
  m.Downcase("FOO")
  m.AssertNumberOfCalls(t, "Downcase", 1)
}
```

Here you can see how everything gets set up once we run go generate:

```
$ go test
# github.com/ernesto-jimenez/test
./main_test.go:14: undefined: mockDowncaser
FAIL    github.com/ernesto-jimenez/test [build failed]

$ go generate
Generating mock for: downcaser

$ go test
PASS
ok      github.com/ernesto-jimenez/test 0.011s
```

Whenever we make a change to an interface we just need to run `go
generate` and the corresponding mock will be updated.

[`mockery`][mockery] is the main reason I started contributing to
[`testify/mock`][testify-mock] and became a maintainer for `testify`.
However, since it was developed before `go/types` was part of the
standard library in 1.5, it's implemented usding the lower level
`go/ast` which makes the code harder to follow and also introduces some
bugs like [failing to generate mocks from interfaces using
composition][mockery-issue].

## gogen experiments

I've open sourced the code generation tools I've been building to learn
more about code generation in my [`gogen`][gogen] package.

It includes three tools right now:

* [goautomock][goautomock]: is similar to mockery but implemented using
  `go/types` rather than `go/ast`, so it worsk with composed interfaces
  too. It's also easier to mock interfaces from the standard library.
* [gounmarshalmap][gounmarshalmap]: takes an struct and generates a
  `UnmarshalMap(map[string]interface{})` function for the struct that
  decodes a map into the struct. It's thought as an alternative to
  [`mapstructure`][mapstructure] using code generation rather than
  reflection.
* [gospecific][gospecific]: is a tiny experiment to generate specific
  packages from generic ones that rely on `interface{}`. It reads the
  generic's package source code and generates a new package using a
  specific type where the generic package used `interface{}`.

# Wrappping up

Code generation is great, it can save us from writing tons of
repetitive code while keeping our programs type safe. We use it
extensively when working on [Slackline][slackline] and we'll probably
use it soon in [testify][testify-codegen] too.

However, remember to ask yourself: is writing this tool
worth the time?

[xkcd][xkcd] wants to help us answering that question:
[![](http://imgs.xkcd.com/comics/is_it_worth_the_time.png)][xkcd]

[go-generate-post]: https://blog.golang.org/generate
[xkcd]: https://xkcd.com/1205/
[slackline]: https://slackline.io
[testify-codegen]: https://github.com/stretchr/testify/pull/241
[impl]: https://github.com/josharian/impl
[testify]: https://github.com/stretchr/testify
[testify-mock]: https://godoc.org/github.com/stretchr/testify/mock
[mockery]: https://github.com/vektra/mockery
[mockery-issue]: https://github.com/vektra/mockery/issues/18
[ast]: https://en.wikipedia.org/wiki/Abstract_syntax_tree
[gogen]: https://github.com/ernesto-jimenez/gogen
[mapstructure]: https://github.com/mitchellh/mapstructure
[goautomock]: https://github.com/ernesto-jimenez/gogen/tree/master/cmd/goautomock/main.go
[gounmarshalmap]: https://github.com/ernesto-jimenez/gogen/tree/master/cmd/gounmarshalmap
[gospecific]: https://github.com/ernesto-jimenez/gogen/tree/master/cmd/gospecific
