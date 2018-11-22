+++
author = ["Alexander Neumann"]
title = "Splitting Data with Content-Defined Chunking"
date = 2018-12-04T00:00:00Z
series = ["Advent 2018"]
+++

This post you'll learn what Content-Defined Chunking (CDC) is and how you can use it to split large data into smaller blocks in a deterministic way. These blocks can be found again in other data later, even if the location of the block is different than the first time. I wrote a small Go package to do the splitting, which performs really well.

Why is splitting data an interesting problem?
=============================================

In my spare time, I'm working on a fast backup program called [restic](https://restic.net), which is written in Go. When a (possible large) file is read, the program needs to save the data somewhere so that it can be restored later. I think it is a good idea to split files into smaller parts, which are more manageable for a program and it allows detecting and efficiently handling small changes in big files, like virtual machine images. Once such a part of a file is saved, it can be referenced when the same data is contained in different files, so restic de-duplicates the data it reads. The parts are identified based on the SHA-256 hash of the contents, so a list of these hashes can describe the content of a file.

There are different strategies for splitting files, the most obvious one would be to just use static boundaries, e.g. after every megabyte of data. This gives us manageable chunks, but if the data in the file is shifted, for example by inserting a byte at the start of the file, all chunks will be different and need to be saved anew. We can do better than that.

Rabin fingerprints
==================

