+++
author = ["David du Colombier"]
date = "2014-12-16T08:00:00-00:00"
title = "Wrapping Git in rc shell"
series = ["Advent 2014"]
+++

# Wrapping Git in rc shell

## Introduction

When [Rob Pike](http://herpolhode.com/rob/) [announced](https://groups.google.com/forum/#!topic/golang-dev/sckirqOWepg) the migration of Go from [Mercurial](http://mercurial.selenic.com/) and [Rietveld](https://code.google.com/p/rietveld/) to [Git](http://git-scm.com/) and [Gerrit](https://code.google.com/p/gerrit/), like most people, I was pretty enthusiastic. After all, with the increasing number of contributors and development branches, this sounded like a logical evolution.

However, as a maintainer of the Plan 9 port of Go, I felt worried, because Git doesn't work natively on Plan 9, yet.

[Plan 9](http://plan9.bell-labs.com/plan9/) is a distributed operating system built at Bell Labs in the late 80's. On Plan 9, every resource looks like a file system and is accessed by a standard protocol called 9P. Plan 9 is very different from Unix, and though there is a [POSIX environment](http://plan9.bell-labs.com/sys/doc/ape.html), porting a large code base like Git could be very difficult and time-consuming.

Go runs on a variety of hardware and operating systems. There are builders corresponding to every operating systems and architectures supported. Every change on the Go repository is built and run by all the builders. There are two builders running Plan 9 on 386 and amd64.

As Go contributors, we rely on the builders to be able to detect issues and regressions in changes. Moving the Go repository from Mercurial to Git implies that the builders will need to run Git instead of Mercurial. It would be very unfortunate if we couldn't run the Plan 9 builders anymore.

Rob Pike announced on November 14 that the migration will be done in early December. So we basically had two weeks to find a solution.

After looking briefly at the Git [code](https://github.com/git/git/blob/master/) and dependencies, I was only sure about one thing: it was not feasible to port Git to Plan 9 in two weeks.

My first idea was to change the builder program just enough to retrieve the repository archives on the GitHub website. It would have worked, but it would have been painful to maintain our own variant of the builder program.

Other people proposed different solutions like porting [libgit2](https://libgit2.github.com/) to Plan 9, or building something around [gogits](https://github.com/gogits/git). However, writing our own Git tool didn't seem a good solution on the long run. It would be far too time-consuming to maintain.

Incidentally, [Nick Owens](https://github.com/mischief) showed me a tiny script he was using to `go get` GitHub repositories from Plan 9, but it didn't have all the features required to run the Go builder.

I spent a couple of days just thinking about the different possibilities, but in the end, the idea was clear. The best approach on the short-term was to wrap the `git` command around a set of HTTP requests to GitHub.

## How the Go Builder works

To know what our Git wrapper should be able to do, you have to understand how the Go Builder works.

### The Go Dashoard

The Go [Dashboard](http://build.golang.org/) is running on [Google App Engine](https://cloud.google.com/appengine/). It relies on the Go [Watcher](https://godoc.org/golang.org/x/tools/dashboard/watcher) program to be notified about the new commits in the repository.

Periodically, the Go [Builder](https://godoc.org/golang.org/x/tools/dashboard/builder) sends the following `HTTP GET` request to the dashboard:

```
https://build.golang.org/todo?builder=<name>&goHash=&kind=build-go-commit&packagePath=
```

Where `<name>` is the name of the builder, starting with the `goos-goarch` string, like `linux-amd64` or `plan9-386-ducolombier` for example.

Then, the builder receives a JSON response, including the next commit to build. The response looks like:

```
{ "Error" : "",
  "Response" : { "Data" : { "Branch" : "",
          "Desc" : "tag go1.4\n\nLGTM=bradfitz, minux\nR=golang-codereviews, bradfitz, chaishushan, minux\nCC=golang-codereviews\nhttps://codereview.appspot.com/191770043",
          "FailNotificationSent" : true,
          "Hash" : "9ef10fde754f1c5f56cea56e104a871693e520e1",
          "NeedsBenchmarking" : false,
          "Num" : 100486,
          "PackagePath" : "",
          "ParentHash" : "586738173884643d0dca3fd844f73294984d0b9c",
          "PerfResults" : [  ],
          "ResultData" : [  ],
          "Time" : "2014-12-11T05:32:25Z",
          "TryPatch" : false,
          "User" : "Andrew Gerrand <adg@golang.org>"
        },
      "Kind" : "build-go-commit"
    }
}
```

Once the build is completed, the builder sends the result back to the dashboard, with the following `HTTP POST` request:

```
https://build.golang.org/result?builder=<name>&key=<key>&version=1
```

Where `<key>` is the builder key corresponding to the builder.

The JSON request looks like:

```
{ "Builder" : "plan9-386-ducolombier",
  "GoHash" : "",
  "Hash" : "9ef10fde754f1c5f56cea56e104a871693e520e1",
  "Log" : "",
  "OK" : true,
  "PackagePath" : "",
  "RunTime" : 318897794039
}
```

If the build and tests succeed, `OK` is true and the dashboard will display `ok`. Otherwise, `OK` is false, and the dashboard will display `fail` with a link to the build log provided in `Log`.

### Mercurial

When you launch the builder for the first time, it will start by cloning the repository:

```
% hg clone -U https://code.google.com/p/go /tmp/gobuilder/goroot --rev=tip
% cd /tmp/gobuilder/goroot
% hg update default
```

Otherwise, it will use the current contents of the `goroot` directory.

Periodically, the builder will request the dashboard for new commits. When a new commit arrives, the builder will pull new changes from the `goroot` directory:

```
% cd /tmp/gobuilder/goroot
% hg pull
```

Then, the builder will clone the specific commit in its own directory:

```
% hg archive -t files -r 9ef10fde754f1c5f56cea56e104a871693e520e1 /tmp/gobuilder/plan9-386-ducolombier-9ef10fde754f/go
```

Previously, the `hg clone` command was used, but it was replaced by `hg archive`, because it was significantly faster on the slow ARM builders.

Then the builder will run ```all.bash```, ```all.bat``` or ```all.rc```, depending the operating system:

```
% /tmp/gobuilder/plan9-386-ducolombier-9ef10fde754f/go/src/all.rc
```

## Wrapping Git

### From Mercurial to Git

As written earlier, the builder relies on the following Mercurial commands:

* `clone`
* `pull`
* `update`

The `clone` command works a little differently with Git, because you can't clone a specific commit. You have to `clone` the repository, then `checkout` the commit you want.

So, basically, the Git equivalents are:

* `clone`
* `pull`
* `checkout`

### GitHub

[GitHub](https://github.com/) repositories have a very nice property where you can download a `zip` or `tar.gz` archive of any _branch_ or _commit_ of a project over HTTP. It also work with partial hashes.

Luckily, the [Go repository](https://go.googlesource.com/go) is also [mirrored](https://github.com/golang/go/) on GitHub.

For example, on the Go repository, you can download:

```
https://github.com/golang/go/archive/master.tar.gz
https://github.com/golang/go/archive/release-branch.go1.4.tar.gz
https://github.com/golang/go/archive/9ef10fde754f1c5f56cea56e104a871693e520e1.tar.gz
https://github.com/golang/go/archive/9ef10fde754f.tar.gz
```

We will use this feature extensively to implement the Git wrapper.

### Git commands

I could have chosen to write the Git wrapper in Go, but I chose to write it in rc shell instead. The [rc shell](http://plan9.bell-labs.com/sys/doc/rc.html) is the Plan 9 shell. It was written by [Tom Duff](http://www.disneyresearch.com/people/tom-duff/) and originates from [Tenth Edition Research Unix](http://www.cs.bell-labs.com/10thEdMan/). The rc shell is similar to the [Bourne shell](http://www.in-ulm.de/~mascheck/bourne/) but have more features and a much nicer syntax.

#### Clone

We would like to clone a repository from either a remote URL or a local directory.

```
% git clone https://go.googlesource.com/go /tmp/gobuilder/goroot
```

The case of `go.googlesource.com` is particular. We know that all repositories from `https://go.googlesource.com/<repo>` will be mirrored on `https://github.com/golang/<repo>`, so we'll use this property.

So the last command can be translated to:

```
% git clone https://github.com/golang/go /tmp/gobuilder/goroot
```

The clone consists in downloading `https://github.com/golang/go/archive/master.tar.gz` then extracting it in the destination directory.
Since we want to be able to pull from this repository later, we keep the URL of the remote repository in the `.git/config` file.

We can also clone a local directory. In this case, the directory is simply copied. However, contrary to Git, we copy the original remote URL in `.git/config`, so future checkout will be easily feasible.

```
% git clone /tmp/gobuilder/goroot /tmp/gobuilder/plan9-386-ducolombier-9ef10fde754f/go
```

#### Pull

We would like to pull new changes from an existing repository.

```
% cd /tmp/gobuilder/goroot
% git pull
```

Pulling from an existing repository consists in three steps:

* remove all the files, except the `.git` directory
* obtain the URL of the repository from the `.git/config` file
* retrieve and extract the `master.tgz` archive from the previous URL

#### Checkout

We would like to checkout a specific commit or branch from an existing repository.

```
% cd /tmp/gobuilder/plan9-386-ducolombier-1757b5cc7449/go
% git checkout 1757b5cc7449a9883687e78f9be010fc1d876e32
```

The `checkout` command works exactly like `pull`, except it doesn't download the `master` branch archive, but the specified `commit` or `branch`.

### The go tool

After having implemented the most useful Git commands, we have everything we need to run `go get` and `go get -u` successfully on any GitHub repository.

## Conclusion

You can download the [Git wrapper](http://9legacy.org/9legacy/tools/git) tool from the [9legacy](http://9legacy.org/) website.

When the new Go [repository](https://github.com/golang/go) was opened on December 8 and the Git dashboard was online, I launched the `plan9-386` builder which ran successfully on the first time. It was the second builder running after `linux-amd64`, since the Git migration.

There is still room for improvement for this Git wrapper. For example, we would like to use it on other websites than GitHub, which offer a similar interface to download repository archives.

While this Git wrapper is sufficient to use `go get` and run the builder, it would be much more convenient to run the real Git command, especially for us, who are developing in the Go repository.

We would be very pleased if someone could volunteer to work on a port of Git to Plan 9.
