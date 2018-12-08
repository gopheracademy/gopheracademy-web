+++
author = ["Aaron Schlesinger"]
title = "Bringing Sanity to your Dependencies with Go Modules and Athens"
linktitle = "golang modules"
date = 2018-12-15T00:00:00Z
series = ["Advent 2018"]
+++

# Bringing Sanity to your Dependencies with Go Modules and Athens

<p align="center"><img src="/images/advent-2018/go-modules-athens/athens-gopher.png" width="360"></p>

_Go Advent, Dec. 15, 2018_

As many of us know, Go version 1.11 introduced [Go Modules](https://github.com/golang/go/wiki/Modules), a brand new dependency management system. 

## A Little Bit About Modules

Before 1.11, our dependencies were a collection of Go packages with a single version number attached to all of them. As packages evolved, their versions were changed. Tools like [Godep](https://github.com/tools/godep), [glide](https://github.com/Masterminds/glide), [gb](https://github.com/constabulary/gb) and [dep](https://github.com/golang/dep) made it really easy for us to fetch all the packages that our apps needed, at the versions we needed.

Then, Go modules came along and changed the way we refer to dependencies. _Modules_ are a collection of Go packages that can have multiple versions attached to them at once. Most commonly, these versions follow the [semver](https://semver.org/) format, which the modules tooling respects.

But, module features don't stop there. Here are a few more cool features that I like:

- It can ingest dependency files from other dependency managers (e.g. `glide.yaml`, `Gopkg.toml`)
- No need to put your code into the `GOPATH` (if you don't want)
- Dependency management is built right into the `go` CLI (i.e. `go get` is now version aware)
- Most importantly for today, modules supports a [download API](https://docs.gomods.io/intro/protocol/)

## What The Modules System Still Lacks

Just like Glide, Dep, and friends, Go modules by default fetches code from version control systems (VCS) directly:

![direct from VCS](/images/advent-2018/go-modules-athens/go-modules-before.png)

The direct-from-VCS approach has served us pretty well since the beginning of versioned dependencies in Go, but it has led to some issues for the community. Here are a few that stand out in my mind:

- When Google Code shut down, we lost some packages that were important to the community
- The [go-bindata package](https://github.com/go-bindata/go-bindata) was removed from GitHub and then pushed to another repository, and everybody who depended it had to create a package alias
- Some packages are massive when you `git clone` them. For example, Kubernetes is over 250M when checked out (using `git clone --depth=1`), so they take a long time to fetch

### GitHub Isn't A CDN

The above issues reveal a common underlying problem. Our module _code_ is not separated from module _artifacts_ that get downloaded by tools. And since we don't have that separation, we have to write our modules using the same systems (the VCS hosts) as we do to serve those very same modules to the Gophers who depend on them.

GitHub and other VCS hosts are great tools for collaborating on code, making changes, and tracking history. But since we've also been relying on them to deliver artifacts, those features start to become limitations. And as the community continues to grow, these limitations will come up more and more often.

We need to move from fetching modules directly from VCS:

![go modules before](/images/advent-2018/go-modules-athens/go-modules-before.png)

To fetching modules from a content delivery network (CDN):

![cdn diagram](/images/advent-2018/go-modules-athens/go-modules-with-cdn.png)

## The HTTP Download API

The download API is the feature of modules that lets us add a layer of indirection between the VCS and the module consumer. That layer of indirection is the CDN in the above diagram. We can design an architecture around the API that lets us use the same VCS tools and workflows we already use, but still serve module artifacts separately.

The new servers for module artifacts are called _module proxies_. They're free to implement the API and efficiently serve module artifacts to consumers, without worrying about the VCS hosting service at all.

Many of us use Git to push our code to GitHub and Git tags or GitHub releases to release a new version of our code. This architecture allows us to keep using those tools and that workflow. 

>Note: if you're not using [semver](https://semver.org) tags to release new versions of your module, you should start doing that now!

And by building _proxy servers_, we don't have to change anything about our workflow to fetch modules. A simple `go get` against a proxy server will fetch a module at the version of your choice.

## Introducing Athens

Now that there's a high level architecture for module proxies, the question is what should a proxy look like? There are two main possibilities:

- A CDN backed by static storage like a cloud blob store or a filesystem
- A server, backed by a database, that fetches modules from an upstream VCS host as needed

In order to not disturb workflows like discussed in the previous section, we need to implement the second one, and Athens is the first open source implementation to do that.

In other words, Athens is an immutable mirror of upstream VCS hosts. When you execute `go get github.com/athens-artifacts/maturelib@v1.0.0` against an Athens proxy, Athens will store [this code](https://github.com/athens-artifacts/maturelib/releases/tag/v0.0.1) in its database forever, regardless of what happens to that code on GitHub.

![athens-diagram](/images/advent-2018/go-modules-athens/athens-diagram.png)

## Experiences with Athens

We've tested Athens out in lots of different situations:

- As a server on `localhost`
- A server just for a team's private code
- A public, global proxy (experimental)

As we've gathered feedback and reviewed our experiences in each situation, we've found some common benefits.

### Builds are Faster

Go modules maintains a local on-disk cache (at `$GOPATH/pkg/mod`), which of course makes builds faster than fetching modules from the network! But, for cold caches (like in CI/CD systems), builds are faster than with fetch-from-VCS situations.

We get speed for a pretty simple reason - Athens serves zip files with module code in them, and no VCS history. It's the same thing as if you ran `git clone --depth 1` and then zipped up the result.

### A Vendor in The Cloud

Athens holds an immutable database, so we're insulated from changes to upstream repositories, even if we delete the `vendor` directory. Simply put, that lets us delete code from our repository while still keeping our builds fast and, more importantly, reliable.

### HTTP Download API

Because the download API is HTTP, we can take advantage of all the battle tested web technologies we know and love. For example, internal teams can load balance across many Athens servers, and we can put a global caching proxy (a proxy in a proxy - inception!) in front of the public proxy.

### Federation

This is my favorite topic because it promotes collaboration and open-ness. The download API is the _lingua franca_ of the Go modules ecosystem and Athens implements it. Anyone can build their own proxy or run Athens themselves and participate in the module proxy ecosystem.

## Where is Athens Now?

The project started as a few lines of code I wrote one evening and has grown to a community of over 50 contributors across 3 continents. Even more importantly to me is that we've maintained a nice, supportive and inclusive place to be, and we all work to keep it that way.

We released a [beta](https://medium.com/project-athens/the-athens-proxy-is-in-beta-bd95abee07db) version about a month ago, and it's being used inside of several organizations. We're also working on our [beta 2 release](https://github.com/gomods/athens/milestone/3) and a [preview public server](https://github.com/gomods/athens/milestone/5) at the moment.

Please follow [@gomodsio](https://twitter.com/gomodsio) on Twitter to keep up with these releases and other Athens news.

## Get Involved!

We spend a lot of time documenting how to set up and use Athens, so we'd love for you to [try it out](https://docs.gomods.io/install/) and tell us what you think. We'd also really appreciate [starring us on GitHub](https://github.com/gomods/athens) so more Gophers can find us.

There are [lots of ways to contribute](https://docs.gomods.io/contributing/community/participating/) to Athens, and we have interesting work to do. Whether you want to "lurk and learn", write code, or anything in between, you're welcome to join us, and we'll help you get started.

The above link has an exhaustive list of ways to get involved, but here are two super easy ways to start:

- Come chat with us in the `#athens` channel on the [Gophers slack](https://gophersinvite.herokuapp.com/)
- Come to one of our [weekly developer meetings](https://aka.ms/athensdevmeeting)

[Absolutely everybody is welcome](https://arschles.com/blog/absolutely-everybody-is-welcome/) to join us, and I hope to see you soon!

❤️ [@arschles](https://twitter.com/arschles)
