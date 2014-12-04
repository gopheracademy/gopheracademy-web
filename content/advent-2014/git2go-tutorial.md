++++
+author = ["David Calavera"]
+date = "2014-12-05T08:00:00+00:00"
+title = "Git2go inception"
+series = ["Advent 2014"]
++++

How many levels of inception we need to master git?

This tutorial that explains how to use [git2go](https://github.com/libgit2/git2go) to publish and article for the Go Advent 2014.

Git2go implements go bindings for [libgit2](https://github.com/libgit2/libgit2), a pure C implementation of the Git core methods. This is important because, unlike other libraries, libgit2 doesn't talk with any git binary.

You won't find the installation complicated if you've worked already with other C libraries. I recommend you to read the installation section in the [readme](https://github.com/libgit2/git2go/blob/master/README.md#installing). There is a detailed list of pre-requisites that you'll need to install.

The first step to publish our new article is to fetch the project from GitHub. We're going to clone the repository for that.

```go
import "github.com/libgit2/git2go"

repo, err := git.Clone("git://github.com/gopheracademy/gopheracademy-web.git", "web", &git.CloneOptions{})
if err != nil {
	panic(err)
}
```

With `web`, we're telling git2go to clone the project in the directory `web` from our current one.

Now that we have the project download in our machine, let's create a branch to put our article. We need to important pieces of information before that. First we need the commit that we'll use as a target for the branch. We also need to create a signature for the author, myself in this case.

```go
import (
	"time"
	"github.com/libgit2/git2go"
)

signature := &git.Signature{
	Name: "David Calavera",
	Email: "david.calavera@gmail.com",
	When: time.Now(),
}

head, err := repo.Head()
if err != nil {
	panic(err)
}

headCommit, err := repo.LookupCommit(head.Target())
if err != nil {
	panic(err)
}

branch, err = repo.CreateBranch("git2go-tutorial", head.Target(), false, signature, "Branch for git2go's tutorial")
if err != nil {
	panic(err)
}
```

Once we have the branch created, we need to add our markdown document to the index, sometime referred as the staging area. This is the same operation that you would do using `git add file`.

```go
import "github.com/libgit2/git2go"

idx, err := repo.Index()
if err != nil {
	panic(err)
}

err = idx.AddByPath("content/advent-2014/git2go-tutorial.md")
if err != nil {
	panic(err)
}

treeId, err := idx.WriteTree()
if err != nil {
	panic(err)
}
```

That will put our article in the staging area. The next step is to commit this changes!

We're going to use some information from the branch we just created and the tree where the index is pointing to to create this commit. We'll also reuse my previous signature as a committer and author.

```go
import "github.com/libgit2/git2go"

tree, err := repo.LookupTree(treeId)
if err != nil {
	panic(err)
}

commitTarget, err := repo.LookupCommit(branch.Target())
if err != nil {
	panic(err)
}

message := "Add Git2go tutorial"
err = repo.CreateCommit("refs/heads/git2go-tutorial", signature, signature, message, tree, commitTarget)
if err != nil {
	panic(err)
}
```

With this, we created a new commit. We also pointed the reference `refs/heads/git2go-tutorial` to it. And used the base commit of the branch as a parent.

Now, it's time to push our branch to my fork of the project. For this, we're going to need another remote repository. I have my fork in https://github.com/calavera/gopheracademy-web.

Git2go uses a callbacks system to feed some processes with intermediate information. I'm goint to use the ssh credentials in my machine to connect to my repository. In this case, I'll need to provide a callback that connects with the ssh agent in my machine. I'll also need another callback to verify that I'm connecting to GitHub:

```go
import "github.com/libgit2/git2go"

func credentialsCallback(url string, username string, allowedTypes git.CredType) (int, *git.Cred) {
	ret, cred := git.NewCredSshKeyFromAgent(username)
	return git.ErrorCode(ret), &cred
}

func certificateCheckCallback(cert *git.Certificate, valid bool, hostname string) int {
	if hostname != "github.com" {
		return git.ErrUser
	}
	return 0
}
```

Now that we have our callbacks, let's create that new remote and push my new branch.

```go
import "github.com/libgit2/git2go"

fork, err := repo.CreateRemote("calavera", "git@github.com:calavera/gopheracademy-web.git")

cbs := &git.RemoteCallbacks{
	CredentialsCallback: credentialsCallback,
	CertificateCheckCallback: certificateCheckCallback,
}

err = fork.SetCallbacks(cbs)
if err != nil {
	panic(err)
}

push, err := fork.NewPush()
if err != nil {
	panic(err)
}

err = push.AddRefspec("refs/heads/git2go-tutorial")
if err != nil {
	panic(err)
}

err = push.Finish()
if err != nil {
	panic(err)
}
```

And with that, we only need to create a new pull request and wait until this article is published!

PS: I left the complete, working code in this [repository](https://github.com/calavera/go-advent-2014). Play with it and use it to send the next article for the Go Advent Series!