+++
title= "Directional Channels in Go"
date = "2019-12-03T00:00:00+00:00"
series = ["Advent 2019"]
author = ["Andy Walker"]
linktitle = "flag"
+++

Go's [channels](https://tour.golang.org/concurrency/2) provide a primitive for typed, synchronous message passing. Combined with [goroutines](https://tour.golang.org/concurrency/1), they form the backbone of Go's [CSP](https://en.wikipedia.org/wiki/Communicating_sequential_processes)-inspired concurrency model. They're simple and expressive, but they can be difficult to use properly, especially if you need to control who can read from them or write to them.

## The Problem With Bidirectional Channels

Channels are normally declared with the `chan` keyword, followed by the **ElementType**, which represents the type of values passed on that channel. Together, these form the composite type for any value, which you can inspect with `%T`.

```go
var stringChan chan string
fmt.Printf("%T\n", stringChan) // "chan string"
```
[playground](https://play.golang.org/p/F58BWz2HJEZ)

This is the declaration format most people first encounter when working with channels. But any channel created in this way will be bidirectional as the default behavior. This means that anyone who has access to a channel value can read from it *and* write to it. This can cause problems in a concurrent environment, and many a Go programmer has torn hair from their heads trying to debug a `panic: send on a closed channel`.

The common wisdom is that only the sender should close a channel, and this makes sense. Only the sender can know when there's no more data to send, and it's the receiver's responsibility to watch for the close, or ideally, to simply `range` over the channel, exiting the loop naturally when it's done. If this order is upset, it's generally a sign something very wrong is going on, hence the panic. But if anyone can perform any action on a channel, including calling `close()`, how can you reel this in?

## Directional Channels

If you look at the [language spec](https://golang.org/ref/spec#Channel_types) for channels, it turns out that channel direction can actually be _constrained_!

> The optional <- operator specifies the channel *direction*, *send* or *receive*.

This means channels can actually be declared in one of three ways, depending on whether we want them to be bidirectional, receive-only or send-only:

```go
var bidirectionalChan chan string // can read from, write to and close()
var receiveOnlyChan <-chan string // can read from, but cannot write to or close()
var sendOnlyChan chan<- string    // cannot read from, but can write to and close()
```

A good way to remember how this works is that, in declarations, the arrow indicates how the channel is allowd to be used:
```
<-chan // data only comes out
chan<- // data only goes in
```

At first glance, this might seem pretty useless --how useful is a new channel if it can't work in both directions?-- but there's another important line in the spec in the very same paragraph:

> A channel may be constrained only to send or only to receive by **assignment** or **explicit conversion**.

This means channels can start out bidirectional, but magically _become_ directional simply by assigning a regular channel to a variable of a constrained type. This is very useful for creating receive-only channels that no one can close but you.

## Receive-only Channels

```go
var biDirectional chan string
var readOnly <-chan string

biDirectional = make(chan string)

takesReadonly(biDirectional)
readOnly = biDirectional
```

`readOnly` now shares the same underlying channel, as `biDirectional`, but it cannot be written to *or* closed. This can also be done on the way into our out of a function, simply by specifying a direction in the argument or return type:

```go
func takesReadonly(c <-chan string){
    // c is now receive-only inside the function and anywhere else it might go from here
}

func returnsReadOnly() <-chan string{
    c := make(chan string)
    go func(){
        // some concurrent work with c
    }()
    return c
}
readOnly := returnsReadOnly()

```

This is a pretty nifty trick, and works a bit differently to conversions in the rest of the language, but, most crucially, the change in direction is reflected in the *type*, which means these restrictions can be enforced at *compile time*.

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

This is useful not only to control who can write to or close your channel, but also in terms of descriptiveness and Intentionality. One of the nice things about strongly-typed languages like Go is that they can be tremendously descriptive just through their API. Take the following function as an example:

```go
func SliceIterChan(s []int) <-chan int {}
```

Even without the documentation or implementation, this code unambiguously states that it returns a channel that the consumer is supposed to read from, either forever, or until it's closed (which documentation can help clarify). This lends itself very well to a **for-range** over the provided channel.

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

This is a very powerful technique for asserting control over a channel at an API boundary, and one that comes with no cost or need for explicit conversion, beyond simply specifying the channel direction in a declaration. This is so useful, you should use probably use it wherever you return a channel for reading from, unless there's a very good reason not to.

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

Unlike the example above, neither timers nor tickers are ever closed to prevent erroneous firings, so dedicated `Stop()` methods are provided on both of these types, along with instructions on how to handle this situation correctly. This is another best practice around receive-only channels, and you should work to ensure that you provide similar mechanisms and instructions if there's any chance the consumer might want to stop reading from your channel early. Check out [Principles of designing Go APIs with channels](https://inconshreveable.com/07-08-2014/principles-of-designing-go-apis-with-channels/) by Alan Shreve for more on this topic.

## Send-only Channels

You can also declare channels as send-only, but these are of much more limited use. While they can provide useful assertions internally that a channel is never read from, receiving them with an API is kind of backwards, and you are generally better off using a bidirectional channel internally, and moderating channel writes with a function or method.

Send-only channels make only one appearance in the standard library in `os/signal`:

```
func Notify(c chan<- os.Signal, sig ...os.Signal)
    Notify causes package signal to relay incoming signals to c. If no signals
    are provided, all incoming signals will be relayed to c. Otherwise, just the
    provided signals will.

    Package signal will not block sending to c: the caller must ensure that c
    has sufficient buffer space to keep up with the expected signal rate. For a
    channel used for notification of just one signal value, a buffer of size 1
    is sufficient.

    It is allowed to call Notify multiple times with the same channel: each call
    expands the set of signals sent to that channel. The only way to remove
    signals from the set is to call Stop.

    It is allowed to call Notify multiple times with different channels and the
    same signals: each channel receives copies of incoming signals
    independently.
```

Here, the user is expected to pre-allocate an `os.Signal` channel for receiving incoming signals from the OS. The API asserts that the channel will only ever be written to, and informs the user that they need to create a buffered channel of whatever size they deem necessary to avoid blocking. It might seem necessary to take a send-only channel to allow the user to set their own channel depth, but the signature could just as easily have been something like:

```
func Notify(depth uint, sig ...os.Signal) <-chan os.Signal
```

Returning a receive-only channel, similarly to how package `time` operates. The only difference is by taking a channel as an argument, package `os/signal` can keep track of the user's notify channels, allowing for the multiple calls it mentions to expand the set of signals the channel will receive.

This is a very specific use case, however, and one that involves a global state, so you're better off finding another way to support something like this, if that's your goal.

## Conclusion

I hope this gives advent readers a better understanding of Go's channel direction behavior and using channels to their full capabilities.

## About the Author
Andy Walker is a Go GDE and co-organizer of [Baltimore Go](https://www.meetup.com/BaltimoreGolang/), as well as the primary orgnizer of the GopherCon Guide Program. He is a programmer in security research for a major cybersecurity company. He enjoys hardware, 3D printing, and talking way too much about philosophy. He can be reached at andy-at-[andy.dev](https://andy.dev). Twitter: [@flowchartsman](https://twitter.com/flowchartsman)
