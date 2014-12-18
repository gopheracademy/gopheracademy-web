+++
author = ["Ben Johnson"]
date = "2014-12-15T08:00:00+00:00"
title = "The Case Against Third Party Libraries"
series = ["Advent 2014"]
+++

## Or how I learned to stop worrying and love versionless package management

If you spend any time on the [golang-nuts][] mailing list you'll learn that the
only thing more contentious than generics is package management. When I first
started writing Go I saw the lack of a "real" package manager as a glaring
oversight but the more I write Go the more I appreciate the simplicity
of `go get`.

It's easy to think of the benefits of versioned packages. They give everyone on
the team a consistent set of code and many times they communicate API changes
through a [semantic versioning scheme][semver]. But package versioning is not
without its downfalls.

In this post we'll look at the options available to application developers,
the responsibilities of library developers, and the side effects of versionless
package management.

[golang-nuts]: https://groups.google.com/forum/#!forum/golang-nuts
[semver]: http://semver.org/


## Just use the standard library

Many third party Go libraries attempt to provide richer implementations of
existing standard library packages. However, many times their extensive feature
set becomes too large to test properly and it adds cognitive overhead to a
project by requiring new developers to understand an additional framework.

Of the [200+ Go web frameworks][web-frameworks] on GitHub, I use none of them.
HTTP-based projects have such a range of requirements and trade offs that I
appreciate the balance of functionality and pragmatism that `net/http` provides.
If I need middleware for my application, it's trivial to write an `http.Handler`
that fits my project perfectly. There is no one-size-fits-all web framework.

Logging is another area where less is more. There are [over 500 Go logging
libraries][logging] on GitHub but I prefer the simplicity of the standard
`log` package. It provides a small, fixed-scope library for printing timestamped
messages to standard error. That's all I need. Adding additional features such
as leveled logging adds complexity to the rest of the code by making developers
choose when to use `DEBUG` instead of `INFO` levels. If it's important enough
to log then you should always log it.

[web-frameworks]: https://github.com/search?l=Go&p=3&q=web+framework&type=Repositories
[logging]: https://github.com/search?l=Go&p=3&q=logging&type=Repositories


## Ctrl-C, Ctrl-V

Staunch supporters of [DRY][] may cringe at this suggestion but many times
copying small bits of code into your project can be a better choice than
importing an entire library. Pulling small functions over allows you to tweak
them to your specific needs and grow them over time, if needed. They also limit
the knowledge needed by other developers on the project by not requiring them
to read another library's full docs to understand the dependency.

Several months ago I released a Go [testing][] library that had no `.go` files
because it was small enough that I wanted users to simply copy the functions out
that they needed. Another good example is when using simple algorithmic
functions such as [consistent hashing][jmphash]. Many times they are less than 
50 lines of code. Please consult the library's license before copying any code.

[dry]: http://en.wikipedia.org/wiki/Don%27t_repeat_yourself
[testing]: https://github.com/benbjohnson/testing
[jmphash]: https://github.com/benbjohnson/jmphash


## Fixed-scope libraries

As much as I try to stick with the standard library, there are some times that
I need to use an third party library. Because `go get` doesn't provide
versioning I look at libraries much differently than when I used something like
[rubygems][].

In Ruby it was common to add libraries to an application with reckless abandon.
A three year old Rails project could easily have 50 dependencies in its
`Gemfile`. Over time I would need to upgrade libraries and, undoubtedly, their
functionality or API would change and conflict with other libraries I used. So
I would have to upgrade those other libraries which could cause more conflicts.

That's when I realized that a library with 30 versions is actually 30 separate
libraries.

In Go, I try to write and use libraries that have a fixed scope so they only
need one version. For example, the [css][] library implements the [W3C's CSS
Syntax Module Level 3][css-syntax] spec so its scope is fixed. Once implemented
and tested, changes and bug fixes are minimal.

Another example is the [Bolt][] library. It's goal is to provide a simple, fast,
transactional key/value store. By limiting its problem space, Bolt is able to
keep a minimal API and be well tested. After nearly six months of running in
a variety of production systems without issue, Bolt was upgraded to version 1.0
and development has stopped. The project is not abandoned -- it's simply
complete. Adding features would compromise the stability of the project. The
next version of Bolt won't be 2.0, it will simply be called something else.

[rubygems]: http://rubygems.org/
[bolt]: https://github.com/boltdb/bolt
[css]: https://github.com/benbjohnson/css
[css-syntax]: http://www.w3.org/TR/css3-syntax/


## High quality builds trust & trust builds community

One unexpected side effect to versionless package management is that I now need
to know and trust the developers who write the libraries I use. Because of that,
I've become much more involved in the Go community than I was in any other
language community.

I regularly read through source code of projects I use to see if the library
maintainer's style and project quality is congruent with what I expect. It's
given me a much better appreciation for the developers in the community. Knowing
someone's code is very personal and I believe this is one small reason why the
Go community has thrived.


## Conclusion

The Go language provides amazingly high quality and well thought out
implementations of common libraries. Many times it's all you need for a project.
There will always be times when you need major functionality that is outside
the scope of the standard library. In those cases I hope you take the time to
know the code and the developer for the library that you're using.

