+++
author = ["Nate Finch"]
title = "Make Your Build Better With Mage"
linktitle = "Make Your Build Better With Mage"
date = "2017-12-02T00:00:00"
series = ["Advent 2017"]
+++

Many Go projects can be built using only Go's wonderful built-in tooling.
However, for many projects, these commands may not sufficient. Maybe you want to
use ldflags during the build to embed the commit hash in the binary.  Maybe you
want to embed some files into the binary.  Maybe you want to generate some code.
Maybe you want to run a half dozen different linters.  That's where a build tool
comes into play.

I used to propose using a normal go file to be run with `go run`.  However, then
you're stuck building out a lot of the CLI handling yourself, which is busy
work... no one wants to write *another* CLI parser for their project, plus error
handling, plus handling output etc.

You might consider [make](/advent-2017/make/), which handles the CLI definition
for you, but then you're stuck with writing Bash.  A few months ago, I decided
that neither of these build tool options were sufficient, and decided to make a
third way.  Thus, [Mage](https://magefile.org/) was born.  Mage is a build tool
similar to make or rake, but intead of writing bash or ruby, Mage lets you
write the logic in Go. 

There are many reasons to choose Mage over make. The most important is the
language.  By definition, the contributors to your Go project already know Go.
For many of them, it may be the language they're most comfortable with.  Your
build system is just as important as the thing it's building, so why not make it
just as easy to contribute to?  Why have a second language in your repo if you
can easily avoid it?  Not only is bash an esoteric language to start with, make
piles even more arcane syntax on top of bash.  Now you're maintaining
effectively three different languages in your repo.

One thing I love about Go is how easy it is to make cross platform applications.
This is where Mage really shines.  Although make is installed by default on
Linux and OSX, it is not installed by default on Windows (which, as [Stack
Overflow
notes](https://insights.stackoverflow.com/survey/2017#technologies-and-occupations),
is the most prevalent development OS).  Even if you install make on Windows, now
you have to get bash running, which is non-trivial (yes, you can install the
Windows Subsystem for Linux, but now you're up to a pretty big ask just to build
your Go project).

Mage, on the other hand, is a plain old Go application.  If you have Go
installed (and I presume you do) you can simply `go get
github.com/magefile/mage`.  Mage has no dependencies outside the standard
library, so you don't even have to worry about a dependency manager for it. You
can also download prebuilt binaries from github, if that's preferable.

Once Mage is installed, you use Mage much like make in that you write one or
more scripts (in this case, normal go files that we call magefiles) which mage
then builds and runs for you.  A magefile, instead of having a magic name (like
Makefile), uses the go build tag `//+build mage` to indicate that mage should
read it.  Other than that, there's nothing spceial abouit magefiles and you can
name them whatever you like.

Mage includes all files that have this tag and *only* files that have this tag
in its builds.  This has several nice benefits - you can have the code for your
build spread across any number of files, and those files will be ignored by the
rest of your build commands.  In addition, if you have platform-specific build
code, you can use go's build tags to ensure those are included or excluded as
per usual.  All your existing Go editor integrations, linters, and command line
tools work with magefiles just like normal go files, because they *are* normal
go files.  Anything you can do with Go, any libraries you want to use, you can
use with Mage.

Just like make, Mage uses build targets as CLI commands. For Mage, these targets
are simply exported functions that may optionally take a `context.Context` and
may optionally return an `error`.  Any such function is exposed to Mage as a
build target.  Targets in a magefile are run just like in make

```
//+build mage

package main

// Creates the binary in the current directory.  It will overwrite any existing
// binary.
func Build() {
    print("building!")
}

// Sends the binary to the server.
func Deploy() error {
    return nil
}
```

Running `mage` in the directory with the above file will list the targets:

```
$ mage
Targets:
  build    Creates the binary in the current directory.
  deploy   Sends the binary to the server
```

Mage handles errors returned from targets just like you'd hope, printing errors
to stderr and exiting with a non-zero exit code.  Dependent targets, just like
in make, will be run exactly once and starting at the leaves and moving upward
through a dynamically generated dependency tree.

Mage has a ton of features - running multiple targets, default targets, target
aliases, file targets and sources, shell helpers, and more.  However, for this
blog post I want to dive more into some of the magic behind *how* Mage works,
not just *what* it does.

## How it Works

When you run `mage`, the first thing it has to do is figure out what files it
should read.  It uses the normal `go build` heuristics (build tags, \_platform in
filenames, etc) with one little tweak... normally when you build, go grabs all
files in a directory without tags.  If you specify a tag in the build command it
*adds* any files with that build tag... but it never takes away the files with
no build tags.  This won't work for mage, since I wanted it to only include
files that had a specific tag. This required some hacking.  I ended up copying
the entire go/build package into Mage's repo and inserting some custom code to
add the idea of a required tag... which then excludes any files that don't
explicitly specify that tag.

Once that step is done, we have a list of files with the correct build tags.
Now, what to do with them?  Well, we need to be able to execute the functions
inside them.  To do that, we need to generate some glue code to call the
functions, and build the whole thing into a binary.  Since this process can be
time consuming the first time it's run (on the order of 0.3 seconds on my 2017
MBP), we cache the created binary on disk whenever it's built. Thus, after the
first time it's run, running mage for a project will start instantly like any
normal Go binary (on my machine about 0.01s to print out help, for example).  To
ensure the cached binary exactly matches the code from the magefiles, we hash
the input files and some data from the mage binary itself.  If a cached version
matches the hash (we just use the hash as the filename), we run that, since we
know it must have been built using the exact same code.

If there's no matching binary in the cache, we need to actually do some work. We
parse the magefiles using go/types to figure out what our targets are and to
look for a few other features (like if there's a default target and if there's
any aliases).  Parsing produces a struct of metadata about the binary, which is
then fed into a normal [go
template](https://github.com/magefile/mage/blob/master/mage/template.go#L4)
which generates the func main() and all the code that produces the help output,
the code to determine what target(s) to call, and the error handling. 

This generated code is written to a file in the current directory and then it
and the magefiles are run through a normal execution of `go build` to produce
the binary, then the temp file is cleaned up.  

Now that the glue code and magefiles have been compiled, it's just a matter of
running the binary and passing through the arguments sent to mage (this is the
only thing that happens when the binary is cached).

From there, it's just your go code running, same as always.  No surprises, no
wacky syntax.  The Go we all know and love, working for you and the people on
your team.  

If you want some examples of magefiles, you can check out ones used by
[Gnorm](https://github.com/gnormal/gnorm/blob/master/mage.go),
[Hugo](https://github.com/gohugoio/hugo/blob/master/magefile.go), and hopefully
soon, [dep](https://github.com/golang/dep/pull/1468).

Hop on the #mage channel on gopher slack to get all your questions answered, and
feel free to take a look at our current [issue
list](https://github.com/magefile/mage/issues) and pick up something to hack on.