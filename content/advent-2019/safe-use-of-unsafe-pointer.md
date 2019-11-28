+++
date = "2019-12-05T00:00:00+00:00"
title = "Safe use of unsafe.Pointer"
subtitle = "Using Go tools to safely write unsafe code."
+++

Package [`unsafe`](https://golang.org/pkg/unsafe/) provides an escape hatch from
Go's type system, enabling interactions with low-level and system call APIs, in
a manner similar to C programs. However, `unsafe` has several rules which must
be followed in order to perform these interactions in a sane way. It's easy to
make [subtle mistakes when writing `unsafe` code](https://github.com/golang/sys/commit/b69606af412f43a225c1cf2044c90e317f41ae09),
but these mistakes can often be avoided.

This blog post will introduce some of the current and upcoming Go tooling that can
verify safe usage of the `unsafe.Pointer` type in your Go programs. If you have
not worked with `unsafe` before, I recommend reading my
[previous Gopher Academy Advent series blog](https://blog.gopheracademy.com/advent-2017/unsafe-pointer-and-system-calls/)
on the topic.

Extra care and caution must be taken whenever introducing `unsafe` to a code
base, but these diagnostic tools can help you solve problems before they lead to
a major bug or a possible security flaw in your application.

## Compile-time verification with `go vet`

For several years now, the `go vet` tool has had the ability to check for
invalid conversions between the `unsafe.Pointer` and `uintptr` types. 

Let's take a look at an example program. Suppose we would like to use pointer
arithmetic to iterate over and print each element of an array:

```go
package main

import (
    "fmt"
    "unsafe"
)

func main() {
    // An array of contiguous uint32 values stored in memory.
    arr := []uint32{1, 2, 3}

    // The number of bytes each uint32 occupies: 4.
    const size = unsafe.Sizeof(uint32(0))

    // Take the initial memory address of the array and begin iteration.
    p := uintptr(unsafe.Pointer(&arr[0]))
    for i := 0; i < len(arr); i++ {
        // Print the integer that resides at the current address and then
        // increment the pointer to the next value in the array.
        fmt.Printf("%d ", (*(*uint32)(unsafe.Pointer(p))))
        p += size
    }
}
```

At first glance, this program appears to work as expected, and we can see each
of the array's elements printed to our terminal:

```
$ go run main.go 
1 2 3
```

However, there is a subtle flaw in this program. What does `go vet` have to say?

```
$ go vet .
# github.com/mdlayher/example
./main.go:20:33: possible misuse of unsafe.Pointer
```

In order to understand this error, we must consult [the rules](https://golang.org/pkg/unsafe/#Pointer)
of the `unsafe.Pointer` type:

> Converting a Pointer to a uintptr produces the memory address of the value
> pointed at, as an integer. The usual use for such a uintptr is to print it.
>
> Conversion of a uintptr back to Pointer is not valid in general.
>
> A uintptr is an integer, not a reference. Converting a Pointer to a uintptr
> creates an integer value with no pointer semantics. Even if a uintptr holds
> the address of some object, the garbage collector will not update that
> uintptr's value if the object moves, nor will that uintptr keep the object
> from being reclaimed.

We can isolate the offending portion of our program as follows:

```go
p := uintptr(unsafe.Pointer(&arr[0]))

// What happens if there's a garbage collection here?
fmt.Printf("%d ", (*(*uint32)(unsafe.Pointer(p))))
```

Because we store the `uintptr` value in `p` but do not immediately make use of
that value, it's possible that when a garbage collection occurs, the address
(now stored in `p` as a uintptr integer) will no longer be valid!

Let's assume this scenario has occurred and that `p` no longer points at a
`uint32`. Perhaps when we reinterpret `p` as a pointer, the memory pointed at is
now being used to store a user's authentication credentials or a TLS private key.
We've introduced a potential security flaw in our application and could easily
leak sensitive material to an attacker through a normal channel such as `stdout`
or an HTTP API's response body.

In effect, once we've converted an `unsafe.Pointer` to `uintptr`, we cannot
safely convert it back to `unsafe.Pointer`, with the exception of one special
case:

> If p points into an allocated object, it can be advanced through the object by
> conversion to uintptr, addition of an offset, and conversion back to Pointer.

In order to perform this pointer arithmetic iteration logic safely, we must
perform the type conversions and pointer arithmetic all at once:

```go
package main

import (
    "fmt"
    "unsafe"
)

func main() {
    // An array of contiguous uint32 values stored in memory.
    arr := []uint32{1, 2, 3}

    // The number of bytes each uint32 occupies: 4.
    const size = unsafe.Sizeof(uint32(0))

    for i := 0; i < len(arr); i++ {
        // Print an integer to the screen by:
        //   - taking the address of the first element of the array
        //   - applying an offset of (i * 4) bytes to advance into the array
        //   - converting the uintptr back to *uint32 and dereferencing it to
        //     print the value
        fmt.Printf("%d ", *(*uint32)(unsafe.Pointer(
            uintptr(unsafe.Pointer(&arr[0])) + (uintptr(i) * size),
        )))
    }
}
```

This program produces the same result as before, but is now considered valid by
`go vet` as well!

```
$ go run main.go 
1 2 3 
$ go vet .
```

I don't recommend using pointer arithmetic for iteration logic in this way, but
it's excellent that Go provides this escape hatch (and tooling for using it
safely!) when it is truly needed.

## Run-time verification with the Go compiler's `checkptr` debugging flag

The Go compiler recently gained support for a [new debugging flag](https://go-review.googlesource.com/c/go/+/162237)
which can instrument uses of `unsafe.Pointer` to detect invalid usage patterns
at runtime. As of Go 1.13, this feature is unreleased, but can be used by
installing Go from tip:

```
$ go get golang.org/dl/gotip
go: finding golang.org/dl latest
...
$ gotip download
Updating the go development tree...
...
Success. You may now run 'gotip'!
$ gotip version
go version devel +8054b13 Thu Nov 28 15:16:27 2019 +0000 linux/amd64
```

Let's review another example. Suppose we are passing a Go structure to a Linux
kernel API which would typically accept a C union. One pattern for doing this is
to have an overarching Go structure which contains a raw byte array (mimicking a
C union), and then to create typed variants for possible argument combinations.

```go
package main

import (
    "fmt"
    "unsafe"
)

// one is a typed Go structure containing structured data to pass to the kernel.
type one struct{ v uint64 }

// two mimics a C union type which passes a blob of data to the kernel.
type two struct{ b [32]byte }

func main() {
    // Suppose we want to send the contents of a to the kernel as raw bytes.
    in := one{v: 0xff}
    out := (*two)(unsafe.Pointer(&in))

    // Assume the kernel will only access the first 8 bytes. But what is stored
    // in the remaining 24 bytes?
    fmt.Printf("%#v\n", out.b[0:8])
}
```

When we run this program on a stable version of Go (as of Go 1.13.4), we can see
that the first 8 bytes of the array contain our `uint64` data in its native
endian format (little endian on my machine):

```
$ go run main.go
[]byte{0xff, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}
```

However, it turns out there is an issue with this program as well. If we attempt
to run the program on Go tip with the `checkptr` debug flag, we will see:

```
$ gotip run -gcflags=all=-d=checkptr main.go 
panic: runtime error: unsafe pointer conversion

goroutine 1 [running]:
main.main()
        /home/matt/src/github.com/mdlayher/example/main.go:17 +0x60
exit status 2
```

This check is still quite new and as such does not provide much information
beyond the "unsafe pointer conversion" panic message and a stack trace. But the
stack trace does at least provide a hint that line 17 is suspect.

By casting a smaller structure into a larger one, we enable reading arbitrary
memory beyond the end of the smaller structure's data! This is another way that
careless use of `unsafe` could result in a security vulnerability in your
application.

In order to perform this operation safely, we have to make sure that we
initialize the "union" structure first before copying data into it, so we can
ensure that arbitrary memory is not accessed:

```go
package main

import (
    "fmt"
    "unsafe"
)

// one is a typed Go structure containing structured data to pass to the kernel.
type one struct{ v uint64 }

// two mimics a C union type which passes a blob of data to the kernel.
type two struct{ b [32]byte }

// newTwo safely produces a two structure from an input one.
func newTwo(in one) *two {
    // Initialize out and its array.
    var out two

    // Explicitly copy the contents of in into out by casting both into byte
    // arrays and then slicing the arrays. This will produce the correct packed
    // union structure, without relying on unsafe casting to a smaller type of a
    // larger type.
    copy(
        (*(*[unsafe.Sizeof(two{})]byte)(unsafe.Pointer(&out)))[:],
        (*(*[unsafe.Sizeof(one{})]byte)(unsafe.Pointer(&in)))[:],
    )

    return &out
}

func main() {
    // All is well! The two structure is appropriately initialized.
    out := newTwo(one{v: 0xff})

    fmt.Printf("%#v\n", out.b[:8])
}
```

We can now run our updated program with the same flags as before, and we will
see that the issue has been resolved:

```
$ gotip run -gcflags=all=-d=checkptr main.go 
[]byte{0xff, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}
```

By removing the slicing operation from the `fmt.Printf` call, we can verify that
the remainder of the byte array has been initialized to `0` bytes:

```
[32]uint8{
	0xff, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
}
```

This is a very easy mistake to make, and in fact, I recently had to fix this
exact issue in code I wrote for [the tests in `x/sys/unix`](https://github.com/golang/sys/commit/b69606af412f43a225c1cf2044c90e317f41ae09)!
I've written a fair amount of `unsafe` code in Go, but even veteran programmers
can easily make mistakes. This is why these types of diagnostic tools are so
important!

## Conclusion

Package `unsafe` is a very powerful tool with a razor-sharp edge, but it should
not be feared. When interacting with Linux kernel system call APIs, it is often
necessary to resort to `unsafe` code. Making effective use of tools such as
`go vet` and the `checkptr` compiler debugging flag is crucial in order to
ensure safety in your applications.

If you work with `unsafe` code on a regular basis, I highly recommend joining
the [`#darkarts` channel on Gophers Slack](https://invite.slack.golangbridge.org/).
There are a lot of veterans in that channel who have helped me learn how to make
effective use of `unsafe` in my own applications.

If you have any questions, feel free to contact me! I'm mdlayher on
[Gophers Slack](https://gophers.slack.com/), [GitHub](https://github.com/mdlayher)
and [Twitter](https://twitter.com/mdlayher).

Special thanks to:

- [Cuong Manh Le (@cuonglm)](https://github.com/cuonglm) for his insight regarding the [`=all` modifier for the `checkptr` debugging flag](https://github.com/gopheracademy/gopheracademy-web/pull/332#discussion_r351896035)
- [Miki Tebeka (@tebeka)](https://github.com/tebeka) for review of this post

## Links

* [Package `unsafe`](https://golang.org/pkg/unsafe/)
* [Gopher Academy: unsafe.Pointer and system calls](https://blog.gopheracademy.com/advent-2017/unsafe-pointer-and-system-calls/)
* [cmd/compile: add -d=checkptr to validate unsafe.Pointer rules](https://go-review.googlesource.com/c/go/+/162237)
