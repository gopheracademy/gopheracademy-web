+++
author = ["Damian Gryski"]
date = "2014-12-05T08:00:00+00:00"
title = "String Matching"
series = ["Advent 2014"]
+++

How do you search for a string?  If it's just once, `strings.Index(text,
pattern)` is probably your best option.  The standard library currently uses
[Rabin-Karp](https://en.wikipedia.org/wiki/Rabin%E2%80%93Karp_algorithm) to
search the text for the pattern.  However, there are lots of different cases
for string searching, each of which has its own set of "best" algorithms.
Fortunately for us, many of them already have implementations in Go.

## index/suffixarray

What if you had a single text you wanted to do lots of searches through for
different patterns?  The
[index/suffixarray](http://golang.org/pkg/index/suffixarray/) package
implements a [suffix array](https://en.wikipedia.org/wiki/Suffix_array), which
allows substring searches in O(log n) time on the length of the text.  Creating
the suffix array takes O(n log n) time, but you can quickly recoup that cost if
you're doing lots of searches over the same text.  And unlike `strings.Index`,
the suffix array can return all the locations that match the pattern, not just
the first one.

```go
    idx := suffixarray.New([]byte(`foobarbazbuzquxbazzot`))
    fmt.Println(idx.Lookup([]byte(`baz`), -1))
```

The suffix array package is a hold-over from when godoc was in the standard
library; it powers the search box.  You can also use the suffix array to search
across a large number of documents by joining all the texts into one
*big* string and mapping the offsets returned by `Lookup()` back to the
original documents the strings appeared in.

This example is a bit more involved:

* create the `data` array and list of offsets for each document
* create the suffix array from the combined data
* query the suffix array for all indexes where `ello` appears
* search the offsets list for the document covering that offset
* print out all the matched documents

```go
package main

import (
	"fmt"
	"index/suffixarray"
	"sort"
)

func main() {
	docs := []string{
		"hello world",
		"worldly goods",
		"yello",
		"lowly",
	}

	var data []byte
	var offsets []int

	for _, d := range docs {
		data = append(data, []byte(d)...)
		offsets = append(offsets, len(data))
	}
	sfx := suffixarray.New(data)

	query := "ello"

	idxs := sfx.Lookup([]byte(query), -1)
	var results []int
	for _, idx := range idxs {
		i := sort.Search(len(offsets), func(i int) bool { return offsets[i] > idx })
		if idx+len(query) <= offsets[i] {
			results = append(results, i)
		}
	}

	fmt.Printf("%q is in documents %v\n", query, results)
}
```

## Aho-Corasick: matching a large number of patterns

A suffix array lets us search for a pattern in a (preprocessed) document.  Lets
flip this around and say we have a large number of patterns and want to know
which ones appear in a given text. The [Aho-Corasick string matching
algorithm](https://en.wikipedia.org/wiki/Aho%E2%80%93Corasick_string_matching_algorithm)
takes a set of patterns and creates a giant finite-state-machine which, when
the document is passed to it, matches against all the patterns at once.
CloudFlare has released an implementation in
[cloudflare/ahocorasick](https://github.com/cloudflare/ahocorasick)

Lets use it to match a document against a list of planets.  Note that it even
finds the overlapping match for `venusaturn`.

```go
package main

import (
	"fmt"

	"github.com/cloudflare/ahocorasick"
)

func main() {
	patterns := []string{
		"mercury", "venus", "earth", "mars",
		"jupiter", "saturn", "uranus", "pluto",
	}

	m := ahocorasick.NewStringMatcher(patterns)

	found := m.Match([]byte(`XXearthXXvenusaturnXX`))
	fmt.Println("found patterns", found)
}
```

## Trigram indexing

At the other end of the string matching scale, we have actual search engines
like [bleve](https://github.com/blevesearch/bleve) and
[ferret](https://github.com/argusdusty/Ferret).  Both of these might be
overkill for a small application that just needs a bit of string matching sped
up.  This was the situation I found myself in when working on
[carbonserver](https://github.com/grobian/carbonserver).  Without getting into
much detail, the problem involved matching file-system globs against a large
number files (~700 thousand) in multiple nested directories.  A profile showed
that almost all the time was spent evaluating the globs, reading directories,
and sorting file names.  I wrote quick trigram indexing library
[dgryski/go-trigram](https://github.com/dgryski/go-trigram) and was able to
reduce query times from 20ms to less than 1ms.

There are blog posts describing [trigram
indexing](http://swtch.com/~rsc/regexp/regexp4.html) in detail, and I'll give a
quick example of using my trigram library.  This time our documents will
consist of Go conferences:

```go
package main

import (
        "fmt"

        "github.com/dgryski/go-trigram"
)

func main() {
        docs := []string{
                "dotGo",
                "FOSDEM",
                "GoCon",
                "GopherCon",
                "GopherCon India",
                "GothamGo",
                "Google I/O",
        }

        idx := trigram.NewIndex(docs)

        found := idx.Query("Gopher")
        fmt.Println("matched documents", found)
}
```

Note that with trigram matching, it's important to make sure the resulting
documents actually contain the query string.  It's possible to have trigram
matches for a query, even if the document doesn't actually contain them.  For
example, the document `GopherGoggles`, would match the query `rGop` (trigrams:
`rGo`, `Gop`).

I've covered three different algorithms for string matching, but there are
[many more](http://www-igm.univ-mlv.fr/~lecroq/string/).  The area of [string
searching](https://en.wikipedia.org/wiki/String_searching_algorithm) is still
an active research topic, with applications in a number of different fields.
