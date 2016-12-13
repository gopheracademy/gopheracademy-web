+++
author = ["Scott Mansfield"]
date = "2016-12-17T05:26:58-05:00"
title = "Goroutree: A tree-based set made of coordinating goroutines"
series = ["Advent 2016"]
+++

This was one of thoe projects that sat in the back of my mind for quite a while. It was destined to
join the many others in my side project graveyard unless I had a good reason to finish it, like a
date for a blog post.

This post is an axplanation and exploration of the goroutree data structure. The structure itself is
composed of many separate but coordinating goroutines that make up the tree itself. It has only a
few operations and can't self-balance, but it's a decent proof of concept.

I show a bit of code, but most of my descriptions are in prose because code would make the post get
quite long. if you want to follow along in the code, then open the link at the bottom to the GitHub
repository that hold the package.

# Example

Let's take a look at a usage example right off the bat:

```go
import "github.com/ScottMansfield/goroutree"

func main() {
    g := goroutree.New()

    boolres := make(chan bool)
    g.Insert(boolres, 5)
    <-boolres // will return true

    g.Contains(boolres, 5)
    <-boolres // will return true

    g.Contains(boolres, 4)
    <-boolres // will return false

    g.Delete(boolres, 4)
    <-boolres // will return false

    g.Delete(boolres, 5)
    <-boolres // will return true
}
```

# Goroutree

The `Goroutree` type is very simple:

```go
type Goroutree struct {
    cmdchan chan cmd
}
```

It only has a command channel to send commands to and nothing else. The `New()`
function only needs to start up the manager goroutine and return a pointer to
the `Goroutree`.

```go
func New() *Goroutree {
    cmdchan := make(chan cmd)
    go manager(cmdchan)

    return &Goroutree{cmdchan}
}
```

# Manager Goroutine

There's actually a hidden "super root" node that takes the initial requests and then forwards them
as needed to the root node. This is to support things like having an empty tree or deleting the root
node. For example, when checking if the tree contains a number, if the root node is null the answer
is always false. For deleting the root node, someone has to be notified of the new root's handle
(read: input channel). THe manager goroutine is there to play that role.

Introducing the manager also made the synchronous interface be completely separated from the tree
instead of sometimes requiring a new goroutine to be spawned just to respond to the request.

# Node Goroutine

Each node is a separate goroutine that owns a particular value. It has an input channel that is a
stream of commands which the node will react to one at a time.

Nodes have knowledge of their own value (an int), and several channels:

* input
* parent
* left child
* right child

The input and parent channels are guaranteed to be not nil, but the left and right are not. Each
node is independently and concurrently receiving and serving requests coming in. There's some
situations where a child will message a parent, but mostly the commands go downward.

# Operations

There's three main operations on the set: `Insert`, `Contains`, and `Delete`. There is a fourth,
`Print`, that is mainly used for debugging.

The operations all follow the basic binary search tree patterns. There's no facny rotations that
happen in the middle, but there are a couple somewhat complicated interactions during deletion of a
node. There's some extra commands that are only used internally that I will show as well.

## Sidenote: types

I created a command type and many implementations of that command in order to send the proper data
around. These are the constant type values and interface for the command type:

```go
type cmdType int

const (
    ctInvalid cmdType = iota
    ctInsert
    ctContains
    ctDelete
    ctPrint
    ctNewChild
    ctExtractMin
)

type cmd interface {
    typ() cmdType
}
```

This is probably a pattern that will make some cringe. I use a `typ()` function to determine the
type of the command instead of doing a type switch. I do it because I like it. There's no real
reason to do it that way over using a type switch.

## Insert

The `Insert` function will add a new value to the set if it does not already exist. In effect, it
will create a new goroutine to own that value and attach it at the proper point in the tree. The
channel passed in will receive a boolean; true for successful insert and false if the value already
existed.

Internally, an insert command message is sent through the tree. The manager will create the root if
needed. From there, the nodes will pass down to the left for values less than them and to the right
for greater until the child channel is nil. It then spawns a child goroutine to own the value and
return true to the caller. If the value is equal ot the node's value at any point, it will return a
false to the caller.

## Contains

The `Contains` function tells whether a value exists in the set. This is safe to use concurrently.
The caller's channel with either get a true if the value exists or a false if it doesn't. It does
not modify the tree at all. THe internals are very simple, with a contains command getting sent down
the tree until a node can respond difinitively.

## Delete

`Delete` is by far the most complicated operation. It has a couple simple cases. If the value isn't
equal it will pick the proper side. If that side is nil, a false will be sent back to the caller.

If the value is equal, things get interesting. The node has to figure out how to extract itself from
the tree. In a normal binary tree, this is relatively simple because you can hold on to references
to everything required in one place. In this kind of tree, we need to be able to pull this off with
just messages.

There's three sub-cases here:

1. No children (easy)
1. One child (medium)
1. Two children (hard)

### No children

In this case, the node needs to be able to inform its parent that it will be going away. To do this
there is a special kind of message to replace a child channel. This is where the handle to the
parent node comes in handy. In order to disconnect from the parent node, the child will send a
command to replace itself with nil. In order to tell which child is to be replaced at the parent,
each node on the way down will modify the command with direction it is sending downward. The only
thing left at that point is to return.

