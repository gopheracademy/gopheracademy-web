+++
date = "2016-12-13T00:00:00"
author = [ "Sam Boyer" ]
title = "The Saga of Go Dependency Management"
series = ["Advent 2016"]
+++

# The Saga of Go Dependency Management

The Go community is on the cusp of some major shifts in the way we handle dependencies. These shifts are a long time coming, and involve the work and effort of dozens, if not hundreds, of people. Six to twelve months from now, the Go dependency management landscape is likely to look very different.

For those who haven't been following closely - or, honestly, even for those who have - it can be difficult to keep track of what's going on, and where things are going. So, as part of the Gopher advent series, we thought we'd tie up Go package management past, present, and future in a nice big bow.

## The road so far

Anyone who takes a stroll down the memory lane of Go package management will find ample clumps of ripped-out hair strewn about. The process has been long and frustrating, and there's far too much to cover in detail here. But the history is important to where we are now, so a high-level timeline is worth it.

`go get` was, and is, the only official tooling for retrieving Go code and placing it on disk. It has served Go users and developers alike since the release of Go 1.0. Hand-in-hand with `go get` has been the general recommendation from the Go Team on how to keep everything working well: "write backwards-compatible code." That's a slightly-rephrased version of the Go 1 guarantee. This pairs well with `go get`, as `go get`'s behavior could only be correct if the following were true:

1. All code is always backwards compatible
2. There are no fixed points/releases

