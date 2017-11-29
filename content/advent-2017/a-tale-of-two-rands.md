+++
author = ["Nelz Carpentier"]
title = "A Tale of Two `rand`s"
linktitle = "two rands"
date = 2017-11-10T06:40:42Z
series = ["Advent 2017"]
draft = true
+++

I had always been a bit confused as to how the `crypto/rand` package and the `math/rand` package were related, or how they were expected to work (together). Is this something that everyone else already grokked, or is that just my impostor syndrome talking? Well, one day I decided to see if I could defeat my ignorance, and this blog post is the result of that investigation.

# The `math` One
If you've every poked around in the [`math/rand`](https://golang.org/pkg/math/rand) package, you might agree that it presents a fairly convenient API. My favorite example is the `func Intn(n int) int`, a function that returns a random number within the range that you've given it. SUPER USEFUL!

You may be asking about the difference between the top-level functions and the functions hung off an instance of the `Rand` type. If you look at the [source code](https://golang.org/src/math/rand/rand.go), you'll see that the top-level functions are just convenience wrappers that refer to a globally instantiated package value called `globalRand`.

There are a few gotchas when using this package, though. The basic usage of only provides __pseudo-random__ numbers, as a function of the seed. This means that if you create two `Rand` instances using a functionally equivalent seed, equivalent calls (in order and function) to the two instances will produce _parallel_ outputs. (I found this concept to be personally challenging to my understanding of "random", because I wouldn't expect to be able to anticipate a "random" result.) If the two `Rand` instances are seeded with different values, the parallel behavior will not be observed.

# The `crypto` One
Now, let's look at [`crypto/rand`](https://golang.org/pkg/crypto/rand/). Okay, it's got a nice and concise API surface. The only thing is: HOW THE HECK DO I USE THIS?!? I see that I can generally get byte slices of random 1's and 0's, but what do I do with those?!? That's not nearly as useful as what `math/rand` provides, right?

Hrm. Maybe the question is: How can I combine these two wildly different packages?

# Two Great Tastes That Taste Great Together
(NB: https://www.youtube.com/watch?v=DJLDF6qZUX0)

Let's take a deeper look at the `math/rand` package. We instantiate a `rand.Rand` by providing a `rand.Source`. But a `Source` is, like almost all awesome things in Go, an interface! My spidey-sense is tingling, maybe there's an opportunity here?

The main workhorse in `rand.Source` is the `Int63() int64` function, which returns a non-negative `int64` (i.e. the most significant bit is always a zero). The further refinement in `rand.Source64` just returns a `uint64` without any limitations on the most significant bit.

Whaddya say we try to create a `rand.Source64`, using our tools from `crypto/rand`? (You can follow along with this code on the [Go Playground](https://play.golang.org/p/_3w6vWTwwE).)

First, let's create a struct for our `rand.Source64`. (Also to note: since `math/rand` and `crypto/rand` would collide in usage, we'll use `mrand` and `crand`, respectively, to distinguish between them in the following code.)
```go
type mySrc struct{}
```

Let's address the `Seed(...)` function from the interface. We do not need a seed for interacting with the `crypto/rand`, so this is just a no-op.
```go
func (s *mySrc) Seed(seed int64) { /*no-op*/ }
```

Since the `Uint64()` function returns the "widest" value, requiring 64 bits of randomness, we'll implement that function first. We use the tools from `encoding/binary` to read 8 bytes off the `io.Reader` provided by `crypto/rand`, and turn that directly in to a `uint64`.
```go
func (s *mySrc) Uint64() (value uint64) {
	binary.Read(crand.Reader, binary.BigEndian, &value)
	return value
}
```

The `Int63()` function is similar to the `Uint64()` function, but we just need to make sure the most significant bit is a always a 0. That's pretty easy to do with a quick bitmask applied to a value produced by `Uint64()`.
```go
func (s *mySrc) Int63() int64 {
	return int64(s.Uint64() & ^uint64(1<<63))
}
```

Great! Now we have a fully operable `rand.Source64`. Let's verify it does what we need by putting it through its paces.
```go
var src mrand.Source64
src = &mySrc{}
r := mrand.New(src)
fmt.Printf("%d\n", r.Intn(23))
```

# Tradeoffs
Cool, so with the above code, at about a dozen lines, we have an easy-peasy way of hooking up cryptographically secure random data generation to the nice and convenient API provided by the `math/rand` package. However, I've come to learn that nothing comes for free. What might we be giving up by using this? Let's check what happens when we [benchmark](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go) this code.

(NB: I like using prime numbers in my tests, so you'll see lots of 7919, the 1000th prime, as a parameter.)

What kind of performance do we get out of the top-level functions out of the `math/rand` package?
```go
func BenchmarkGlobal(b *testing.B) {
	for n := 0; n < b.N; n++ {
		result = rand.Intn(7919)
	}
}
```

Not bad! About 38ns/op on my laptop.
```
BenchmarkGlobal-4         	50000000	        37.7 ns/op
```

What if we create a new instance of the `rand.Rand` type, seeded with the current time?
```go
func BenchmarkNative(b *testing.B) {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	for n := 0; n < b.N; n++ {
		result = random.Intn(7919)
	}
}
```

At ~23ns/op, that's really good, too!
```
BenchmarkNative-4         	100000000	        22.7 ns/op
```

Now, let's check the new seed that we wrote.
```go
func BenchmarkCrypto(b *testing.B) {
	random := rand.New(&mySrc{})
	for n := 0; n < b.N; n++ {
		result = random.Intn(7919)
	}
}
```

Oof, at ~900ns/op this clocks in at an at least an order of magnitude more expensive. Is it something we did incorrectly in the code? Or is this maybe the "cost of doing business" with `crypto/rand`?
```
BenchmarkCrypto-4     	 2000000	       867 ns/op
```

Let's build a test to see how long just reading from `crypto/rand` takes in isolation.
```go
func BenchmarkCryptoRead(b *testing.B) {
	buffer := make([]byte, 8)
	for n := 0; n < b.N; n++ {
		result, _ = crand.Read(buffer)
	}
}
```

Okay, the results show that the vast majority of time spent in our new tool comes from the underlying cost of interacting with the `crypto/rand` package.
```
BenchmarkCryptoRead-4     	 2000000	       735 ns/op
```

I don't know that there's a lot we can do to mitigate this. Besides, maybe a routine that runs in ~1 millisecond to get non-deterministic random numbers is not a problem for your use case. That's something you'll need to evaluate for yourself.

# Another Tack?
One of the usages of randomization that I am most familiar with is in [exponential backoff](https://en.wikipedia.org/wiki/Exponential_backoff) tools. The idea is to reduce the chances of accidental synchronization when reconnecting to a stressed server, because pulsed loads might be detrimental to that server's recovery. "Deterministic random" behavior itself is not a problem in these scenarios, but using the same seed across a bunch of instances can be problematic.

And this is a problem when defaulting to the top-level `math/rand` functions (which are implicitly seeded with `1`), or by using the frequently observed pattern of seeding with `time.Now().UnixNano()`. If your services happen to come up at the same time, you just might end up in accidental synchronization with respect to the deterministic random output.

How about we use our powers of `crypto/rand` at instantiation time to seed the `math/rand` tools, after which we can still enjoy the performance benefits of using the deterministic random tools?

```go
func NewCryptoSeededSource() mrand.Source {
	var seed int64
	binary.Read(crand.Reader, binary.BigEndian, &seed)
	return mrand.NewSource(seed)
}
```

We can run benchmarks on this new code, but we already know we'll just be falling back to deterministic random performance characteristics.
```go
func BenchmarkSeed(b *testing.B) {
	random := mrand.New(NewCryptoSeededSource())
	for n := 0; n < b.N; n++ {
		result = random.Intn(7919)
	}
}
```

And now we've proven our assumptions were right.
```
BenchmarkSeed-4           	50000000	        23.9 ns/op
```

# About the Author
Hi, I'm Nelz Carpentier. I'm a Senior Software Engineer at [Orion Labs](https://www.orionlabs.io/) in San Francisco. I've been writing Go for about 3 years now, and upon familiarization it quickly became one of my favorite languages.

Disclaimers: I am neither a security expert, nor an expert on `crypto/rand` implementations across platforms; you might want to consult with your local security expert if you use these tools in mission-critical security use cases.

You can find a distillation of these examples [here](https://github.com/orion-labs/go-crypto-source). It has an Apache 2.0 License, so feel free to slice, dice, and/or borrow whatever you need from it!
