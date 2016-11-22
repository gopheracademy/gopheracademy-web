+++
author = ["Matt Layher"]
date = "2016-12-08T08:00:00+00:00"
title = "Contributing to the Go project"
series = ["Advent 2016"]
+++

Contributing to the Go project can seem overwhelming, especially at first.
The official [Contribution Guidelines](https://golang.org/doc/contribute.html)
document is rather lengthy, but after working through the initial Google CLA
and Gerrit authentication process, it becomes much easier to contribute to
the project.

This post will attempt to demystify the process behind contributing to the
Go project, in an effort to encourage all Gophers to try to tackle an issue
or solve a bug upstream for the benefit of others.

Please note: this post attempts to simplify the steps laid out in the
official Go [Contribution Guidelines](https://golang.org/doc/contribute.html)
document, but it is entirely possible that it may contain incorrect information
or become outdated over time.  When in doubt, refer to the Contribution
Guidelines or consult one of the [many help forums](https://golang.org/help/)
available for Go.

## Gerrit Registration and Google CLA

The Go project uses a system called [Gerrit](https://www.gerritcodereview.com/)
for code review.  Gerrit is where CLs (change lists; akin to a GitHub pull
request) are reviewed and submitted for inclusion in the upstream Go
repositories.

To authenticate to Gerrit, a Google account must be used.  Visit
[go.googlesource.com](https://go.googlesource.com/), click "Generate Password"
on the top right menu, and follow the instructions.  Take note of the Google
account selected, as it must be used for all remaining steps.

Next, [register with Gerrit](https://go-review.googlesource.com/login/) using
the same Google account previously selected.

Finally, you (or your organization) must agree to a Google Contributor
License Agreement (CLA).  If you are the copyright holder for your contributions,
you must agree to the
[individual CLA](https://developers.google.com/open-source/cla/individual).

If your organization is the copyright holder for your contributions, your
organization must agree to the [corporate CLA](https://developers.google.com/open-source/cla/corporate).

## git-codereview setup

It is recommended to install the `git-codereview` tool to simplify the
contribution process.  This tool is not strictly necessary, but this guide
will assume that you are using it to submit your contributions.

```
$ go get -u golang.org/x/review/git-codereview
```

Once `git-codereview` is installed, it is recommended to set up aliases for
its commands in your Git configuration file (typically `~/.gitconfig`).
This guide will also assume that these aliases are in place.

```
[alias]
	change = codereview change
	gofmt = codereview gofmt
	mail = codereview mail
	pending = codereview pending
	submit = codereview submit
	sync = codereview sync
```

## Finding an issue to work on

With the previous steps completed, you are now ready to begin contributing to
the Go project.  To find an issue to work on, browse through the
[Go issue tracker](https://golang.org/issues).  Specifically, searching for
["open" issues with the "HelpWanted" label](https://github.com/golang/go/issues?q=is%3Aopen+is%3Aissue+label%3AHelpWanted)
can be a great starting point.

Once you've found an issue you'd like to work on, leave a comment stating
that you'd like to look into solving an issue.  This helps prevent
duplication of work due to lack of communication.

There are many different repositories that belong under the Go project
umbrella, in addition to the main "go" repository.  Visit
[go.googlesource.com](https://go.googlesource.com/) to browse a list of
all available repositories.

For purposes of demonstration, we will clone the "go" repository, containing
the `go` tool, standard library, and runtime.

```
$ git clone https://go.googlesource.com/go
```

## Making a contribution

Before making any changes, it is a good idea to sync your local repository
with the upstream repository.

```
$ git sync
```

Begin working on your change.  Remember to always use tools like `go fmt`
and `go vet` on your code, and try to follow the conventions already in
place in code you are working on.

Once your change is complete (and tests have been written!), run tests
for the entire tree to ensure your changes don't break other packages
or programs.

```
$ cd go/src
$ ./all.bash
```

When `all.bash` completes, it should print the output `ALL TESTS PASSED`.
At this point, you are ready to submit your change for review.

## Submitting your contribution

Use typical `git` commands like `git add` and `git rm` to stage your
changes.  When ready to submit your changes, think of a meaningful branch
name and run:

```
$ git change <branch>
```

This will open a commit message file in your editor (using `$EDITOR`).

There are some conventions that should be followed for commit messages,
including:

- commit message should be prefixed with package name
- a one-line summary of the change
- if needed, a detailed description of the change (written in complete
sentences with proper punctuation)
- the phrase "Fixes ###", where ### is the Go issue you are resolving

An example of a commit message in this style, from the Contribution
Guidelines:

```
math: improve Sin, Cos and Tan precision for very large arguments

The existing implementation has poor numerical properties for
large arguments, so use the McGillicutty algorithm to improve
accuracy above 1e10.

The algorithm is described at http://wikipedia.org/wiki/McGillicutty_Algorithm

Fixes #159
```

If you wish to make further changes, use normal `git` commands and
just run `git change` again to amend your commit.  It is very common that
a given code review will often go through several rounds of feedback and
requested changes.  Don't be intimidated when asked to make changes
to your contribution!

Finally, submit your change to Gerrit by running:

```
$ git mail
```

The output of `git mail` will print a link to where your change can be
found, such as:

```
remote: New Changes:
remote:   https://go-review.googlesource.com/99999 math: improved Sin, Cos and Tan precision for very large arguments
```

## Gerrit code review

Now that your change has been submitted, it can be reviewed by a member
of the Go team and others in the community.  Code review comments may
be addressed by amending your changes using the process above.

In addition, this is the stage where the "TryBots" are run, by assigning
the label `Run-TryBot +1`.  This starts an automated testing process which
builds your change against the entire Go tree using a multitude of
different operating systems and CPU architectures.

At this point, any number of actions can take place with your change.
Typically, a member of the Go team will comment with a `Code-Review`
label, which can be interpreted as follows:

- `-2`: I am strongly against this change and will almost certainly not
be persuaded otherwise
- `-1`: I disagree with this change, but could be persuaded otherwise
- `+1`: this change looks good to me, but someone else must approve
- `+2`: this change looks good to me, and is ready for submission once
the TryBots indicate a change is OK

Depending on the `Code-Review` label applied and the ensuing discussion,
your change may or may not be accepted into the project.

In particular, if you submit a change with no issue filed and no
discussion beforehand, you can be almost certain it will be rejected.

## Summary

In summary, contributing to the Go project can be intimidating, but is
an excellent way to give back to the Go community and to gain experience
working on a large open source project, used by thousands of people all
over the world.

If your first change is rejected, be polite and ask for clarification
on why, if needed.  If you disagree with a comment, state your case concisely
and make sure that no misunderstandings take place.

If you have any questions or would like to hear about my own experiences
contributing to the Go project, feel free to contact me: "mdlayher" on
[Gophers Slack](https://gophers.slack.com/)!
