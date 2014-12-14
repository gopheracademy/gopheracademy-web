+++
author = ["Damian Gryski"]
date = "2014-12-13T08:00:00+00:00"
title = "Probabilistic Data Structures for Go: Bloom Filters"
series = ["Advent 2014"]
+++

In [part one](http://blog.gopheracademy.com/advent-2014/go-probably/), I talked
about some interesting probabilistic data structures..

In part two, I will discuss a more common approximate data structure: [Bloom
filters](https://en.wikipedia.org/wiki/Bloom_filter) and their variations.

## Bloom filters

A set is a collection of things.  You can add things to the set, and you can
query the set to see if an element has been added.  (We'll ignore deleting
elements from the set for now.) In Go, we might use a `map[string]struct{}` or
`map[string]bool` to represent a set.  If you query a map, you'll get back "No,
that element is not in the set." or "Yes, that element is in the set".  (A map
also supports iterating over the keys, something which bloom filters do not
support.)

A Bloom filter is an 'approximate set'.  It supports the same two operations
(insert and query), but unlike a `map` the responses you'll get from a Bloom
filter query are "No, that element is not in the set" or "Yes, that element is
*probably* in the set".  How often it gives a false positive can be tuned by
the amount of space you want to use, but a good rule of thumb is that by
storing 10 *bits* per element, a Bloom filter will give a wrong answer about 1%
of the time.  One thing you can't do with a Bloom filter is iterate over it to
get back a list of items that have been inserted.

Under the hood, a Bloom filter is a bit-vector.  For each element you want to
put into the set, you hash it several times, and based on the value of each
hash you set certain bits in the bit-vector.  To query, you do the same hashing
and check the appropriate bits.  If any of the bits that are supposed to be set
are still 0, you know that element was never put into the set.  If they're all
ones, you only know that the element *might* be in the set.  Those bits could
have been set by hash collisions from other keys in the set.

## Applications of Bloom Filters

Here's an example where this approximate answer can still be useful.  Google
Chrome will warn you if you are about to visit a site Google has determined is
malicious.  How might we build this functionality into a web browser?  Chrome
could certainly query Google's servers for every URL, but that would slow down
our browsing since we now have to perform two network requests instead of just
one.  And since most URLs *aren't* malicious, the web service would spent most
of its time saying "Safe" to all the requests.

We could eliminate the network requests if Google Chrome had a local copy of
all the dangerous URLs that it could query instead.  But now instead of just
downloading a browser, we'd need to include a several gigabyte data file.

Lets see what happens if we put the malicious URLs into a Bloom filter instead.
First, unlike a Go map, Bloom filters use less space that the actual data they
are storing -- we no longer have to worry the huge download any more.  Now we
just have to check the Bloom filter before visiting a URL.  But what about the
wrong answers?  A Bloom filter of malicious URLs will never report a malicious
URL as "safe", it might only report a "safe" URL as "malicious".  For those
cases, false positives, we can still make the expensive call to Google's
servers to see if it really *is* malicious or one of the 1% false positives.

Bloom filters are also used in Cassandra and HBase as a way to avoid accessing
the disk searching for non-existent keys.  For more applications, I've listed
two papers under Further Reading.

## Bloom Filter Libraries

A standard Bloom filter is fairly easy to implement, so [many people
have](http://go-search.org/search?q=bloom+filter).

## Bloom filter variants

Bloom filters are a good basic data structure that leave them ripe to
variations, generally at the cost of more space.

### Counting Bloom filter

Counting Bloom Filters are poorly named; the name 'counting' sounds like you
can query frequencies instead of just set membership.  In fact, the only
additional feature a counting Bloom filter provides is the ability to delete
entries.  Removing entries from a traditional Bloom filter is not possible
normally.  Resetting bits to 1 might remove additional keys if there were any
hash collisions.  Instead of setting a single bit, a counting Bloom filter
maintains a 4-bit counter, so that hash collisions increment the counters, and
a removal just decrements.  (The math shows that 4 bits is sufficient, with
high probability.)  [Patrick Mylud](http://github.com/pmylund) has an
implementation of a counting Bloom filter in
[pmylund/go-bloom](http://godoc.org/github.com/pmylund/go-bloom).

### Scaling Bloom Filters

Another problem with Bloom filters is that you must know the expected capacity
in advance.  If you guess too high, you'll waste space.  If you guess too low,
you'll increase the rate of false positives before you've inserted all your
items.

A scaling Bloom filter solve this by starting with a small Bloom filter and
creating additional ones as needed.  [Jian Zhen](https://github.com/zhenjl) has
implemented scaling Bloom filters in
[dataence/bloom/scalable](https://github.com/dataence/bloom/scalable).

### Opposite of a Bloom Filter

A Bloom filter provides no false negatives only false positives.  An
interesting curiosity is "what's a data structure that provides for only false
negatives no false positives." A list of keys that expires some entries has
this policy: any item that the list reports as in the set is actually there,
but an entry that is listed as "not present" may have simply been removed
instead.  [Jeff Hodges](https://twitter.com/jmhodges) has implemented this as
[jmhodges/opposite\_of\_a\_bloom\_filter](https://github.com/jmhodges/opposite_of_a_bloom_filter)

## Other solutions

If you want to store lots of Bloom filters, there's the
[bloomd](https://github.com/armon/bloomd) network daemon (with a [Go
client](https://github.com/geetarista/go-bloomd)).  The fine engineering team
at [bitly](http://bit.ly) has written
[dablooms](https://github.com/bitly/dablooms), a high-performance scalable
BBloom filter library in C that has Go bindings included.

## Further Reading

* [Bloom Filters on Wikipedia](https://en.wikipedia.org/wiki/Bloom_filter)
* [Interactive JavaScript Bloom filter demo](http://www.jasondavies.com/bloomfilter/)
* [Network Applications of Bloom Filters: A Survey](http://www.eecs.harvard.edu/~michaelm/NEWWORK/postscripts/BloomFilterSurvey.pdf)
* [Theory and Practice of Bloom Filters for Distributed Systems](http://www.dca.fee.unicamp.br/~chesteve/pubs/bloom-filter-ieee-survey-preprint.pdf)
