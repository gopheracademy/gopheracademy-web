+++
author = ["Matt Layher"]
date = "2017-12-15T08:00:00+00:00"
title = "unsafe.Pointer and system calls"
series = ["Advent 2017"]
+++

`unsafe` is a Go package that, as the official documentation states, contains
operations that step around the type safety of Go programs.

As its name implies, it should be used very carefully; `unsafe` can be
dangerous, but it can also be incredibly useful. For example, when working with
system calls and Go structures that must have an identical memory layout to a C
structure, you may have no choice but to resort to `unsafe`.

The `unsafe.Pointer` type allows you to bypass Go's type system and enables
conversion between arbitrary types and the `uintptr` built-in type. Per the
documentation, there are four operations available for `unsafe.Pointer` that
cannot be performed with other types:

* A pointer value of any type can be converted to an `unsafe.Pointer`.
* An `unsafe.Pointer` can be converted to a pointer value of any type.
* A `uintptr` can be converted to an `unsafe.Pointer`.
* An `unsafe.Pointer` can be converted to a `uintptr`.

This will focus on two useful operations that can only be performed when
employing the help of package `unsafe`: using `unsafe.Pointer` to convert
between two types and using `unsafe.Pointer` with system calls.

## Type conversions with unsafe.Pointer

### Mechanics

One of the most useful reasons to employ `unsafe.Pointer` is to enable an
expedient and concise conversion between two types that share the same layout
in memory.

The documentation states:

> Provided that T2 is no larger than T1 and that the two share an equivalent memory layout, this conversion allows reinterpreting data of one type as data of another type.

The classic example, and the one used in the documentation, is the
implementation of `math.Float64bits`:

```go
func Float64bits(f float64) uint64 {
	return *(*uint64)(unsafe.Pointer(&f))
}
```

This seems like a very concise way to achieve this conversion, but what is
actually happening here? Let's break it down, piece-by-piece:

- `&f` takes a pointer to the `float64` value stored in `f`.
- `unsafe.Pointer(&f)` converts the `*float64` to an `unsafe.Pointer`.
- `(*uint64)(unsafe.Pointer(&f))` converts the `unsafe.Pointer` to `*uint64`.
- `*(*uint64)(unsafe.Pointer(&f))` dereferences the `*uint64`, yielding a `uint64` value.

The expression shown in this first example is a very concise way of performing
the following:

```go
func Float64bits(floatVal float64) uint64 {
	// Take a pointer to the float64 value stored in f.
	floatPtr := &floatVal

	// Convert the *float64 to an unsafe.Pointer.
	unsafePtr := unsafe.Pointer(floatPtr)

	// Convert the unsafe.Pointer to *uint64.
	uintPtr := (*uint64)(unsafePtr)

	// Dereference the *uint64, yielding a uint64 value.
	uintVal := *uintPtr

	return uintVal
}
```

This is an extremely useful operation, and sometimes, a necessary one.

Now that you understand the mechanics of how `unsafe.Pointer` works, let's
walk through a real world example.

### A real world example: taskstats

