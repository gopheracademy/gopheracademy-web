+++
author = ["Matt Layher"]
date = "2015-12-10T08:00:00+00:00"
title = "Go in a Monorepo"
series = ["Advent 2015"]
+++

A "monorepo" is a monolithic code repository which contains many different
projects and libraries.  At DigitalOcean, we have created a monorepo called
`cthulhu` to house all of our Go code.  The purpose of `cthulhu` is to provide
a one-stop shop for our internal Go libraries, services, tools, and third party
dependencies.

[Bryan Liles](https://twitter.com/bryanl) previously wrote a [blog post about
`cthulhu`](https://www.digitalocean.com/company/blog/taming-your-go-dependencies/)
early in 2015.  This post will cover many of the same topics, but will also
detail some of the improvements we have made over the past year to make
`cthulhu` even better.

## Monorepo Structure

`cthulhu` is a single, monolithic git repository.  It contains several important
top-level files and directories:

```
cthulhu
├── .drone.yml
├── analyze.sh
├── docode/
│   └── src/
└── third_party/
    └── src/
```

- `.drone.yml`: configuration for our [drone](https://drone.io/) CI builds
  - Runs automated linting checks, Go tests, and builds and uploads binaries
  to our internal artifacts service
- `analyze.sh`: a shell script used during CI which performs code analysis using
  some of the excellent tools available in the Go ecosystem
  - `goimports`, `go vet`, `golint`, internal linters coming soon!
- `docode/`: used as part of `$GOPATH` to house internal libraries, services,
  and tools
- `third_party/`: used as part of `$GOPATH` to house third party dependencies

By setting the `$GOPATH` environment variable to
`${CTHULHU}/third_party:${CTHULHU}/docode`, we are able to `go get` third party
dependencies and build and test our internal code.

## Internal Code

All of our internal code resides within `${CTHULHU}/docode/`.  We have created
four partitioned areas within this directory, which each serve a different
purpose:

```
docode/
└── src/
    ├── doge/
    ├── exp/
    ├── services/
    └── tools/
```

- `doge/`: **D**igital**O**cean **G**o **E**nvironment: our internal "standard library"
  - contains a wide variety of packages used across many projects
    - structured key/value logging
    - HTTP client which can retry requests based on API output
    - minimal database interaction layers
    - service health and metrics
- `exp/`: a new addition; the home for experimental code
  - used as a proving ground before a service or tool can be deployed to
  production
  - artifacts not generated as part of CI process, to prevent mass deployment
  of experimental code
  - used to alleviate pain of "feature branches" with continuous rebasing on
  master
- `services/`: long-running services which perform heavy lifting tasks for the
DigitalOcean cloud
  - typically HTTP API servers and clients
  - gRPC services being developed at this time
- `tools/`: short-running utilities which perform helpful actions
  - internal fork of `goimports`
  - internal linting tools

## Third Party Code

Third party dependencies reside within `${CTHULHU}/third_party/`.  Because this
directory appears in `$GOPATH` before `${CTHULHU}/docode/`, it is possible to
simply `go get` dependencies to add them to the repository.

To avoid dealing with the potential hassles of git submodules, we simply rename
the `.git` directory in each dependency to `.checkout_git` before committing it
to the repository.  When we want to update a vendored dependency, we rename the
directory back to `.git` and pull the latest changes.  Perhaps there is a better
approach than this technique, but because we rarely need to bump third party
dependencies, this approach has worked well thus far.

## Advantages of a Monorepo

When the idea of `cthulhu` was proposed by
[Antoine Grondin](https://twitter.com/AntoineGrondin), I was initially very
skeptical.  However, after spending a couple of weeks working in a monorepo,
I firmly believe that it is the best possible option when working with a
team of Go developers.

Many of the excellent tools in the Go ecosystem work even better when used in
a monorepo.  For example, `gorename` is incredibly useful.  Just recently, I
was able to use it to rename an identifier from `foo.FooDB` to `foo.DB` in
every single Go source file in the entire repository.  The ability to refactor
en-masse in this fashion is invaluable.  Any changes made will be compiled and
tested across all internal Go projects.

Recently, we have begun to experiment with adding a wide variety of automated
linting checks which are run using `analyze.sh` during CI builds.  Because all
of our Go code resides in `cthulhu`, we can fail any builds which introduce
code that does not meet the standards set by tools like `gofmt` and `go vet`.
In addition, we have started working on internal linting tools using the
excellent `go/ast` package in the standard library.  One of these is a tool
we call `explint`, which ensures that code in our experimental `exp/`
directory cannot be imported by code in `services/` or `tools/`: our production
code directories.

Finally, using a monorepo completely solves the issue of vendoring third party
dependencies.  Every project within `cthulhu` uses the same version of a third
party library, and whenever the library is bumped, every project which makes use
of it is built and tested automatically.  Recently, we tried to make use of the new
`$GO15VENDOREXPERIMENT=1`, but unfortunately, it caused issues with `goimports`,
and we were forced to revert the change.  In the future, we'd like to try it
again, once [`goimports` is updated](https://github.com/golang/go/issues/12278).

## Disadvantages of a Monorepo

While the benefits of using a monorepo do outweigh the costs in our case, it
isn't always the perfect approach.

As `cthulhu` has continued to grow, our CI test durations have grown in
parallel.  More projects and tests are added daily, and some of these require
database integration testing.  We briefly experimented with using Russ Cox's
[gt](https://github.com/rsc/gt) utility, but as of today, it has not been
adopted in `cthulhu`.

Because a monorepo may have many commits per day to many different projects,
"feature branches" can quickly become out of date.  This is why we chose to
adopt the `exp/` experimental directory, instead of requiring experimental
code to live in feature branches.  `exp/` has worked well for us thus far,
but we are still working out a set of guidelines which should be followed when
moving a project from `exp/` to a production directory like `services/` or
`tools/`.

As stated before, we decided to rename the `.git` directory for our third party
dependencies instead of dealing with git submodules.  This approach is likely
not the ideal one, but because we rarely need to touch third party code, it has
not been a major issue thus far.

## Summary

DigitalOcean's monorepo, `cthulhu`, has provided one of the most pleasant
development experiences I have ever had.  It works vastly better than attempting
to keep dependencies up to date across multiple repositories.  It enables
large-scale refactoring using the excellent tools in the Go ecosystem, without
fear of breaking one or more external projects.

If you have any questions or would like clarification on monorepos, feel free
to contact me: "mdlayher" on [Gophers Slack](https://gophers.slack.com)!