_(to be clear, the authors of `go get` do not necessarily believe this is good or right - just that when these things aren't true, it's not `go get`'s problem.)_

In an ecosystem structured around `go get`, the most obvious problem was reproducibility: because `go get` fetches the latest code for any packages not already on disk, it's impossible to guarantee that your users  - or your teammates, or build system - will have the same versions of your software's dependencies that you do. Being the most gaping wound, reproducibility was the main problem that tooling has sought to address, especially early on. [godep](https://github.com/tools/godep) - still the most widely used tool today - emerged just thirteen months after the 1.0 release of Go, with a laser focus on reproducible builds.

`go get` had another noteworthy side effect on the Go mindset: most developers don't make releases of their code. `go get` works directly with git, hg, bzr, and svn repositories and deals exclusively with a repository's "default branch." This behavior makes releases irrelevant in the default case. And when tagging and releasing your code makes no difference to the average user, there's not much incentive to do it.

Over time, [many tools have emerged](https://github.com/golang/go/wiki/PackageManagementTools). Some do more, up to replacing the entire build toolchain. Others might proudly describe themselves as little more than `rsync` wrappers. Divisions widened as tools proliferated, but there was at least one generally clear trend: developers wanted to be able to encapsulate dependencies on a project-by-project basis.

The Go team responded to this trend: in Go 1.5, [the `vendor` directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo/edit) was introduced, providing an official, toolchain-supported mechanism by which we could encapsulate a project's dependencies. But it wasn't until February 2016 - when Go 1.6 was released and `vendor` was enabled by default - that the ball really started rolling.

### 2016: A race against time

There'd been plenty of discussions over the years about Go's issues with dependency management. (We even made a [dedicated mailing list](https://groups.google.com/forum/#!forum/go-package-management) back in October, 2013). But they tended to be fractious, bikesheddy, and ultimately unproductive.

With `vendor` on by default, though, the stakes changed. Before, those of us who worked on tools were mostly trying to improve a bad situation. But the new possibilities opened up by `vendor` created a void - one that would either be filled by good practices, or ones that..."work for now." This lent new urgency to discussion - and tool authors started coming together, hammering out agreement, especially once the costs of such DIY [became evident](https://groups.google.com/d/msg/golang-nuts/AnMr9NL6dtc/UnyUUKcMCAAJ). 

Everything came to a head at Gophercon in July, where upwards of a hundred Gophers assembled to discuss shortcomings with `vendor/` and the ecosystem in general. Go team members, including Rob Pike, also went, which gave a distinctly different tone to the meeting. Up to this point, the Go team had largely treated dependency management as a problem for the community to resolve. After the meeting, though, there was a clear mandate to seriously address the problem.

Mandate notwithstanding, it wasn’t clear to leaders in the community how we’d take the first step. So, Peter Bourgon stepped up to break the collective paralysis: he offered to convene a committee that would work out a proposal for an official, unified tool. That became [a committee plus an advisory group](https://docs.google.com/document/d/18tNd8r5DV0yluCR7tPvkMTsWD_lYcRO7NhpNSDymRr8). Between the two, perspectives and experience from all the major tools have been on hand.

## The package management committee

If you've spent much time in the world of software, "a committee to develop a unified tool" probably sets your mental alarm bells a-jangling. It certainly did for those of us on the committee, and we've tried to manage that risk from our very first meeting.

One step we've taken is to publicly release our work as soon as we felt it could engender productive discussion. That's worked out quite well. The comments we've received have been invaluable, and led us to shift direction more than once. We've also relied heavily on excellent research and feedback from the advisory group.

Here's a quick history of the committee's work from our first meeting in early September:

* To get the conversation started, we each created some imaginary CLI backscroll describing how a user would work with our ideal tool
* We then laid out a set of [major user stories](https://docs.google.com/document/d/1wT8e8wBHMrSRHY4UF_60GCgyWGqvYye4THvaDARPySs)
* Working from the user stories, we explored the [design space](https://docs.google.com/document/d/1TpQlQYovCoX9FkpgsoxzdvZplghudHAiQOame30A-v8) of the problem
* From the user stories and design space, we created a (working) [specification](https://docs.google.com/document/d/1qnmjwfMmvSCDaY4jxPmLAccaaUI5FfySNE90gB0pTKQ/edit#heading=h.4d61hnb1y8gc) for a tool
* We're now iterating on tool implementation, with [gps](https://github.com/sdboyer/gps) as the engine. We’re learning and tweaking as we go, and plan to open up the repository publicly in early January

The [original plan](https://docs.google.com/document/d/18tNd8r5DV0yluCR7tPvkMTsWD_lYcRO7NhpNSDymRr8) was for the committee to deliver a more formal proposal, then begin implementation. However, it became clear over the course of our discussions that a proposal absent implementation would result in exactly the kind of design-by-committee waterfall that we wanted to avoid. So, while there is a proposed spec, please expect it to evolve as we implement and learn.

### One tool, to unite them all

Our goal for the first iteration is a tool that’s minimal, but covers the fundamental requirements. The [spec doc](https://docs.google.com/document/d/1qnmjwfMmvSCDaY4jxPmLAccaaUI5FfySNE90gB0pTKQ/edit#heading=h.4d61hnb1y8gc) has a much more detailed picture of what we believe that entails, but here are some highlights:

* Designating releases (via VCS tags) for importable code will become the norm
* [SemVer](http://semver.org/) will be the standard we follow for release numbering
* Projects (roughly, repositories) will be the unit of versioning; all packages from a given project will have to be at the same version
* Dependencies will be "flattened" - only one version of a given import path allowed per `vendor/` tree; `vendor/` dirs that were committed to upstream deps will be stripped out
* Committing `vendor/` is a choice left to the user/team, but doing so will no longer have [harmful downstream effects](https://groups.google.com/d/msg/golang-nuts/AnMr9NL6dtc/UnyUUKcMCAAJ)
* The tool will use a "two-file system" - a manifest that describes constraints, and a lock that describes a precisely reproducible build
* It will be possible to designate other projects' `main` packages when computing dependencies (crucial for some `go generate` workflows)

Because we see this as merely the first phase of work, the committee has punted on some significant issues. "Punted" here means that we've considered each of these areas, but all we're doing now is writing the tool in a way that minimally constrains our options:

* Any changes to how `GOPATH` works (the tool will operate entirely within the confines of `vendor` as it exists today)
* A central packaging registry (a la [npm](https://www.npmjs.com/))
* Supporting anything other than the upstream source types (git, bzr, hg, svn) that `go get` supports today
* Use metadata from other tools where possible (this is a toss-up, we may actually need it in the first iteration)

The committee's goal is for this to tool to become official. "Official," as in, it's distributed as part of the standard `go` tooling. Of course, that entails code review and approval from the Go team itself. But it would be horribly unwise to make anything official without having solidly kicked its tires beforehand. Thus, we expect there to be at least six months from when we feel the basic requirements of the new tool are met, to when it becomes part of the Go toolchain.

A single dependency management tool is the most important step to healing a fractured ecosystem. This is why we want the tool to _be_ official, capable of replacing existing community tools. Now, deprecating community tools in favor of the new tool will, necessarily, be a choice for each tool's author. But there's general consensus amongst existing tool authors that deprecation in favor of an official tool is ideal. It'll still be a delicate process, of course, but we're doing everything we know how to [avoid making the problem worse](https://xkcd.com/927/).

## What this portends for you, fellow Gopher

While we don't know exactly what the post-official-tool world will look like, we have a pretty good idea:

* Existing projects should migrate from their current tool to the official one. (We hope to make this process as automated and painless as possible)
* Projects should tag official releases according to SemVer. Existing projects may want to retroactively tag releases. (Actually, please start doing this right away!)
* You'll stop monkeying with GOPATH, and instead work on projects with dependency code encapsulated under `vendor/`.
* You can commit `vendor/`, or not - it’s up to you.

We're hoping it will be minimally disruptive to existing workflows. Ideally, it’ll just make a lot of annoying headaches go away. The more elaborate and non-vanilla your current workflows are, however, the more adjusting you’ll likely have to do.

Please do keep in mind that, even though the new tooling will allow the ecosystem to better support backwards-incompatible changes when they do happen, **it is still recommended that you AVOID breaking changes after releasing 1.0.0**. The Go community has made it as far as it has without robust dependency management largely because of this ethos. Having tooling that better organizes the problem _in no way lessens the value of this approach to building software_.

## Where you can help (yes, you! PLEASE DO!)

Once we open up the provisional tool to the public, we're going to need a lot of help, on a number of fronts. Fortunately, we've got a great community, full of industrious gophers - everyone's gonna show up to help. Right?

_Note: We make no pretense that this list is exhaustive!_

* A tool that can statically analyze a project and suggest the next SemVer tag to use for the next release
* People to test out the migration path from existing tools and provide feedback
* People to help with the UX of the tool, particularly when it comes to dealing with failures and conflicts in dependency selection

Comments can be made right here, but may be more productive if channeled to Peter Bourgon, the #vendor channel on the Gopher slack, or the [go-pm](https://groups.google.com/forum/#!forum/go-package-management) mailing list.

