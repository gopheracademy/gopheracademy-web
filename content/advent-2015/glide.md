+++
author = ["Matt Farina"]
date = "2015-12-18T08:00:00-05:00"
series = ["Advent 2015"]
title = "Manage Dependencies Like Other Languages With Glide"

+++

Managing dependent packages in most of the popular and new languages have common
patterns implemented by package managers. If you
need to manage packages in Python, JavaScript, Rust, Java, Ruby, C# (via .NET),
and numerous other languages it's a similar activity. There are config files,
versions and ranges, pinning to versions, and per project package versions.
These patterns are found in languages using both static and dynamic typing and
work for both compiled and interpreted languages.

With Go 1.5 or newer, the `GO15VENDOREXPERIMENT`, and [Glide](https://github.com/Masterminds/glide)
you can manage Go dependencies the same way as these other languages.

Before we look at how Glide manages packages we need to take a look at the
vendor experiment.

## Vendor Experiment
In Go 1.5 the vendor experiment was introduced. It allows any package to have
a `vendor/` directory. When the compiler looks for an imported package it looks
in the `vendor/` sub-directory for the current directory.
If the package isn't found there it walks up the directory tree looking for
`vendor/` sub-directories that have the package. After exhausting these it looks
in the `GOPATH` and then the `GOROOT` as it did before.

This allows you to have a project structure such as:

```
- $GOPATH/src/github.com/example/foo
  |
  -- main.go
  |
  -- vendor/
       |
       -- github.com/example/bar
```

In this example, when `main.go` imports `github.com/example/bar` the source will
be pulled from the `vendor/` directory instead of the `GOPATH`. The application,
in this case `github.com/example/foo`, still needs to be in the `GOPATH`.

Using the vendor experiment we've learned some gotchas. A couple of them are:

1. If a dependent package is in two `vendor/` directories it's seen as two
   packages. That means if both `github.com/example/foo` and `github.com/example/bar`
   have the same 3rd package in their `vendor/` directory they are seen as two
   different packages which can [cause compatibility problems when trying to
   share instances](https://github.com/mattfarina/golang-broken-vendor).
2. When a dependent package is in multiple different `vendor/` directories it
   will show up in the compiled binary multiple times. There is a danger of
   bloat.

This leads to the conclusion that when using the vendor experiment it's useful
to only have one `vendor/` directory with all the dependencies. Libraries that
use external packages shouldn't store them in a vendor directory. It should be
used by applications. As always, this is the rule and there are exceptions.

In Go 1.5 you opt-in to the experiment by setting the environment variable
`GO15VENDOREXPERIMENT=1`. Running `go env` you can see the current status of it.
In Go 1.6 the vendor experiment will be enabled by default and opt-out.

## Using Glide
[Glide](https://github.com/Masterminds/glide) is a package manager that manages
the packages in a `vendor/` folder. In a `glide.yaml` file you specify your
dependent packages and optionally information such as the version or version
range and version control system. Glide will fetch the packages and make sure
they are on the right version in a manner that enables reproducible builds.

You can install the latest release of Glide via Homebrew or by downloading a
[binary](https://github.com/Masterminds/glide/releases). Using `go get`, which
is an option, you'll get the latest development version rather than the
latest release.

Once Glide is installed the easiest way to get started is to let Glide create
a `glide.yaml` file for you. Run the following command from the root of your project:

```sh
$ glide init
```

Glide will look at the import tree for your codebase to find the imports in the
code and build a `glide.yaml` file. If the project is already managed using
Godep, GPM, or GB the version to use will be pulled from those config files
automatically.

To install the packages it found the first time run:

```sh
$ glide update
```

This will fetch the dependencies, inspect them for any of their dependencies, and
fetch the complete dependency tree. If those projects use Godep, GPM, or GB the
version information for those will be fetched. The dependency tree will be placed
in a `vendor/` folder alongside the `glide.yaml` file.

When an update is run a `glide.lock` file is generated containing the complete
dependency tree. That includes your projects dependencies and any dependencies of
those. This file contains a hash of the `glide.yaml` to make sure it's always in
sync. To update the contents of that run `glide update`.

To restore the dependency tree and set the packages to the locked versions use:

```sh
$ glide install
```

When a valid `glide.lock` file is present it installs from there otherwise it
performs an update. Installing is a faster operation than updates because it doesn't
need to discover the dependency tree and uses concurrency to speedup fetching
packages and setting versions.

The project tree ends up looking like:

```
- $GOPATH/src/github.com/example/foo
  |
  -- main.go
  |
  -- glide.yaml
  |
  -- glide.lock
  |
  -- vendor/
       |
       -- github.com/example/bar
```

Where you manage the `glide.yaml` file Glide manages the `vendor/` directory and
`glide.lock` file.

### Versions In Glide
Glide supports a variety of versions including [semantic versions (SemVer)](http://semver.org/),
[semantic version ranges](https://github.com/Masterminds/semver#basic-comparisons),
tags, branches, and commit ids. Let's look at a simple example:

```yaml
package: github.com/example/foo
import:
- package: github.com/example/bar
  version: ^1.2.0
```

The version on `github.com/example/bar` is a semantic version range (using a
common shorthand) that means `>= 1.2.0, < 2.0.0`. This shorthand is used by a
variety of package managers in different languages. Both the range and shorthand
are supported here.

Glide will find the latest release version that meets this criteria. In the
`glide.lock` file it will pin to the commit id for the chosen version. This
allows for the flexibility of ranges while enforcing versions down to the
commit id for reproducible builds.

### Fetching More Dependencies
Glide has a counterpart to `go get` to fetch new dependencies, add them to the
`glide.yaml` file, and update the dependency tree.

```sh
$ glide get github.com/Masterminds/semver
```

This will add the package and walk the dependency tree to make sure the versions
all resolve.

This command can accept multiple packages:

```sh
$ glide get github.com/Masterminds/semver github.com/Masterminds/vcs
```

It can also handle specifying the version to use:

```sh
$ glide get github.com/Masterminds/semver#^1.0.0
```

### Other Features

Glide has numerous other features such as:

* Working with forks so path rewriting isn't needed.
* Private packages with access restrictions.
* Works with Git, Bzr, Hg, and Svn.
* Plugins, using the Git model
* more...

## Community Welcome

The Glide project is always happy to get feedback or contributions. We Welcome
[issues](https://github.com/Masterminds/glide/issues), [pull requests](https://github.com/Masterminds/glide/pulls),
or you can find is in #masterminds on Freenode IRC.
