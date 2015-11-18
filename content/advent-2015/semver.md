+++
author = ["Matt Farina"]
date = "2015-12-02T08:00:00+00:00"
title = "Working with Semantic Versions"
series = ["Advent 2015"]
draft = true
+++

[Semantic Versioning](http://semver.org/) (a.k.a SemVer) has become a popular way
to handle versions. The structure not only allows for incremental releases but
allows people and automation to deduce what those changes mean. This makes SemVer
ideal for a wide range of uses even though they are most well known for package
managers.

Before we look at how we can work with them in Go let's take a look at what a
semantic version looks like.

![Semantic Version](/postimages/advent-2015/semver.png)


The diagram shows the parts of a semantic version. Quite often
you'll see just the first 3 numbers separated by a `.`. A general breakdown of a
semantic version is:

- The major number is incremented when the API to the package or application
  changes in backwards incompatible ways.
- The minor number is incremented when new features are added to the API without
  breaking backwards compatibility. If the major number is incremented the
  minor number returns to 0.
- The patch number is incremented when no new features are added but bug fixes
  are released. If the major or minor numbers are incremented this returns to 0.
- A pre-release is a `.` separated list of identifiers following a `-`. For
  example, `1.2.3-beta.1`. These are optional and are only needed for pre-release
  versions. In this case `1.2.3` would be a release version following a pre-release
  like `1.2.3-beta.1`.
- The final section of information is build metadata. This is a `.` separated
  list of identifiers following a `+`. This is different from pre-release
  information and should be ignored when determining precedence.

While the spec doesn't list anything about a `v` prefix on a semantic version
they are sometimes present. For example, you might see a semantic version as
`v1.2.3`. In this case the `v` should be ignored.

This and more can be found in the [Semantic Versioning Specification](http://semver.org).

Because of the nature of semantic versions it's possible to easily parse them,
sort them, and compare a version against a range or constraint.

## Parsing Semantic Versions

There are a number of packages designed to work with semantic versions. In this
case we're going to use the
[`github.com/Masterminds/semver`](https://github.com/Masterminds/semver) package.
It's built to the spec, supports the optional `v` prefix, provides sorting, and
has the ability to test if a semantic version is within a range or other
constraint. The constraint handling is similar or the same as you'll find in
libraries for other languages including Node.js, Rust, and others.

The following example parses a semantic version and displays an error if it could
not be parsed or prints out the major version if there were no issues.
```go
v, err := semver.NewVersion("1.2.3-beta.1+build345")
if err != nil {
    fmt.Println(err)
} else {
    fmt.Println(v.Major())
}
```
The returned value is an instance of [`semver.Version`](https://godoc.org/github.com/Masterminds/semver#Version)
containing a number of useful methods. If the version wasn't semantic it will
return a
[`semver.ErrInvalidSemVer`](https://godoc.org/github.com/Masterminds/semver#pkg-variables)
error.

The real power isn't in the ability to parse an individual semantic version but
to perform more complicated operations on them.

## Sorting Semantic Versions

When you have a series of versions they may not be in any order. Wouldn't it be
great to sort semantic versions using the `sort` package in the standard library?
With [`github.com/Masterminds/semver`](https://github.com/Masterminds/semver)
you can do just that. For example,

```go
raw := []string{"1.2.3", "1.0", "1.0.0-alpha.1" "1.3", "2", "0.4.2",}
vs := make([]*semver.Version, len(raw))
for i, r := range raw {
    v, err := semver.NewVersion(r)
    if err != nil {
        t.Errorf("Error parsing version: %s", err)
    }

    vs[i] = v
}

sort.Sort(semver.Collection(vs))
```

In this example a series of semantic versions are converted into instances of
[`semver.Version`](https://godoc.org/github.com/Masterminds/semver#Version) and
turned into a [`semver.Collection`](https://godoc.org/github.com/Masterminds/semver#Collection).
A [`semver.Collection`](https://godoc.org/github.com/Masterminds/semver#Collection)
has the methods needed by the `sort` package to reorder the collection. This
is smart enough to get the pre-release information sorted correctly, ignore
metadata, and handle the other elements of sorting.

## Ranges, Constraints, and Wildcards

Does a version sit within a range or other constraint? That's a common question
posed about versions. Those checks are possible. For example,

```go
c, err := semver.NewConstraint(">= 1.2.3, < 2.0.0, != 1.4.5")
if err != nil {
    fmt.Println("Error parsing constraint:", err)
    return
}

v, err := semver.NewVersion("1.3")
if err != nil {
    fmt.Println("Error parsing version:", err)
    return
}

a := c.Check(v)
fmt.Println("Version within constraint:", a)
```

For anyone familiar with the version ranges in other tools you'll know there
are common shortcuts for ranges. Those are available in this `semver` package.
Those include:

- `^1.2.3` which keeps major version compatibility. It's equivalent to
  `>= 1.2.3, < 2.0.0`. This is useful when you need to support an API version.
- `~1.2.3` is to support patch level only changes. It's equivalent to
  `>= 1.2.3, < 1.3.0`. This allows for bug fixes without the addition of new
  features.
- `1.2.3 - 3.4.5` is a range where anything within that range is allowed. It's
  a shortened syntax for `>= 1.2.3, <= 3.4.5`.
- Wildcards using the `x`, `X`, or `*` characters can be used as well. For
  example you can use `2.x`, `1.2.x`, or even just `*`. These can be mixed with
  other comparison operations or be used on their own.

## Go Forth And SemVer

If you have something that could be versioned I would suggest using semantic
versioning. If you're tooling is in Go there are options such as
[`github.com/Masterminds/semver`](https://github.com/Masterminds/semver) that
can make working with the semantic versions easy. If you've not already embraced
semantic versioning now is a great time to get started.
