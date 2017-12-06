+++
author = ["Aditya Mukerjee"]
title = "Managing Dependencies, Forks, and Code Patches with Dep"
linktitle = "Short Title (use when necessary)"
date = 2017-12-06T00:00:00Z
+++


A few months ago, we switched over to using [dep](https://github.com/golang/dep/) to manage dependencies for [Veneur](https://veneur.org).
 
We were excited to discover that dep provided a straightforward workflow for a longstanding problem in Go development: managing forks of dependencies and contributions to upstream projects.
 
This is a common problem with dependency management. Some tools attempt to provide workflows for managing it, and in my experience, the workflow dep has chosen is the cleanest I’ve seen.
 
 
I often seen this problem manifest in one of two ways:
 
 
### Scenario #1: Contributions to upstream projects

```go
package b

import “github.com/c/d”
```
 
github.com/a/b imports github.com/c/d. In the process of developing a/b, we discover we need to extend c/d to support a new feature. We provide a pull request to the c/d project, but we don’t know how long it will take for c/d to merge our changes – it could be a few hours, or a couple of months. Because we want to begin using the new feature in a/b immediately, we have to fork the project to a repository we control and rewrite all of our imports from `github.com/c/d` to `github.com/a/d`. Because some of our dependencies also pull in `c/d`, we end up having to rewrite imports for projects inside our vendor directory too.
 
 
### Scenario #2: Known bugs in upstream projects
 
github.com/a/b imports github.com/c/d. For several months, we’ve been building a/b against the c/d commit specified by the SHA prefix `afadedface1`. We decide to upgrade to a newer version, specified by the SHA prefix `badcab2`. After a few days, we notice some sporadic problems – under a rare set of conditions, a new bug introduced in `badcab2` can cause a race condition, leading to silent failures. We report the bug upstream, but the fix is complicated. It will require several months for upstream maintainers to implement without causing problems for other users of c/d. We decide to revert back to `afadedface1` and wait for the fix before upgrading again.
 
 
A few weeks later, another developer (“Alice”) decides to upgrade the `c/d` package to the latest commit - `cafebead3` - in order to pull in an unrelated feature that she needs. The fix for the networking bug is still in progress, so `cafebead3` still doesn’t contain the fix for the networking bug, and upgrading to `cafebead3` will reintroduce the same bug that upgrading to `badcab2` did. But Alice was out of the office during the week that `badcab2` was tested, so she isn’t aware of the rare bug. She has no reason to believe that this change could be problematic. The file that specifies the version requirements for `a/b` doesn’t contain any comments or self-documentation indicating that `c/d` should not be upgraded.  She follows the standard process for upgrading dependencies – updating the SHA in the requirements file – and in doing so, reintroduces the bug that had previously been discovered.
 
 
 
### What went wrong?
 
The problem in the first scenario is that we want to create a temporary fork, and we’re using a workflow – rewriting imports – that is designed for permanent, hard forks. When we create the github.com/a/d repository, we don’t want to treat it as a different package from github.com/c/d. It’s only a separate repository because we don’t have write access to c/d, so we can’t create feature branches there directly. (If we had no intention of submitting our patch upstream, a hard fork with rewritten imports would be the right solution.)
 
 
The second scenario is more complicated. It’s a problem that arises from multiple sources: communication gaps, cross-organization development, and an inability to automatically surface information at the right times.
 
 
There’s a difference between pinning to a SHA because you know it works and pinning to a SHA because you know that other versions won’t work. Upgrading software always carries the necessary risk of unknown problems, but there’s no reason we should be exposed to the unnecessary risk of known problems. Alice was able to reintroduce the bug accidentally because the amount of friction associated with upgrading a dependency is identical whether or not known problems exist. We need a way to increase the friction of upgrading a dependency when newer versions are known to cause problems.
 
I’m illustrating these examples with git SHAs, but keep in mind: these same problems can happen even if a/b and b/c are using semantic version schemes. New features that are backwards-compatible can be introduced as minor or point releases, and it’s not unheard-of for minor or point releases to introduce inadvertent bugs, either. Any version management scheme that allows flexibility in versions, instead of explicitly pinning to a specific SHA, runs the risk of introducing bugs when upgrading versions. Any version management scheme that does pin to a specific SHA requires a way to document problems with versions in a seamless way.
 
Unfortunately, many do not. Many vendoring tools use JSON for configuration, and the JSON spec doesn’t support comments. And even when the tool supports inline documentation of requirements with comments, that still doesn’t prevent the problem from happening if someone ignores or overlooks the comment. We need to ensure that this can be caught at build-time, within our CI system.
 
 
### But what about tests?
 
You might think, “The second scenario could have been prevented with regression testing. If they had added tests for the networking bug, Alice wouldn’t have been able to reintroduce the bug by accident”.
 
Adding tests is certainly a good idea, and it’s not mutually exclusive with the dep-based workflow approach. However, we can’t rely solely on tests to prevent these categories of problems. Some bugs manifest in ways that aren’t easily testable, because they require complex interactions of system state. Or, we may have identified that an upgraded dependency is causing problems before we understand the actual mechanism of the problem, so we don’t yet know which path needs to be tested.
 
But, most importantly, writing regression tests for the bug often requires whitebox testing, which can only be done by modifying the c/d package itself. At that point we find ourselves back in the first scenario: we’ve forked a package to add a feature (regression tests are a feature!) and we want it to be merged upstream, once the underlying bug is fixed.
Even if you’re able to provide complete test coverage for your packages and all of their dependencies, you still need a dependency management workflow that facilitates this process.
 
 
### Is this specific to Go?
 
Nope. This category of problems exists with software in general, and these are only two examples of how it can manifest. These particular examples are written to be familiar to Go programmers, but it’s not hard to construct similarly tough situations for Python, Java, Ruby, Javascript, C/C++, and so on. (Furthermore, this same problem can manifest at the OS level as well – instead of having package `a/b` import `c/d`, we could have application `b` linking against `libcd`).
 
 
So how does dep help us deal with these problems?
 
With dep, we can address the first scenario with a single line. In our Gopkg.toml, we can use a `source` entry to specify an alternate repository source for a given package. We can switch to our own fork of a project without having to rewrite any import paths.
 

Previously, we might have had:

```toml
[[constraint]]
name = "github.com/c/d
version = "1.0.0"
```

And instead, we can change this to:

```toml
[[constraint]]
name = "github.com/c/d
source = "https://github.com/myusername/d.git"
revision = "afadedface1"
```
 
 
When our patch is merged upstream, we just revert this commit, and we automatically switch back to using the upstream repository. Depending on the version or SHA range we originally specified in our requirements, we may not even have to update the version range this time; dep can pull in the latest compatible version.
 
 
The second scenario – accidentally reintroducing a known bug through an upgrade – is prevented as well, using this same trick. One convenient feature about Go: the `source` field is almost always redundant for typical use, because Go package names (and import paths) already specify source URLs. The mere presence of a `source` directive itself serves as a signal that something is afoot – a “proceed with extra caution” warning sign that will hopefully make Alice think twice before updating the version requested for that package.
 
But, even if Alice ignores or overlooks the `source` directive, we have another failsafe! It’s still impossible for her to accidentally update c/d to ` cafebead3`, because that commit only exists in the upstream repository, and we’ve specified our fork as the source repository.
 
We achieve a good balance here – it’s still possible to upgrade the dependency if needed (if the bug is fixed, or if it’s later deemed an acceptable risk). But there’s just enough friction in this workflow that Alice is unlikely to upgrade to a buggy version by accident.
 