### One child.

This is very similar to the case above, but instead of sending nil as a replacement it send the only
child it has to the parent and then returns.

### Two children

Ah, this one is fun. In order to remove itself, the node in every case has to find a suitable
replacement. When there are two children, there are two choices: the left subtree's maximum or the
right subtree's minimum. To simplify things, I elected to always find the minimum value of the right
subtree.

To do this, there's a special internal-only command to find the minimum value. This command is first
passed right by the node being deleted and then left as far as possible. At the end of the leftward
movement, it has one or zero children and can be deleted according to the first two cases above.

Now in order to delete the initial requested value out of the tree, I cheat a little bit here. It
was probably possible to surgically move the minimum right subtree value in to place, but it's far
easier to just return the found value and delete that minimum value node. Then we take the value and
replace the value in the running node that was to be deleted. We move only the value instead of the
goroutine connections but gain the same effect.

While the goroutine is performing this operation it is blocking on a response from the minimum value
search, so no other commands will end up coming through. This should guarantee that we don't end up
in some weird inconsistent state, but like I said before I haven't done any testing for high
concurrency situations.

## Print

The print command prints the tree in an in-order representation left to right, but puts newlines
between each noe and prints the number of levels down in spaces before each number. This output
allows visual and programmatic verification of structure of the tree. 

This command is actually implemented as a synchronous printing of the entire tree. The node will
send the print command down the left subtree and wait for it to return before printing and then
sending the message down the right subtree. This means that a print is a blocking operation for the
entire tree, whereas many other operations could be done in parallel, e.g. two Contains commands may
be able to go down different branches concurrently.

# Tests

There are tests in the repository to test out all of the behavior in a single-threaded fashion. They
are pretty heavily nested, which is my preferred way of making tests like this. There's one main
test function per external operation (except `Print`) that should exercise all of the different code
paths of each function. They also show how to use it in a few different ways.

# Performance

The goroutree definitely is not a contender for the fastest tree-based set that you could find. With
all of the overhead of passing messages around, it has a lot of context switching to slow it down.
It's actually pretty difficult to benchmark something like this. Checking the speed of almost any
operation is going to depend heavily on the depth of the tree at the time the operation is done. To
at least attempt some benchmarking, I wrote some benchmarks that will insert the same value into the
same tree over and over again. There will be a slight difference between the run times of the first
case where the value is inserted and of the subsequent cases where the value already exists. Since
there's no way to shut down a tree I had to cheat a bit. I also only created one edge of the tree to
simulate a much larger tree.

The output below is the benchstat output for 30 runs of the `Insert` benchmarks. Note that at 40
levels deep a full tree would be holding over a trillion items. I augmented the output with the
number of items a full tree would have with the same number of levels.

```
$ benchstat testoutput
name                     time/op        full tree size
Insert/Levels/0Deep-8     879ns ± 3%    0
Insert/Levels/1Deep-8    1.25µs ± 4%    1
Insert/Levels/2Deep-8    1.60µs ± 4%    3
Insert/Levels/3Deep-8    1.94µs ± 3%    7
Insert/Levels/4Deep-8    2.33µs ± 3%    15
Insert/Levels/5Deep-8    2.67µs ± 4%    31
Insert/Levels/6Deep-8    3.04µs ± 2%    63
Insert/Levels/7Deep-8    3.53µs ± 2%    127
Insert/Levels/8Deep-8    3.80µs ± 3%    255
Insert/Levels/9Deep-8    4.13µs ± 2%    511
Insert/Levels/10Deep-8   4.51µs ± 3%    1023
Insert/Levels/20Deep-8   8.23µs ± 2%    1048575
Insert/Levels/30Deep-8   12.0µs ± 2%    1073741823
Insert/Levels/40Deep-8   16.0µs ± 4%    1099511627775
Insert/Levels/50Deep-8   20.2µs ± 3%    2^50 - 1
Insert/Levels/60Deep-8   25.1µs ± 2%    2^60 - 1
Insert/Levels/70Deep-8   30.1µs ± 2%    2^70 - 1
Insert/Levels/80Deep-8   34.6µs ± 3%    2^80 - 1
Insert/Levels/90Deep-8   39.4µs ± 1%    2^90 - 1
Insert/Levels/100Deep-8  44.0µs ± 2%    2^100 - 1
```

# Future Work

If I do continue hacking on this (or if anyone else wants to fork) there's a couple things I would
definitely consider:

1. Add the ability to shut down the tree. This shouldn't be too hard, just send a kill message down
the tree recursively.
1. Better testing under higher concurrency scenarios
   - Current testing is single-threaded, there may be some deadlocks waiting to happen
1. Rotations to keep the tree balanced
   - This would require augmented nodes and for the depth information to be kept in sync
   - THe messages to do this are already in place
1. The implementation could instead take an interface that implements a `Compare()` function that
returns -1, 0, or 1 instead of just using ints.


# Links

* Code on GitHub: https://github.com/ScottMansfield/goroutree
* Godoc: https://godoc.org/github.com/ScottMansfield/goroutree