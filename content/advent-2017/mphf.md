+++
author = ["Damian Gryski"]
date = "2017-12-02T00:00:00-05:00"
linktitle = "Minimal Perfect Hash Functions"
series = ["Advent 2017"]
title = "Minimal Perfect Hash Functions"
draft = true
+++

A regular hash function turns a key (a string or a number) into an integer.
Most people will know them as either the cryptographic hash functions (MD5,
SHA1, SHA256, etc) or their smaller non-cryptographic counterparts frequently
encountered in hash tables (the `map` keyword in Go). Collisions, where two
input values hash to the same integer, can be an annoyance in hash tables and
disastrous in cryptography. Collisions can happen with any standard hash function and any number of keys.
In particular, as long as the set of strings to be hashed is
larger than the output size of the hash, there will always be at least one
collision. For example, imagine a hash function that produces a single byte
of output. There are 256 possible output values. If I try to hash 257
strings, at least one of them must collide -- there just aren't enough
different outputs available. This is known as the [pigeonhole principle](
https://en.wikipedia.org/wiki/Pigeonhole_principle).

However, if we know the set of keys in advance, we can be more careful. A
*perfect* hash function can be constructed that maps each of the keys to a
distinct integer, with no collisions. These functions only work with the
specific set of keys for which they were constructed. Passing an unknown key will
result a false match or even crash.

A minimal perfect hash function goes one step further. It maps the N keys to
exactly the integers 0..N-1, with each key getting precisely one value.

We can rank hash functions on a few different criteria: speed to construct,
speed to evaluate, and space used. Imagine a hash function that stores every
key in an array, and just walks down the array looking for a match, then
returns that integer. Obviously this maps each element to a distinct value,
and it's also quick to construct. However, both the space required and the execution
time are not optimal. We can do better.

Let’s start with a very basic implementation. Here’s an example set of keys.
Let’s pretend they’re commands for some simple network protocol, like NATS or
Redis. We’ll read 4 bytes from the network and we want to check if we have a
valid command before dispatching to the appropriate processing loop.

Here are the commands:
```
INFO
CONN
PUB 
SUB 
UNSU
PING
PONG
+OK 
-ERR
AUTH
PUSH
ADD 
DECR
SET 
GET 
QUIT
```

I’ve left the spaces after the three letter commands so that each key fits
into a uint32. In fact, we won't deal with these as strings but we'll turn
them directly into uint32s.

Here's our first hash function. It takes the uint32 and returns the bottom 4 bits.

```
func hash1(x uint32) uint32 {
    return x & 15
}
```

And then check for duplicates:

```
dups := make(map[uint32]bool)
for _, k := range keys {
    h := hash1(k)
    if dups[h] {
        log.Println("duplicate found")
    }
    dups[h] = true
}
```

This is very fast, but when we test, half of the keys collide. Changing the
function to look at the upper 4 bits doesn't work either. The duplicate
initial letters (PUSH, PUB) and trailing letters (PONG, PING) means we need
to do more shuffling.

Here’s a second attempt:

```
const multiplier = 31
func hash2(x uint32) int32 {
   var u32 = (x * multiplier) >> 28
   return u32 & 15
}
```

Here we’ve made two changes. First, we’re multiplying by 31, a nice random
number that shows up in hash functions. Second, we’re going to extract the
high bits of the result. High bits of multiplications tend to have a bit more
entropy than the low bits, another common hash function trick.

This is an improvement, although there are still 6 collisions, down from 8.
But now we have a framework we can use. Can we find a value for `multiplier`
that eliminates all collisions? This is easy enough to brute force.

```
2017/11/29 22:18:25 len(keys)= 16
2017/11/29 22:18:25 searching for multiplier to fit 16 keys in 16 slots
m = 715138
INFO 3
CONN 6
PUB  15
SUB  7
UNSU 13
PING 8
PONG 0
+OK  1
-ERR 4
AUTH 2
PUSH 11
ADD  14
DECR 5
SET  10
GET  9
QUIT 12
```

And indeed, when we set the hash function to use `715138` we
see that it does indeed map the 16 keys perfectly into 0 to 15.

So in order to check if the bytes we've read are valid, we hash them with our
perfect hash function and look at the appropriate index in an array. If it's
the key we're looking for, then we know it's valid. This will be fast because
the arrays are small and we're just comparing two uint32s.

```
func u32s(s string) uint32 {
        return binary.LittleEndian.Uint32([]byte(s))
}

// in the order listed above
var values = []uint32{
	u32s("PONG"),
	u32s("+OK "),
	u32s("AUTH"),
	u32s("INFO"),
	u32s("-ERR"),
	u32s("DECR"),
	u32s("CONN"),
	u32s("SUB "),
	u32s("PING"),
	u32s("GET "),
	u32s("SET "),
	u32s("PUSH"),
	u32s("QUIT"),
	u32s("UNSU"),
	u32s("ADD "),
	u32s("PUB "),
}
```

Lets benchmark this against a regular Go map.

```
var sink bool

func BenchmarkHash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, k := range keys {
			sink = values[hash2(k)] == k
		}
	}
}

func BenchmarkMap(b *testing.B) {
	m := make(map[uint32]bool)
	for _, k := range keys {
		m[k] = true
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, k := range keys {
			sink = m[k]
		}
	}
}
```

And the output:

```
BenchmarkHash-8   	100000000	        20.4 ns/op
BenchmarkMap-8    	10000000	       172 ns/op
```

A very significant improvement.

But how can we generalize this? And does this always work? And is it always
fast?

Well, the first thing we notice is that as the set becomes larger, it becomes
much more difficult to find a value for `multiplier` that works, and one might
not even exist. Even with 32 elements, trying to find the right
value for `multiplier` might be cost prohibitive or even impossible. We
certainly wouldn’t want to do it at runtime. These questions are why Minimal
Perfect Hash Functions are an interesting research topic.

*Hash-Displace*

The issue we ran into with our second attempt was that it was too difficult to
find a single value for the multiplier that worked for larger sets. Two keys
that collide with one hash function are unlikely to collide with a second hash
function as well. We can take advantage of this fact by switching to a
construction that uses more than one hash function. The first hash function
will distribute the keys over the buckets, with "a few" collisions.  Then, for
each set of collisions, we try to find a second hash function that distributes
the keys evenly with no collisions. The paper fully describing this approach is
[Hash, Displace, and Compress](http://cmph.sourceforge.net/papers/esa09.pdf)
and, an earlier version is [Practical Minimal Perfect Hashing Functions for
Large Databases](http://eprints.cs.vt.edu/archive/00000223/01/TR-90-41.pdf).
My simplified version of this algorithm is here:
https://github.com/dgryski/go-mph .

My implementation is about 25% faster than using a regular Go map for 16 keys
and 50% faster when testing with the 235,000 entries in */usr/dict/words*.
Unlike the previous algorithm, this one has no issues with large key sets.
Constructing the hash function for this wordlist takes only 100ms-125ms.

There are three ways to judge a hash function: construction time, evaluation
time, and space usage. Our first successful hash function had virtually no
space usage, a fast evaluation time, but a huge construction time.

This algorithm has a pretty small construction cost. It turns out to be
linear in the number of keys. The evaluation time is also constant time: one
"standard" hash function evaluation, some integer mixing, and two table
lookups.

As for space usage, in my implementation this algorithm uses 8 bytes per
entry: each key gets its own 4-byte index (0..N-1) and another 4-byte seed
for the second hash function. The "Hash, Displace, and Compress" paper gives a method that allows the
intermediate arrays to be compressed to reduce the space needed, but still
used for querying without decompression. As is, they can easily be written
out to disk and loaded back later, or even by a different process. The slices
could even be accessed via `mmap`.

For my version, I could actually reduce the space usage a little bit at the
cost of a performance hit. In order to make the lookups faster, the arrays
are sized to be the next larger power of two. When we hash, we use a bitmask
to get the appropriate slot in the table. If we replace the bitmask with a
(much slower) modulo operator, then we could properly size the arrays with
exactly N entries.

*Massive Key Sets*

Using 8 bytes per entry might not seem like much, but what if you have a
billion keys? Or 10 billion?

Last February I saw a paper [Fast and scalable minimal perfect hashing for
massive key sets](https://arxiv.org/abs/1603.04330). This paper aims not only
to be a fast construction of a minimal perfect hash function, but also to
drastically reduce the space needed to store the mappings.

Similar to the two-level hashing used for hash/displace, this algorithm uses
multiple hash functions to deal with collisions. However, instead of the
targets being hash table entries, the targets are *bits* in a bit vector. If
only a single key hashes to a particular bit, then the bit is set to 1. If more
that one key hashes to that bit, then the bit is left as 0 and the keys that
collide are moved to the next layer down. Eventually, all the keys will have been
placed at some level. To look up a value, we must find out which bit it maps
to. We hash the key with the first hash function and look up that bit in the
first-level bitvector. If it's a 1, we stop. If it's a 0, we move to searching
in the second-level bitvector with the second hash function, and so on. In
order to figure out the value 0..N-1 to return for the hash function, the
algorithm uses a trick common in [succinct data structures]
(https://en.wikipedia.org/wiki/Succinct_data_structure). We know there must be
exactly one set bit per key in the bit vector. So once we've found the bit for
a key, we set the return value to be the number of 1s *earlier* in the all levels of
bit vectors. This can be made efficient by storing extra indexing information about the number
of 1s at each level and bit vector subsection.  Since there are exactly N bits set,
the hash function will return 0..N-1 as we wanted.

Another advantage of the data struture the paper describes: it can be
constructed in parallel by different threads using atomics to access the
bit vectors.

In terms of speed, it is only a tiny bit faster than a regular Go map, but
uses drastically less space. Using the same word list as above, the
hash/displace algorithm takes 8 *bytes* per entry; total space about 2MB.
This algorithm only takes 3.7 *bits*, for a total of about 110KB.

My implementation is here: https://github.com/dgryski/go-boomphf
