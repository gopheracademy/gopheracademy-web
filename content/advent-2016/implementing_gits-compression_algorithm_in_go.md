+++
author = ["Aditya Mukerjee"]
date = "2016-12-22T00:00:00"
linktitle = "Implementing Git's Compression in Go"
title = "Implementing Git's Compression In Go"
series = ["Advent 2016"]

+++

# Implementing Git's Compression in Go


## Git + Go = Gitgo

[Gitgo](https://github.com/ChimeraCoder/gitgo) is a pure Go library for core Git functions. As a ground-up implementation with no C bindings, it implements the entire structure of Git repositories from scratch. This includes an arcane part of Git that every Git user uses on a daily basis: _packfiles_.

Packfiles are a custom-designed file format that Git uses internally for compressing and storing binary diffs. The original goal is to reduce network usage and make `git clone` and `git fetch` fast. Packfiles are a delightful balance of elegance and complexity – layering its own custom deduplication logic on top of a more standard compression algorithm (zlib). If you're interested in learning more about the innards of packfiles, I've [written at length about how they work](https://codewords.recurse.com/issues/three/unpacking-git-packfiles). But for this Advent post, I'm going to talk about three patterns we can learn from the experience of implementing a compression file format in Go, which are all generalizable to other Go code.


## Bits, bytes, and nibbles


Go is strongly inspired by C – a language which makes heavy use of pointer arithmetic and bitwise operations. As a memory-safe language, pointer arithmetic is all-but-nonexistent in Go. But that doesn't mean we need to avoid bitwise operations as well. In Go, bitwise operations can be used to consolidate and optimize code.

There's a fine line to walk here – overusing bitwise operations can lead to absolutely inscrutable code, so it's important to use discretion. Go is inspired by C, but that doesn't mean we use pointer arithmetic everywhere in Go code! 

At the same time, sometimes the tradeoff in readability is worth it – such as if your application has very tight space requirements. That could mean network usage, as it does with Git, or it could be for situations in which you are optimizing for memory consumption or disk utilization. For example, here's how we decode part of the packfile compression header in Gitgo, which uses bitwise operations to encode the size (a number) in as few bytes as possible. It also makes use of the three bits of “extra” space in the first byte to tell us the object type.

In other words, the byte itself is separated into three parts, each of which contains completely different pieces of information. Since we know that most of this information is well under 255 (the largest number that can fit in a single byte), we can save several bytes from each header by squeezing this information into the same byte. The result will look something like this:

```go
    // This will extract the last three bits of
    // the first nibble in the byte
    // which tells us the object type
    object._type = packObjectType(((_byte >> 4) & 7))

    // determine the (decompressed) object size
    // and then deflate the following bytes

    // The most-significant byte (MSB)
    // tells us whether we need to read more bytes
    // to get the encoded object size
    MSB := (_byte & 128) // will be either 128 or 0 // This will extract the last four bits of the byte
    var objectSize = int((uint(_byte) & 15))
```

When using this approach, documentation is key. Most code that involves bitwise operations – in any language, not just Go – tends to have about half the documentation it really needs in order to make the purpose of the code clear. Because Go as a language stresses readability, when trading off readability for performance, that factor is even larger. Write about twice the comments and twice the amount of contextual documentation that you think you need. It's a pretty frustrating feeling to stumble on code that someone else has written and having to peruse the output of [od](https://en.wikipedia.org/wiki/Od_%28Unix%29) in order to piece together what the code is supposed to do. Perhaps the only feeling worse that that one is when you find yourself in that same situation with code that you wrote months (or years) ago.

So, don't be afraid to use bit-level operations to consolidate your code. You'll get the benefits of lower memory and/or network usage, and your code may even execute faster as well. Just make sure it's all documented!


## Treat errors as values


In Go, errors are values, just like any other. There's nothing special about them, which means that we're not restricted in how we choose to handle them – we can take full advantage of the entire language. 


That doesn't mean we _can't_ do:

```go

if err != nil{
	return err
}

```
but it does mean we can take advantage of more sophisticated approaches as well.

Rob Pike [wrote about this a while back](https://blog.golang.org/errors-are-values), and gave a few examples of alternative methods. One of these alternatives is used in the critical path for the packfile parsing in Gitgo:

```go
type errReadSeeker struct {
        r   io.ReadSeeker
        err error
}

// Read, but only if no errors have been encountered
// in a previous read (including io.EOF)
func (er *errReadSeeker) read(buf []byte) int {
        var n int
        if er.err != nil {
                return 0
        }
        n, er.err = io.ReadFull(er.r, buf)
        return n
}
```

Parsing packfiles is done in stages, which means we don't want to have to do an error check after every single read operation. Instead, the local `errReadSeeker` type allows us to assume all read operations succeed, and only check once, at the very end, for errors.



## Ask Errors how they behave

As we already saw, by treating errors as values, we're able to eliminate the repetitive calls to:


```go
_, err := r.Read(b)
if err != nil{
	// do something
}

// do some stuff with the first chunk of data

err = r.Read(b)
if err != nil{
	// do something
}

// do some different stuff with the second chunk of data
```


But it also helps us use more expressive patterns for error handling. For example, we might want to handle certain errors differently. Let's say we're reading packfiles that may exist either locally or over a network. If our network connection drops, we want to be able to replay our request with a slight delay. But this is less useful for local storage, where an error is more likely to mean that the file is missing or the drive is corrupted – in that case, we want to fail immediately, without waiting to retry.

Treating errors as values gives us another pattern we can use to dispatch our error handling logic: local interface types.

 

```go
type replayable interface {
	Delay() int64
}


func (er *errReadSeeker) read(buf []byte) int {
	var n interface
	if er.err != nil {
		if replayable, ok := err.(replayable); ok {
			← time.After(replayable.Delay())
			er.err = nil
		} else {
			return 0
		}
	}
	n, er.err = io.ReadFull(er.r, buf)
	return n
}

```

Especially if the reading logic is used by many different functions in different locations, the benefits of treating errors as values that can store state is immense. We don't need to duplicate this complicated logic everywhere, and we can even store other forms of information in addition to the delay. (For example, we'd probably also want to add some timeout logic to avoid an infinite loop of retries.)

The interesting thing to note here is that the `replayable` interface is entirely local – the underlying libraries providing the network functions don't have to export it directly or even knowingly support it at all. In addition, we're able to handle this error appropriately even though it has to pass through the `io` package, which doesn't know anything about network error types. And finally, we're able to do this without ever even _knowing_ the specific underlying concrete type of the error, let alone using that in the function signature, since we know that [using concrete error types in return values can cause problems](https://golang.org/doc/faq#nil_error).



## Going further with Gitgo


With these three design patterns, I'm just scratching the surface of what we can learn from implementing Git in Go. There's a lot more that we could talk about as well, such as more nuanced uses of interface types or design strategies for concurrent systems - but we'll have to save that for another post.


In the meantime, though, if you'd like to write a Go application that interacts with Git repositories, or if you'd like to learn more about how Git really works under-the-hood by hacking on a ground-up implementation of Git in Go, take a look at [Gitgo](https://github.com/ChimeraCoder/gitgo)!


