+++
author = ["Damian Gryski"]
date = "2014-12-01T08:00:00+00:00"
title = "Probabilistic Data Structures for Go"
series = ["Advent 2014"]
+++

Imagine you had access logs for a very high traffic website.  How would you
determine how many different IP addresses accessed your site?  Or how many hits
from a particular IP?  Or which ones accessed it the most?  Assuming IPv4
addresses, you could use a `map[uint32]int` to maintain the counts, but that
could end up using a lot of memory.  It's certainly possible to have a map with
4 billion entries, and a real log server wouldn't have accesses from every
single valid IP address, but the problem still exists.

Luckily, there's a class of algorithms that lets you trade memory usage for
accuracy.  In many cases the reduction in memory can be significant and the
drop in accuracy is minimal.  The
[go-probably](https://github.com/dustin/go-probably) library by Dustin Sallings
implements a number of these basic data structures and algorithms.  (It's
almost always better to process things exactly, if you have the memory though.
These algorithms can be slower than exact answers for small data sets.)

I also like this package because it was one of the first I contributed to on
GitHub, and my first commit-bit on somebody else's repository.

Let's look at how the types in this package would handle the above problems.

## HyperLogLog: cardinality estimation

The algorithm we're going to use for cardinality estimation (i.e., counting
distinct items in our set) is
[HyperLogLog](https://en.wikipedia.org/wiki/HyperLogLog).   I'm not going to
explain the math (there are [already good  blog
posts](http://research.neustar.biz/2012/10/25/sketch-of-the-day-hyperloglog-cornerstone-of-a-big-data-infrastructure/)
for that), only how to use the implementation in `go-probably`.

An abridged look at at the API shows:

```go
func NewHyperLogLog(stdErr float64) *HyperLogLog
func (h *HyperLogLog) Add(hash uint32)
func (h *HyperLogLog) Count() uint64
```

First we construct an instance of a HyperLogLog estimator with
`NewHyperLogLog()`.  Then for each item, we need to pass a 32-bit hash of the
value to `Add()`, and at the end we call `Count()` to get the estimate.

For this example I'm using crc32 as our hash function.  It works "good enough"
and has the advantage of being in the standard library.  In production I might
use [murmur3](https://github.com/spaolacci/murmur3) or
[xxhash](https://github.com/vova616/xxhash), both of which are faster but are
external dependencies.)

```go
package main

import (
	"bufio"
	"fmt"
	"hash/crc32"
	"os"

	"github.com/dustin/go-probably"
)

func main() {
	hll := probably.NewHyperLogLog(0.0001)
	for scanner := bufio.NewScanner(os.Stdin); scanner.Scan(); {
		hll.Add(crc32.ChecksumIEEE(scanner.Bytes()))
	}
	fmt.Println("estimated distinct items: ", hll.Count())
}
```

### Count-Min Sketch: approximate frequencies

The next data structure provided by `go-probably` is [Count-Min
Sketch](https://en.wikipedia.org/wiki/Count%E2%80%93min_sketch).  A CM-sketch
lets you estimate how many times you've seen different elements in your
data set.  A count-min sketch is similar to a [Bloom
filter](https://en.wikipedia.org/wiki/Bloom_filter) in that they're both
probabilistic, but a Bloom filter returns a boolean "Have I seen this", rather
than an estimated count.

A count-min sketch is useful for on-line queries, since if you knew which
addresses you wanted to track you would do so as you saw them in the input.

The API exposed by the Sketch type is a bit more complex, but for most uses you
can focus on the three methods which are similar to those for HyperLogLog:

```go
func NewSketch(w, d int) *Sketch
func (s *Sketch) Increment(h string) (val uint32)
func (s Sketch) Count(h string) uint32
```

Unlike with HyperLogLog, we don't need to provide a hash function; `Add()` uses
its own hash function internally.  This demo program reads an input file and
then prompts the user for entries to provide estimated counts for.

```go
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/dustin/go-probably"
)

func main() {
	input := flag.String("f", "", "input file")
	flag.Parse()

	sk := probably.NewSketch(1<<20, 5)

	f, err := os.Open(*input)
        if err != nil {
            log.Fatal(err)
        }
	for scanner := bufio.NewScanner(f); scanner.Scan(); {
		sk.Increment(scanner.Text())
	}

	fmt.Print("query> ")
	for scanner := bufio.NewScanner(os.Stdin); scanner.Scan(); fmt.Print("query> ") {
		query := scanner.Text()
		fmt.Printf("esimated count for %q: %d\n", query, sk.Count(query))
	}
}
```

## TopK queries

Because a Count-Min sketch contains estimated counts, it's fairly
straightforward to use it along with a heap to estimate the most popular
elements in a stream.  However, I prefer a much more magical algorithm that
includes a tiny sketch as one of its sub-pieces.  I've implemented it in
[dgryski/go-topk](https://github.com/dgryski/go-topk)

## Putting it together: Hokusai

A good example of using Count-Min sketches in a real system is the Hokusai
paper, by Sergiy Matusevych, Alex Smola, Amr Ahmed.  It tracks all its keys and
creates aggregate sketches over time, allowing point queries when the full set
of keys would be prohibitively large to store.  The example given in the paper
is tracking the popularity of all the different search queries provided to a
search engine in a given week, and being able to plot how often a given search
query occurred.

I have a basic implementation this system in
[dgryski/hokusai](https://github.com/dgryski/hokusai).

A number of my recent patches to `go-probably` came from my needs while
implementing this paper.

## Further Reading

The go-probably source code has links to all the papers describing the
algorithms, most of which are quite readable.

The Highly Scalable blog had a good post on [Probabilistic Data Structures for
Web Analytics and Data
Mining](https://highlyscalable.wordpress.com/2012/05/01/probabilistic-structures-web-analytics-data-mining/)
that gives a good overview how some of these algorithms actually work.

Google has a modification to HyperLogLog called
[HyperLogLog++](http://research.google.com/pubs/pub40671.html). There's a good
analysis of the changes they've made at [HyperLogLog++: Google’s Take On
Engineering
HLL](http://research.neustar.biz/2013/01/24/hyperloglog-googles-take-on-engineering-hll/).
There is a Go implementation at
[clarkduvall/hyperloglog](https://github.com/clarkduvall/hyperloglog).
