+++
author = ["Phil Pearl"]
date = "2018-12-25T08:00:00+00:00"
title = "Avoiding high GC overhead with large heaps"
series = ["Advent 2018"]
+++

I've [blogged before](https://syslog.ravelin.com/go-and-a-not-tiny-amount-of-memory-a7a9430d4d22) about running into Garbage Collector (GC) problems caused by large heaps. Quite a [few times](https://syslog.ravelin.com/gc-is-bad-and-you-should-feel-bad-e9bdd9324f0). In fact every time I've hit this problem I've managed to be [surprised](https://syslog.ravelin.com/whats-all-that-memory-for-e89522e1c2c6), and in my shock I've blogged about it [again](https://syslog.ravelin.com/further-dangers-of-large-heaps-in-go-7a267b57d487). 

# What's the problem?

I've stated the fundamental problem so many times now it feels a little trite. The Go GC's basic job is to work out which pieces of memory are available to be freed, and it does this by scanning through memory looking for pointers to memory allocations. To put it simply, if there are no pointers to an allocation it can be freed. This works very well, but the more memory there is to scan the more time it takes. If you have tens of gigabytes of memory allocated this can become a problem.

# Is it a big problem?

How much of a problem? Let's find out! Here's a tiny program to demonstrate. We allocate a billion (1e9) 8 byte pointers, so approximately 8 GB of memory. We then force a GC and time how long it takes. And we do that a couple of times to prove it isn't a fluke. We also call [`runtime.KeepAlive()`](https://golang.org/pkg/runtime/#KeepAlive) to ensure the GC/compiler doesn't throw away our allocation in the meantime.

```go
func main() {
	a := make([]*int, 1e9)

	for i := 0; i < 2; i++ {
		start := time.Now()
		runtime.GC()
		fmt.Printf("GC took %s\n", time.Since(start))
	}

	runtime.KeepAlive(a)
}
```

On my 2015 MBP I get the following output.

```
GC took 2.932771405s
GC took 702.994884ms
```

Yep, the GC took 0.7 seconds. And why should that be surprising? I've allocated 1 billion pointers. That's actually a little less than a nano-second per pointer to check them. Which is a pretty good speed for looking at pointers.

# So what next?

That seems like a fundamental problem. If our application needs a large in-memory lookup table, or if our application fundamentally is a large in-memory lookup table, then we've got a problem if the GC insists on periodically scanning all the memory we've allocated. We'll lose huge amounts of the available processing power to the GC. What can we do about this?

We essentially have two choices. We either hide the memory from the GC, or make it uninteresting to the GC.

## Make our memory dull

How can memory be uninteresting? Well, the GC is looking for pointers. What if the type of our allocated object doesn't contain pointers? Will the GC still scan it?

We can try that. In the below example we're allocating exactly the same amount of memory as before, but now our allocation has no pointer types in it. We allocate a slice of a billion 8-byte ints, again this is approximately 8GB of memory.

```go
func main() {
	a := make([]int, 1e9)

	for i := 0; i < 2; i++ {
		start := time.Now()
		runtime.GC()
		fmt.Printf("GC took %s\n", time.Since(start))
	}

	runtime.KeepAlive(a)
}
```

Again, I've run this on my 2015 MBP

```
GC took 353.964µs
GC took 187.131µs
```
The GC is over 1 million times faster, for exactly the same amount of memory allocated. It turns out that the Go memory manager knows what types each allocation is for, and will mark allocations that do not contain pointers so that the GC does not have to scan them. If we can arrange for our in-memory tables to have no pointers, then we're on to a winner.

## Keep our memory hidden

The other thing we can do is hide the allocations from the Go GC. If we ask the OS for memory directly, the Go GC never finds out about it, and therefore does not scan it. Doing this is a little more involved than our previous example!

Here's the equivalent of our first program where we allocate a []*int with a billion (1e9) entries. This time, we use the mmap syscall to ask for the memory directly from the OS kernel. Note this only works on unix-like operating systems, but there are similar things you can do on Windows.

```go
package main

import (
	"fmt"
	"reflect"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

func main() {

	var example *int
	slice := makeSlice(1e9, unsafe.Sizeof(example))
	a := *(*[]*int)(unsafe.Pointer(&slice))

	for i := 0; i < 2; i++ {
		start := time.Now()
		runtime.GC()
		fmt.Printf("GC took %s\n", time.Since(start))
	}

	runtime.KeepAlive(a)
}

func makeSlice(len int, eltsize uintptr) reflect.SliceHeader {
	fd := -1
	data, _, errno := syscall.Syscall6(
		syscall.SYS_MMAP,
		0, // address
		uintptr(len)*eltsize,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_ANON|syscall.MAP_PRIVATE,
		uintptr(fd), // No file descriptor
		0,           // offset
	)
	if errno != 0 {
		panic(errno)
	}

	return reflect.SliceHeader{
		Data: data,
		Len:  len,
		Cap:  len,
	}
}

```

Here's the output.

```
GC took 470.809µs
GC took 179.03µs
```

Now, the memory here is invisible to the GC. This has the interesting consequence that pointers stored in this memory won't stop any 'normal' allocations they point to from being collected by the GC. And this has bad consequences, which are tragically easy to demonstrate.

Here we try to store the numbers 0, 1 & 2 in heap-allocated ints, and store pointers to them in our off-heap mmap-allocated slice. We force a GC after allocating and storing a pointer to each int.

```go
func main() {

	var example *int
	slice := makeSlice(3, unsafe.Sizeof(example))
	a := *(*[]*int)(unsafe.Pointer(&slice))

	for j := range a {
		a[j] = getMeAnInt(j)

		fmt.Printf("a[%d] is %X\n", j, a[j])
		fmt.Printf("*a[%d] is %d\n", j, *a[j])

		runtime.GC()
	}

	fmt.Println()
	for j := range a {
		fmt.Printf("*a[%d] is %d\n", j, *a[j])
	}
}

func getMeAnInt(i int) *int {
	b := i
	return &b
}
```

And here's our output. The memory backing our ints is freed up and potentially re-used after each GC. So our data is not as we expected and we're lucky not to crash.
```
a[0] is C000016090
*a[0] is 0
a[1] is C00008C030
*a[1] is 1
a[2] is C00008C030
*a[2] is 2

*a[0] is 0
*a[1] is 811295018
*a[2] is 811295018
```

Not good. If we alter this to use a normally allocated []*int as follows we get the expected result.

```go
func main() {

	a := make([]*int, 3)

	for j := range a {
		a[j] = getMeAnInt(j)

		fmt.Printf("a[%d] is %X\n", j, a[j])
		fmt.Printf("*a[%d] is %d\n", j, *a[j])

		runtime.GC()
	}

	fmt.Println()
	for j := range a {
		fmt.Printf("*a[%d] is %d\n", j, *a[j])
	}
}
```

```
a[0] is C00009A000
*a[0] is 0
a[1] is C00009A040
*a[1] is 1
a[2] is C00009A050
*a[2] is 2

*a[0] is 0
*a[1] is 1
*a[2] is 2
```
## The nub of the problem

So, it turns out that pointers are the enemy, both when we have large amounts of memory allocated on-heap, and when we try to work around this by moving the data to our own off-heap allocations. If we can avoid any pointers within the types we're allocating they won't cause GC overhead, so we won't need to use any off-heap tricks. If we do use off-heap allocations, then we need to avoid storing pointers to heap allocations unless these are also referenced by memory that is visible to the GC.

# How can we avoid pointers?

TODO: arrgrgh
In large heaps, pointers are evil and must be avoided. But you need to be able to spot them to avoid them, and they aren't always obvious. Strings, slices and time.Time all contain pointers. If you store a lot of these in memory it may be necessary to take some steps.

When I've had issues with large heaps the major causes have been the following.

- lots of strings
- Timestamps on objects using time.Time
- Maps with slice values
- Maps with string keys

## strings
What is a string? Well, there are two parts to it. There's the string header, which tells you how long it is, and where the underlying data is. And then there's the underlying data, which is just a sequence of bytes.

When you pass a string variable to a function it is the string header that gets written to the stack, and if you keep a slice of strings, it is the string headers that appear in the slice.

The string header is described by [`reflect.StringHeader`](https://golang.org/pkg/reflect/#StringHeader), which looks like the following.

```go
type StringHeader struct {
	Data uintptr
	Len  int
}
```

So strings fundamentally contain pointers, so we want to avoid storing strings!

- If your string takes only a few fixed values then consider using integer constants instead
- If you are storing dates and times as strings, then perhaps parse them and see the section below on time.Time
- If you fundamentally need to keep hold of a lot of strings then read on...

Let's say we're storing a hundred million strings. For simplicity, lets assume this is a single huge global `var mystrings = []string`.

What do we have here? Underlying `mystrings` is a `reflect.SliceHeader`, which looks similar to the `reflect.StringHeader` we've just seen.

```go
type SliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}
```

So `mystrings` is fundamentally a `SliceHeader`. Len and Cap will each be 100,000,000, and Data will point to a contiguous piece of memory large enough to contain 100,000,000 `StringHeader`s. That piece of memory contains pointers and hence will be scanned by the GC.

The strings themselves comprise two pieces. The `StringHeader`s that are contained in this slice, and the Data for each string, which are separate allocations, none of which can contain pointers.


TODO: diagram showing mystrings, the data it points to, the slice headers in this data, and the string data they point to.

So the only real problem for the GC here is the big piece of memory containing all the string headers. This has pointers, so needs to be scanned in every GC cycle.

What can we do about this? Well, if all the string bytes were in a single piece of memory, we could track the strings by offsets to the start and end of each string in this memory. By tracking offsets we no-longer have pointers, and the GC is no longer troubled.

Here's a small program to demonstrate this. We'll create 100,000,000 strings, copy the bytes from the strings into a single big byte slice, and store the offsets. We'll then show the GC time is still small, and demonstrate that we can retrieve the strings by showing the first 10.

```go
package main

import (
	"fmt"
	"runtime"
	"strconv"
	"time"
	"unsafe"
)

func main() {
	var stringBytes []byte
	var stringOffsets []int

	for i := 0; i < 1e8; i++ {
		val := strconv.Itoa(i)

		stringBytes = append(stringBytes, val...)
		stringOffsets = append(stringOffsets, len(stringBytes))
	}

	runtime.GC()
	start := time.Now()
	runtime.GC()
	fmt.Printf("GC took %s\n", time.Since(start))

	sStart := 0
	for i := 0; i < 10; i++ {
		sEnd := stringOffsets[i]
		bytes := stringBytes[sStart:sEnd]
		stringVal := *(*string)(unsafe.Pointer(&bytes))
		fmt.Println(stringVal)

		sStart = sEnd
	}
}
```

```
GC took 187.082µs
0
1
2
3
4
5
6
7
8
9
```

The fundamental principle here is that if you never need to free a string, you can convert it to an index into a larger block of data and avoid having large numbers of pointers. I've built a slightly more sophisticated thing that follows this principle [here](https://github.com/philpearl/stringbank) if you are interested.

## time.Time
I like using time.Time. It makes operating on times reasonably straight-forward. But it has a dirty secret. It contains a pointer!

```go
type Time struct {
	wall uint64
	ext  int64

	// loc specifies the Location that should be used to
	// determine the minute, hour, month, day, and year
	// that correspond to this Time.
	// The nil location means UTC.
	// All UTC times are represented with loc==nil, never loc==&utcLoc.
	loc *Location
}
```
Yes, it contains a pointer to a time.Location. Even if you only ever want to use UTC times, the possibility of the pointer is there in the struct and the GC will fundamentally find it _interesting_.

So, in the world of very large heaps, we can't use time.Time to store time. What can we use instead? One solution is to use the time in nanoseconds since 1 Jan 1970 UTC, as there are functions in the time package to convert this to and from time.Time. There's one thing to beware of though: the zero value of time.Time is not 0 nanoseconds on this scale. In fact `time.Time{}.UnixNano()` is undefined!


## map[int][]somethingelse

Go Maps don't fundamentally contain pointers, but if the key or value type contains a pointer then the map does and you're heading to large-heap heck. We've seen already that slices contain pointers, so if your map values are slices, you'll struggle once the map is very large.

I came across this one neat trick for handling maps from integers to slices. The trick is very similar to the tricks with strings above. We arrange all the slice entries into one big slice, then keep start and end offsets as values in the map. We can then reconstruct our original slice by sub-slicing the big slice using the offsets.

Here's a toy and rather foolish example to illustrate this.

```go
type offsets struct {
	start int
	end   int
}

var numbersLessThan = make(map[int]offsets)
var numberSlice []int

func initNumbers() {
	for i := 1; i < 100; i++ {
		var off offsets
		off.start = len(numberSlice)
		for j := i - 1; j >= 0; j-- {
			numberSlice = append(numberSlice, j)
		}
		off.end = len(numberSlice)
		numbersLessThan[i] = off
	}
}

func printNumbersLessThan(x int) {
	offsets := numbersLessThan[x]
	fmt.Println(numberSlice[offsets.start:offsets.end])
}

func main() {
	initNumbers()

	printNumbersLessThan(5)
	printNumbersLessThan(7)
}

```
```
[4 3 2 1 0]
[6 5 4 3 2 1 0]
```

## map[string]something

TODO: this section needs more work

Why is a map[string]something a different problem to the basic problem of storing strings? Well, our solution to dealing with strings is to convert them into offsets into a large byte slice. If we make the keys to our map these offsets, then the map will only work correctly if we can find the same set of offsets each time we try the same string.

So if I turn up with the string "cat" I might initially get offset 7. I can put that in the map together with the length. If I come back the next day with "cat", then how do I remember that "cat" is already in the byte slice at offset 7? I don't have a mechanism for this. Instead I'd end up with offset 2783, and the map would not behave as I would like.

There are several ways to solve this. A man wiser than me suggested converting the strings to 64-bit or 128-bit integer hashes using a strong hash function, then using the hashes in a map[int]something. This is probably a very good plan. I, on the other hand, took this as a challenge to write my very first hash-table in my 26-year professional programming career.

I got so excited about this I ended up making two. https://github.com/philpearl/intern & https://github.com/philpearl/symboltab. They're both very specialised. Both map strings to integers, and allow you to retrieve the original string using the integer. Both also store the strings in a way similar to the big byte slice method I described, but use the hash table to avoid adding duplicates. symboltab has the additional constraint that the first integer returned is 1, the next is 2, etc., so the integers can then be used as indexes into a slice.

(Why start at 1 and not 0? No idea. Seems a foolish decision at this point.)

The combination of symboltab and a slice allows you to build a map from strings to anything with very low GC overhead.