I was recently exploring the [Linux taskstats interface](https://www.kernel.org/doc/Documentation/accounting/taskstats.txt),
and I wanted a way to retrieve the kernel's C `taskstats` struct in Go.  After
sending a CL to add the structure to `x/sys/unix`, I realized just how 
[large and complicated](https://godoc.org/golang.org/x/sys/unix#Taskstats) this
structure actually was.

In order to use this structure, I would need to painstakingly parse each field
from a byte slice.  To further complicate things, each integer type is stored
in "native" [endianness](https://en.wikipedia.org/wiki/Endianness), so the
integers may be stored in a different format in memory depending on your CPU.

This task is a great fit for a concise `unsafe.Pointer` conversion. Here's
what I wrote:

```go
// Verify that the byte slice containing a unix.Taskstats is the
// size expected by this package, so we don't blindly cast the
// byte slice into a structure of the wrong size.
const sizeofTaskstats = int(unsafe.Sizeof(unix.Taskstats{}))

if want, got := sizeofTaskstats, len(buf); want != got {
	return nil, fmt.Errorf("unexpected taskstats structure size, want %d, got %d", want, got)
}

stats := *(*unix.Taskstats)(unsafe.Pointer(&buf[0]))
```

How does it work?

First, I determine the exact size that the structure would occupy in memory
using `unsafe.Sizeof` by passing an instance of the structure as an argument.

Next, I verify that the byte slice being converted is exactly the same length
as the size of the `unix.Taskstats` structure. This ensures that I only retrieve
the exact data I want, and that I don't read arbitrary memory.

Finally, I perform the `unsafe.Pointer` conversion to a `unix.Taskstats` structure.

But why do I have to specify index 0 of the slice?

If you're familiar with [slice internals](https://blog.golang.org/go-slices-usage-and-internals),
you'll know that a slice is actually a header and a pointer to an underlying
array.  When converting slice data using `unsafe.Pointer`, you have to specify
the *memory address of the first element of the array*, not the slice header
itself.

Using `unsafe` this made the conversion extremely concise and simple.  Because
integer data is stored with the same endianness as our CPU, converting using
`unsafe.Pointer` means that the integer values will be what we expect.

You can see this code in action in my
[taskstats](https://github.com/mdlayher/taskstats) package.

## System calls with unsafe.Pointer

### Mechanics

When working with system calls, it is sometimes necessary to pass a pointer to
some memory to the kernel to allow it to perform some task.  This is another
vital use case for `unsafe.Pointer` in Go.  It is necessary to use
`unsafe.Pointer` when working with system calls because it can be converted to
`uintptr` for use with the [`syscall.Syscall`](https://golang.org/pkg/syscall/#Syscall)
family of functions.

There are a large number of system calls for many different operating systems,
but for this example, we will focus on [`ioctl`](https://en.wikipedia.org/wiki/Ioctl).
`ioctl`, in UNIX-like systems, is usually used to perform operations on file
descriptors that don't cleanly map to typical filesystem operations, like
`read` and `write`.  In fact, because of its immense flexibility, the bare
`ioctl` system call is not present in Go's `syscall` or `x/sys/unix` packages.

Let's walk through another real world example.

### A real world example: ioctl/vsock

In the past few years, Linux has gained a new socket family, `AF_VSOCK`, which
enables bi-directional, many-to-one communication between a hypervisor and its
virtual machines.

These sockets use a context ID for communication. The context ID can be
retrieved by sending an `ioctl` with a special request number to the
`/dev/vsock` device.

Here's the definition of the `ioctl` function:

```go
func Ioctl(fd uintptr, request int, argp unsafe.Pointer) error {
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		fd,
		uintptr(request),
		// Note that the conversion from unsafe.Pointer to uintptr _must_
		// occur in the call expression.  See the package unsafe documentation
		// for more details.
		uintptr(argp),
	)
	if errno != 0 {
		return os.NewSyscallError("ioctl", fmt.Errorf("%d", int(errno)))
	}

	return nil
}
```

As noted by the comment, there is an important caveat to the use of
`unsafe.Pointer` in this scenario:

> The Syscall functions in package syscall pass their uintptr arguments directly to the operating system, which then may, depending on the details of the call, reinterpret some of them as pointers. That is, the system call implementation is implicitly converting certain arguments back from uintptr to pointer.
>
> **If a pointer argument must be converted to uintptr for use as an argument, that conversion must appear in the call expression itself.**

But why is this the case?  This is special pattern recognized by the compiler
that essentially instructs the garbage collector to not re-arrange the memory
referenced by the pointer until the function call completes.

You can read the documentation for more technical detail, but you must always
remember this rule when working with system calls in Go.  In fact, I realized
while I was authoring this post that my code was technically in violation of
this rule!  This has now been fixed.

With this in mind, we can see how this function could be useful.

In the case of VM sockets, we want to pass a `*uint32` to the kernel so that
it can populate the value at that memory address with our local context ID.

```go
f, err := fs.Open("/dev/vsock")
if err != nil {
	return err
}
defer f.Close()

// Context ID is populated by Ioctl.
var cid uint32

// Retrieve the context ID of this machine from /dev/vsock.
err = Ioctl(f.Fd(), unix.IOCTL_VM_SOCKETS_GET_LOCAL_CID, unsafe.Pointer(&cid))
if err != nil {
	return err
}

// Return the now-populated context ID to the caller.
return cid, nil
```

This is just one example of using `unsafe.Pointer` with system calls.
You can use this pattern to send and receive arbitrary data, or to configure a
kernel interface in some special way.  The possibilites are almost endless!

You can see this code in action in my [vsock](https://github.com/mdlayher/vsock)
package.

## Conclusion

Although using package `unsafe` can be fraught with peril, it can be an
extremely powerful and useful tool when applied properly.

Now that you've read this post, I encourage you to
[read the official `unsafe` documentation](https://golang.org/pkg/unsafe/)
thoroughly before employing it in your programs.

If you have any questions, feel free to contact me! I'm mdlayher on
[Gophers Slack](https://gophers.slack.com/), [GitHub](https://github.com/mdlayher)
and [Twitter](https://twitter.com/mdlayher).

Special thanks to [Hazel Virdó](https://twitter.com/HazelVirdo) for her feedback
and editing of this article!

## Links

- [Package unsafe](https://golang.org/pkg/unsafe/)
- [Endianness](https://en.wikipedia.org/wiki/Endianness)
- [Package taskstats](https://github.com/mdlayher/taskstats)
- [Go Slices: usage and internals](https://blog.golang.org/go-slices-usage-and-internals)
- [ioctl](https://en.wikipedia.org/wiki/Ioctl)
- [Package vsock](https://github.com/mdlayher/vsock)