During the research I did before starting to implement what would later become restic, I discovered a publication titled ["Fingerprinting by Random Polynomials"](http://www.xmailserver.org/rabin.pdf) by [Michael O. Rabin](https://en.wikipedia.org/wiki/Michael_O._Rabin) published in 1981. It describes a way to efficiently compute a ["Rabin fingerprint"](https://en.wikipedia.org/wiki/Rabin_fingerprint) (or "hash", in modern terms) for a number of bytes. It has the great property that it can be computed as a [rolling hash](https://en.wikipedia.org/wiki/Rolling_hash): the Rabin fingerprint of a window of bytes can be efficiently calculated while the window moves through a very large file. The idea is that when a new byte is to be processed, the oldest byte is first removed from the calculation, and the new byte is factored in.

How can we use that to find chunk boundaries? Restic uses a window of 64 bytes, so while reading a file, it computes the fingerprint of all such windows: byte 0 to byte 63, byte 1 to byte 64 and so on. It then checks every fingerprint if the least significant bits are all zero. If this is the case, then we found a chunk boundary! For random input bytes, the calculated fingerprints are roughly random as well, so by varying the number of bits that are checked we can (roughly) configure how large the resulting chunks will be. Since the cut points for individual chunks only depend on the 64 bytes preceding them, this method of splitting data is called "Content-Defined Chunking" (CDC).

Early on I recognized that the algorithm and implementation has the potential to be very helpful to other projects as well, so I published it as a [separate package](https://github.com/restic/chunker) for other people to import ([API on godoc.org](https://godoc.org/github.com/restic/chunker)) with a very liberal BSD 2-clause license.

Enough theory, let's dive into the code! 

Splitting data
==============

The type which does all the hard work of computing the fingerprints over the sliding window is contained in the `Chunker` type. The Rabin fingerprint needs a so-called "irreducible polynomial" to work, which is encoded as an `uint64`. The `chunker` package exports the function `RandomPolynomial()` which creates a new polynomial for you to use.

Our first program will generate a random polynomial and create a new `Chunker` which reads data from `os.Stdin`:

```go
p, err := chunker.RandomPolynomial()
if err != nil {
	panic(err)
}

fmt.Printf("using polynomial %v for splitting data\n", p)

chk := chunker.New(os.Stdin, p)
```

The `Next()` method on the `Chunker` reads data out the reader (`os.Stdin` in this example) into a byte slice buffer. The methods returns a next `Chunk` and an error. We call it repeatedly until `io.EOF` is returned, which tells us that all data has been read:

```go
buf := make([]byte, 16*1024*1024) // 16 MiB
for {
	chunk, err := chk.Next(buf)
	if err == io.EOF {
		break
	}

	if err != nil {
		panic(err)
	}

	fmt.Printf("%d\t%d\t%016x\n", chunk.Start, chunk.Length, chunk.Cut)
}
```

Let's get some random data to test:

```
$ dd if=/dev/urandom bs=1M count=100 of=data
100+0 records in
100+0 records out
104857600 bytes (105 MB, 100 MiB) copied, 0,548253 s, 191 MB/s
```

Now if we feed the data in this file to the program we've just written ([code](https://gist.github.com/fd0/a74e42a7a4f51c4ccaa6c11a09c14619)), it prints the start offset, length, and fingerprint for the chunk it found:

```
$ cat data | go run main.go
using polynomial 0x3dea92648f6e83 for splitting data
0       2908784 000c31839a100000
2908784 2088949 001e8b4104900000
4997733 1404824 000a18d1f0200000
6402557 617070  001bc6ac84300000
7019627 1278326 001734984b000000
8297953 589913  0004cf802b600000
8887866 554526  001eb5a362900000
9442392 1307416 000ef1e549c00000
[...]
102769045       864607  00181b67df300000
103633652       592134  0000a0c8b4200000
104225786       631814  001d5ba20d38998b
```

On a second run, it'll generate a new polynomial and the chunks will be different, so if you depend on your program computing the same boundaries, you'll need to pass in the same polynomial.

Let's fix the polynomial (`0x3dea92648f6e83`) for the next runs and change the program like this ([code](https://gist.github.com/bf984e1c40b56eaeff310d07a0d71128)):

```go
chk := chunker.New(os.Stdin, 0x3dea92648f6e83)
```

After replacing the randomly generated polynomial with a constant, every new run will give us the same chunks:

```
$ cat data | go run main.go
0       2908784 000c31839a100000
2908784 2088949 001e8b4104900000
4997733 1404824 000a18d1f0200000
[...]
102769045       864607  00181b67df300000
103633652       592134  0000a0c8b4200000
104225786       631814  001d5ba20d38998b
```

Now we can experiment with the data. For example, what happens when we insert bytes at the beginning? Let's test that by inserting the bytes `foo` using the shell:
```
$ (echo -n foo; cat data) | go run main.go
0       2908787 000c31839a100000
2908787 2088949 001e8b4104900000
4997736 1404824 000a18d1f0200000
6402560 617070  001bc6ac84300000
[...]
```

We can see that the first chunk is different: it's three bytes longer (2908787 versus 2908784), but the fingerprint at the end of the chunk is the same. All other chunks following the first one are also the same!

Let's add a hash to identify the contents of the individual chunks ([code](https://gist.github.com/1e06773ef15cac6e16efe0c291c8f4b4)):

```go
for {
	chunk, err := chk.Next(buf)
	if err == io.EOF {
		break
	}

	if err != nil {
		panic(err)
	}

	hash := sha256.Sum256(chunk.Data)
	fmt.Printf("%d\t%d\t%016x\t%032x\n", chunk.Start, chunk.Length, chunk.Cut, hash)
}
```

These are the chunks for the raw, unmodified file:
```
$ cat data | go run main.go
0       2908784 000c31839a100000        8d17f5f7326a2fd7[...]
2908784 2088949 001e8b4104900000        ba7c71226609c0de[...]
4997733 1404824 000a18d1f0200000        9c4db21626a27d4b[...]
6402557 617070  001bc6ac84300000        4c249c3b88500c23[...]
7019627 1278326 001734984b000000        5bec562b7ad37b8b[...]
[...]
```

Now we can see that adding bytes at the start only changes the first chunk:

```
$ (echo -n foo; cat data) | go run main.go
0       2908787 000c31839a100000        f41084d25bc273f2[...]
2908787 2088949 001e8b4104900000        ba7c71226609c0de[...]
4997736 1404824 000a18d1f0200000        9c4db21626a27d4b[...]
6402560 617070  001bc6ac84300000        4c249c3b88500c23[...]
7019630 1278326 001734984b000000        5bec562b7ad37b8b[...]
```

The SHA-256 hash for the first chunk changed from `8d17f5f7326a2fd7[...]` to `f41084d25bc273f2[...]`, but the rest are still the same. That's pretty cool.

Let's see how many different chunks we have in our data file by counting the different hashes:
```
$ cat data | go run main.go | cut -f4 | sort | uniq | wc -l
69
```

What happens when we feed the program our data twice, how many chunks does it detect? Let's find out:

```
$ (cat data; cat data) | go run main.go | cut -f4 | sort | uniq | wc -l
70
```

Huh, so we only got a single additional chunk. The cause is that for the first round, it'll find the same chunks as before (it's the same data after all) right until the last chunk. In the previous run, the end of the last chunk was determined by the end of the file. Now there's additional data, so the chunk continues. The next cut point will in the second run of the data, it'll be the same as the first cut point, just with some additional data at the beginning of the chunk, so the SHA-256 hash is different. Afterwards, the same chunks follow in the same order.

Changing a few consecutive bytes of data somewhere in most cases also only affects a single chunk, let's write the output to a file, change the data a bit using `sed`, and write another file:

```
$ cat data | go run main.go > orig.log
$ cat data | sed 's/EZE8HX/xxxxxx/' | go run main.go > mod.log
```

The string `EZE8HX` was present in my randomly generated file `data` only once and the `sed` command above changed it to the string `xxxxxx`. When we now compare the two log files using `diff`, we can see that for exactly one chunk the SHA-256 hash has changed, but the other chunks and the chunk boundaries stayed the same:

```diff
$ diff -au orig.log mod.log
--- orig.log    2018-11-24 13:14:45.265329525 +0100
+++ mod.log     2018-11-24 13:19:32.344681407 +0100
@@ -55,7 +55,7 @@
 83545247       547876  00198b8839f00000        2cbeea803ba79d54[...]
 84093123       889631  0007ddacabc00000        a2b6e8651ae1ab69[...]
 84982754       2561935 000c53712f500000        bca589959b019a80[...]
-87544689       2744485 0000a49118b00000        02b125104f06ed85[...]
+87544689       2744485 0000a49118b00000        ce2e43496ae6e7c3[...]
 90289174       1167308 00034d6cce700000        ad25a331993d9db7[...]
 91456482       1719951 001d2fea8ae00000        ba5153845c5b5228[...]
 93176433       1362655 0003009fc1600000        8e7a35e340c10d61[...]
```

This technique can be used for many other things besides a backup program. For example, the program `rsync` uses content-defined chunking to efficiently transfer files by detecting which parts of the files are already present on the receiving side (with a different rolling hash). The [Low Bandwidth Network File System (LBFS)](https://pdos.csail.mit.edu/papers/lbfs:sosp01/lbfs.pdf) uses CDC with Rabin fingerprints to only transfer chunks that are needed over the network. Another application of a rolling hash is finding strings in text, for example in the [Rabin-Karp algorithm](https://en.wikipedia.org/wiki/Rabin%E2%80%93Karp_algorithm).

If you're interested in how the `chunker` package is used in restic, there's a [post over at the restic blog](https://restic.net/blog/2015-09-12/restic-foundation1-cdc) which explains this in a bit more detail.

Performance
===========

Performance is crucial for backup programs. If creating backups is too slow people tend to stop using it.

The `chunker` package is written in plain Go. It doesn't have any fancy low-level assembler, but the code is already pretty fast, thanks to some optimized calculations. The package has some benchmarks:

```
$ go test -run xxx -bench 'BenchmarkChunker$'
goos: linux
goarch: amd64
pkg: github.com/restic/chunker
BenchmarkChunker-4            20          70342669 ns/op         477.01 MB/s
--- BENCH: BenchmarkChunker-4
    chunker_test.go:325: 23 chunks, average chunk size: 1458888 bytes
    chunker_test.go:325: 23 chunks, average chunk size: 1458888 bytes
PASS
ok      github.com/restic/chunker       1.655s
```

Running on an older machine with an Intel i5 CPU `chunker` processes at about 477 MB per second on a single core.

Conclusion
==========

Content-Defined Chunking can be used to split data into smaller chunks in a deterministic way so that the chunks can be rediscovered if the data has slightly changed, even when new data has been inserted or data was removed. The [`chunker` package](https://github.com/restic/chunker) provides a fast implementation in pure Go with a liberal license and a simple API. If you use the `chunker` package for something cool, please let us know by [opening an issue](https://github.com/restic/chunker/issues/new) in the repo over at GitHub!

