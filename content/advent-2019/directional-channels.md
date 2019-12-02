+++
title= "Directional Channels in Go2"
date = "2019-11-03T00:00:00+00:00"
series = ["Advent 2019"]
author = ["Andy Walker"]
linktitle = "flag"
+++

Go's [channels](https://tour.golang.org/concurrency/2) provide a primitive for typed, synchronous message passing. Combined with [goroutines](https://tour.golang.org/concurrency/1), they form the backbone of Go's [CSP](https://en.wikipedia.org/wiki/Communicating_sequential_processes)-inspired concurrency model, but they can express more than just the notion of message passing.

Channels are declared with the `chan` keyword, followed by the **ElementType**, which represents the type of values passed on that channel. Together, these form the composite type for any value, which you can inspect with `%T`.

```go
var stringChan chan string
fmt.Printf("%T\n", stringChan) // "chan string"
```
[playground](https://play.golang.org/p/F58BWz2HJEZ)

This is the declaration format most people first encounter when working with channels, and any channel created in this way will be *bidirectional*, which means it can be both read from and written to. So far so good, but if you look at the [language spec](https://golang.org/ref/spec#Channel_types), you'll see that the channel direction can also be constrained:

> The optional <- operator specifies the channel *direction*, *send* or *receive*.

This means channels can actually be declared in one of three ways, depending on whether we want them to be bidirectional, receive-only or send-only:

```go
var bidirectionalChan chan string // can read from, write to and close()
var receiveOnlyChan <-chan string // can read from, but cannot write to or close()
var sendOnlyChan chan<- string    // cannot read from, but can write to and close()
```

At first glance, this seems pretty useless --why would you want to create a channel that you can't read from or write to?-- but there's another important line in the spec in the very same paragraph:

> A channel may be constrained only to send or only to receive by **assignment** or **explicit conversion**.

This means channels can magically _become_ directional simply by assigning a regular biderectional channel to a variable of a constrained type, or passing it into a function with directional channel arguments:

```go
var biDirectional chan string
var readOnly <-chan string
func takesReadonly(c <-chan string){}

biDirectional = make(chan string)

takesReadonly(biDirectional)
readOnly = biDirectional
```

`readOnly` now shares the same underlying channel, as `biDirectional`, but it cannot be written to *or* closed. Most crucially, this distinction is part of its *type*, which means these restrictions can be enforced at *compile time*:

```go
go func() {
    biDirectional <- "hello" // no problem
    close(biDirectional)     // totally fine
}()
go func() {
    readOnly <- "hello" //"invalid operation ... (send to receive-only type <-chan string)"
    close(readOnly)     //"invalid operation: ... (cannot close receive-only channel)"
}()

fmt.Printf("%T\n", readOnly) // "<-chan string" (different type)
fmt.Println(<-readOnly)      // "hello" (same underlying channel!)
```
[playground](https://play.golang.org/p/y1xe8R9wQHK)

How is this useful to you as a programmer? Descriptiveness and Intentionality. One of the nice things about strongly-typed languages like Go is that they can be tremendously descriptive just through their API. Take the following function as an example:

```go
func SliceIterChan(s []int) <-chan int {}
```

Even without the documentation or implementation, this code unambiguously states that it returns a channel that the consumer is supposed to read from, either forever, or until it's closed (which documentation can help clarify). This lends itself very well to a natural **for-range** over the provided channel.

```go
for i := range SliceIterChan(someSlice) {
    fmt.Printf("got %d from channel\n", i)
}
fmt.Println("channel closed!")
```

Diving into the implementation, the function creates a bidirectional channel for its own use, and then all it needs to do to ensure that it has full control over writing to and closing the channel is to return it, whereupon it will be converted into a read-only channel automatically.

```go
// SliceIterChan returns each element of a slice on a channel for concurrent
// consumption, closing the channel on completion
func SliceIterChan(s []int) <-chan int {
	outChan := make(chan int)
	go func() {
		for i := range s {
			outChan <- s[i]
		}
		close(outChan)
	}()
	return outChan
}
```
[playground](https://play.golang.org/p/nGMksaNgxAg)

This is a very powerful technique for asserting control over a channel at an API boundary, and one that comes with no cost or need for explicit conversion. Indeed, you might even go so far as to say that, if your API provides a channel for returning results or signals, you should *always* explicitly return a receive-only channel.

This is a similar approach to what the standard library does with tickers and timers in the `time` package:

```
type Ticker struct {
        C <-chan Time // The channel on which the ticks are delivered.
        // Has unexported fields.
}
    A Ticker holds a channel that delivers `ticks' of a clock at intervals.

func After(d Duration) <-chan Time
    After waits for the duration to elapse and then sends the current time on
    the returned channel. It is equivalent to NewTimer(d).C. The underlying
    Timer is not recovered by the garbage collector until the timer fires. If
    efficiency is a concern, use NewTimer instead and call Timer.Stop if the
    timer is no longer needed.
```

Though, unlike the example above, neither timers nor tickers are ever closed to prevent erroneous firings, and dedicated `Stop()` methods are provided on both of these types, along with instructions on how to handle this situation correctly.

This is another good practice around read-only channels, and you should work to ensure that any read-only channels your code might provide have similarly well-defined methods and procedures if there is any chance that the caller would want to stop reading from them early. For further musings on this, check out [Principles of designing Go APIs with channels](https://inconshreveable.com/07-08-2014/principles-of-designing-go-apis-with-channels/) by Alan Shreve.

Write only channels are useful as well, but mostly for internal confirmation that a channel is not read from at any point deeper in your code. Their use can be seen as an API promise that the caller will be the only reader of a particular channel that they provide. This is really only useful if you indend on non-blocking writes to the channel, allowing the caller to set up a buffered channel of the depth they deem necessary to keep up, so you're generally better off returning a read-only channel instead.

Still, for this very specific use-case, the conversion works much the same way, and this is used by the standard library in `os/signal` where [`Notify`](https://golang.org/pkg/os/signal/#Notify) takes a channel that will be used to relay signals from the OS and [`Stop`](https://golang.org/pkg/os/signal/#Stop) ceases notifications on a previously-provided channel. Note that the docs very specifically call out that notification is non-blocking, and that the caller must ensure sufficient buffer space.

## About the Author
Andy Walker is a Go GDE and co-organizer of [Baltimore Go](https://www.meetup.com/BaltimoreGolang/), as well as the primary orgnizer of the GopherCon Guide Program. He is a programmer in security research for a major cybersecurity company. He enjoys hardware, 3D printing, and talking way too much about philosophy. He can be reached at andy-at-[andy.dev](https://andy.dev).
