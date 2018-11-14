+++
author = ["Matt Layher"]
date = "2018-12-16T08:00:00+00:00"
title = "Exploring byte parsing APIs in Go"
series = ["Advent 2018"]
+++

Several years ago, I began exploring Linux's Netlink inter-process communication
interface. Netlink is typically used for retrieving information from the Linux
kernel, and in order to cross the kernel boundary, information is typically
packed into Netlink attributes.  After some experimentation, I created my own
[netlink package](https://github.com/mdlayher/netlink) for Go.

Over time, the APIs in the package have evolved considerably.  In particular,
Netlink attributes have always been reasonably complicated to handle. Today,
we'll explore some of the byte parsing APIs I've created for dealing with
Netlink attributes. The techniques described here should be widely applicable
in many other Go libraries and applications as well!

## An introduction to Netlink attributes

Netlink attributes are packed in a type/length/value, or TLV, format, as is the
case with many binary network protocols.  This format enables great extensibility
because many attributes can be packed back-to-back in a single byte slice.

The value within an attribute can contain:

* An unsigned 8/16/32/64-bit integer
* A null-terminated C string
* Arbitrary C structure bytes
* Nested Netlink attributes
* A Netlink attribute array

For our purposes, we can define a Netlink attribute in Go as follows:

```go
type Attribute struct {
	// The type of this Attribute, typically matched to a constant.
	Type uint16

	// Length omitted; Data will be a byte slice of the appropriate length.

	// An arbitrary payload which is specified by Type.
	Data []byte
}
```

Today, we'll skip the low-level byte parsing logic in favor of discussing various
high level APIs, but you can learn more about dealing with Netlink attributes
from [my blog series about Netlink](https://medium.com/@mdlayher/linux-netlink-and-go-part-1-netlink-4781aaeeaca8).

## A first pass at a byte parsing API

A single byte slice can contain many Netlink attributes.  Let's define an initial
parsing function that accepts an input byte slice and returns a slice of Attributes.

```go
// UnmarshalAttributes unpacks a slice of Attributes from a single byte slice.
func UnmarshalAttributes(b []byte) ([]Attribute, error) {
	// ...
}
```

As an example, let's say we want to unpack a `uint16` and `string` value from the
slice of Attributes. You can safely ignore `parseUint16` and `parseString`;
they'll deal with some of the tricky parts of Netlink attribute data.

To unpack the attribute data, we can use a loop and match on the Type field:

```go
attrs, err := netlink.UnmarshalAttributes(b)
if err != nil {
	return err
}

var (
	num uint16
	str string
)

for _, a := range attrs {
	switch a.Type {
	case 1:
		num = parseUint16(a.Data[0:2])
	case 2:
		str = parseString(a.Data)
	}
}

fmt.Printf("num: %d, str: %q", num, str)
// num: 1, str: "hello world"
```

This works fine, but there's a catch: what happens if the byte slice for our
`uint16` value is more or less than 2 bytes?

```go
// A panic waiting to happen!
num = parseUint16(a.Data[0:2])
```

If it's shorter than two bytes, this code will panic and take down your
application.  If it's longer than two bytes, we're silently ignoring any extra
data (and this value is not actually a `uint16`!).

## Adding validation and error handling

Let's revise our parsing functions slightly. Each one should do some internal
validation, and if the byte slice doesn't meet our constraints, we can return
an error.

```go
attrs, err := netlink.UnmarshalAttributes(b)
if err != nil {
	return err
}

var (
	num uint16
	str string

	// Used to check for errors without shadowing num and str later.
	err error
)

for _, a := range attrs {
	// This works, but it's a bit verbose.
	// Be cautious of variable shadowing as well!
	switch a.Type {
	case 1:
		num, err = parseUint16(a.Data)
	case 2:
		str, err = parseString(a.Data)
	}
	if err != nil {
		return err
	}
}

fmt.Printf("num: %d, str: %q", num, str)
// num: 1, str: "hello world"
```

This also works, but you have to be cautious about your error checking strategy,
and also make sure you don't accidentally shadow one of the variables you're
trying to unpack by using the `:=` assignment operator.

Can we further improve upon this pattern?

## An iterator-like parsing API

The above strategies worked well for several years, but after writing a number of
Netlink interaction packages, I decided to start work on an improved API.

The new API uses an iterator-like pattern that is inspired by the
[`bufio.Scanner`](https://golang.org/pkg/bufio/#Scanner) API from the standard
library.  The [Go blog's Errors are values](https://blog.golang.org/errors-are-values)
post does an excellent job explaining this strategy as well.

The [`netlink.AttributeDecoder`](https://godoc.org/github.com/mdlayher/netlink#AttributeDecoder)
type is my take on an iterator-like parsing API.  After using the
`netlink.NewAttributeDecoder` constructor, several methods are exposed which
enable iteration over an internal attribute slice:

* `Next`: advance the internal pointer to the next attribute
* `Type`: return the type value of the current attribute
* `Err`: return the first error encountered during iteration

Let's revisit the previous example while trying out this new API:

```go
ad, err := netlink.NewAttributeDecoder(b)
if err != nil {
	return err
}

var (
	num uint16
	str string
)

// Continue advancing the internal pointer until done or error.
for ad.Next() {
	// Check the current attribute's type and extract it as appropriate.
	switch ad.Type() {
	case 1:
		// If data isn't a uint16, an error will be captured internally.
		num = ad.Uint16()
	case 2:
		str = ad.String()
	}
}

// Check for the first error encountered during iteration.
if err := ad.Err(); err != nil {
	return err
}

fmt.Printf("num: %d, str: %q", num, str)
// num: 1, str: "hello world"
```

A variety of methods are available for extracting data during iteration, such as
`Uint8/16/32/64`, `Bytes`, `String`, and the most powerful method of all: `Do`.

`Do` is a special purpose method that allows the decoder to deal with arbitrary
data, such as C structures, nested Netlink attributes, and Netlink arrays.  It
accepts a closure, and passes the current data pointed at by the decoder to the
closure.

To deal with nested Netlink attributes, create another `AttributeEncoder` within
a `Do` closure:

```go
ad.Do(func(b []byte) error) {
	nad, err := netlink.NewAttributeDecoder(b)
	if err != nil {
		return err
	}

	if err := handleNested(nad); err != nil {
		return err
	}

	// Make sure to propagate internal errors to the top-level decoder!
	return nad.Err()
})
```

To keep closure bodies small, helper functions can be defined for parsing
arbitrary types from Netlink attributes:

```go
// parseFoo returns a function compatible with Do.
func parseFoo(f *Foo) func(b []byte) error {
    return func(b []byte) error {
		// Some parsing logic...
		foo, err := unpackFoo(b)
		if err != nil {
			return err
		}

		// Store foo in f by dereferencing the pointer.
		*f = foo
		return nil
	}
}
```

Now, this helper function can be used directly with `Do`:

```go
var f Foo
ad.Do(parseFoo(&f))
```

This API provides a great amount of flexibility to its callers. All error
propagation is handled internally and bubbled up to the caller via a call to
the `Err` method from the top level decoder.

## Conclusion

Although it took some time and experimentation, I'm quite pleased with the
`netlink.AttributeDecoder`'s iterator-like byte parsing API. It's been a great
fit for my needs, and thanks to Terin Stock, we've also added a
[symmetrical encoder API](https://github.com/mdlayher/netlink/pull/95), inspired
by the success of the decoder API!

If you're working on a package API that you aren't totally happy with, the
standard library is a great place to look for inspiration!  I also highly
recommend getting in touch with the various
[Go help communities](https://golang.org/help/#help), as there are many folks
out there who are more than willing to provide excellent advice and critique!

If you have any questions, feel free to contact me! I'm mdlayher on
[Gophers Slack](https://gophers.slack.com/), [GitHub](https://github.com/mdlayher)
and [Twitter](https://twitter.com/mdlayher).

## Links

* [Package netlink](https://github.com/mdlayher/netlink) 
* [Linux, Netlink, and Go blog series](https://medium.com/@mdlayher/linux-netlink-and-go-part-1-netlink-4781aaeeaca8)
* [Go blog: Errors are values](https://blog.golang.org/errors-are-values)
* [`bufio.Scanner`](https://golang.org/pkg/bufio/#Scanner)
* [`netlink.AttributeDecoder`](https://godoc.org/github.com/mdlayher/netlink#AttributeDecoder)
