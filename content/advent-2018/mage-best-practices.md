+++
author = ["Nate Finch"]
title = "Best Practices for Using Mage To Build Your Project"
linktitle = "Mage Best Practices"
date = 2018-11-02T06:40:42Z
+++

On my team at Mattel, we have a [magefile](https://magefile.org) for every Go
project (and we have several Go projects).  Our use of mage has grown with the
team and the projects, and it has been a big help keeping our dev practices
uniform and shareable.  Here’s how we do it.

### Retool

One of the great things about Go is the tooling, not just the official tooling
but all the community tooling.  The only downside is keeping track of everything
you need to build your complicated projects.  That’s where
[retool](https://github.com/twitchtv/retool) comes in.  It's like `dep` but for
go tools, pinning revisions so that all your developers run the exact same
version.  

Mage makes it easy to use retool – a `Tools` target ensures you have retool
installed (`go get` won't hit the internet if the code already resides locally,
so it's ok to run it every time), and then calls `retool sync` to make sure
you're up to date.  Any other target that uses a build tool should define the
Tools target as a dependency - `mg.Deps(Tools)`.  Thanks to Mage, no matter how
many places that gets called in your build infrastructure, it'll only do the
tools check once.

```
func Tools() error {
	mg.Deps(checkProtoc)

	update, err := envBool("UPDATE")
	if err != nil {
		return err
	}
	retool := "github.com/twitchtv/retool"

	args := []string{"get", retool}
	if update {
		args = []string{"get", "-u", retool}
	}

	if err := sh.Run("go", args...); err != nil {
		return err
	}

	return sh.Run("retool", "sync")
}
```

Now that you have your tools synced, you need to make sure you run the version
retool caches, instead of the one that is in your PATH.  This is easy with a
little wrapper function:

```
// retool runs a command using a retool-cached binary.
func retool(cmd string, args ...string) error {
	return sh.Run("retool", append([]string{"do", cmd}, args...)...)
}
```

Here, we use mage's `sh` package to output nice error messages when things fail
(instead of just "exiting with code 1"), support verbose or quiet output, and
automatic expansion of $FOO style environment variables.  Prepending `retool do`
on the front of the command, so makes it use retool to run the tool from
retool's cache. 

### Deps

Have you ever forgotten to update your dependencies before rebuilding?  I think
we all have. With mage, `mg.Deps(Dep)` at the beginning of any build target
ensures that we've run `dep ensure` and our code isn't out of date.  (Yeah, we
should move to modules at some point.)

### Releases

We generate releases uses the wonderful [goreleaser](https://goreleaser.com/),
which automates creating releases for a multitude of release platforms, with a
single simple configuration file.  Now, go releaser is easy to run, but I'll be
damned if I can remember how to create a tag and push it to a remote in git.
Now I can do both in one simple command like `TAG=v1.2.3 mage release` and we
even clean up the tag if something goes wrong.  

```
func Release() (err error) {
	if os.Getenv("TAG") == "" {
		return errors.New("TAG environment variable is required")
	}
	if err := sh.RunV("git", "tag", "-a", "$TAG"); err != nil {
		return err
	}
	if err := sh.RunV("git", "push", "origin", "$TAG"); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			sh.RunV("git", "tag", "--delete", "$TAG")
			sh.RunV("git", "push", "--delete", "origin", "$TAG")
		}
	}()
	return retool("goreleaser")
}
```

Along the same lines, sometimes you just want to produce a binary and not do the
whole release process.  But who can remember how to format ldflags to embed
build info into your binary?  Mage remembers -

```
func Build() error {
    mg.Deps(Dep)
    return sh.RunV("go", "build", "-o", "project", "-ldflags="+ldflags(), "github.com/Mattel/project")
}

func ldflags() string {
	timestamp := time.Now().Format(time.RFC3339)
	hash := hash()
	tag := tag()
	if tag == "" {
		tag = "dev"
	}
	return fmt.Sprintf(`-X "github.com/Mattel/project/proj.timestamp=%s" ` +
    `-X "github.com/Mattel/project/proj.commitHash=%s" ` +
    `-X "github.com/Mattel/project/proj.gitTag=%s"`, timestamp, hash, tag)
}

// tag returns the git tag for the current branch or "" if none.
func tag() string {
	s, _ := sh.Output("git", "describe", "--tags")
	return s
}

// hash returns the git hash for the current repo or "" if none.
func hash() string {
	hash, _ := sh.Output("git", "rev-parse", "--short", "HEAD")
	return hash
}
```

### Reusing Mage Code

Of course, all this code is mostly the same across projects, so sharing it
between projects should be a priority.  That's easy with the new mage:import
directive.  This will bring in all the mage targets from the given package as
targets for anyone that runs this magefile.  If you import it without the _ you
can use common helper functions from the package as well.  An easy way to
customize that code across projects is to use simple init function in your local
magefile to set the current repo name, and then use that in your imported code:

```
import (
    //mage:import
    _ "github.com/Mattel/mage"
)

func init() {
    os.Setenv("MATTEL_REPO", "projname")
}
```

Then your common build target might look like this:

```
func Build() error {
    mg.Deps(Dep)
    return sh.RunV("go", "build", "-o", "$MATTEL_REPO", "-ldflags="+ldflags(), "github.com/Mattel/$MATTEL_REPO")
}
```

At Mattel, using mage has solved a lot of common dev problems, like mismatched
binaries, forgetting processes, and just time spent wrangling the command line.
Plus, standardizing across projects makes it easier to jump into a new project
and figure out how everything fits together.

I'd love to hear how others are using mage (you'd be surprised at the variety of
ways people use it).  Come talk on #mage on [gopher
slack](https://gophers.slack.com/messages/general/), or post on the new magefile
[google group](https://groups.google.com/forum/#!forum/magefile).


