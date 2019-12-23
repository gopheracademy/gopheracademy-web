+++
author = ["Seebs"]
date = "2019-12-09T00:00:00+00:00"
title = "Benchmark Surprises"
subtitle = "Any Sufficiently Advanced Benchmark Is Indistinguishable From Magic"
series = ["Advent 2019"]
+++

I love a good mystery. If you really want my undivided attention, present me
with a program which does a thing, and also a proof that it can't possibly
do that thing. For instance, if you can make a change to a program that
can't affect its performance, and change its performance dramatically? I'm
hooked.

So let's look at a few real-world examples of performance mysteries which
showed up in benchmarks, and explanations of them. This might, if you're
a *boring* person, also be used as advice on avoiding these kinds of weird
and magical behaviors. I guess. If that's what you're into.

All of these examples showed up in `#performance` in Gopher slack. I'm
abbreviating and summarizing a bit for space, feel free to go skim the
archives for more detail.

Also, for those who like spoilers: The answer is nearly always "the
microbenchmark is not an accurate reflection of real workloads", the
interesting part is in trying to figure out *how* it's not like real
workloads.

## Buffering for Performance

Someone came in with a fascincatingly weird behavior. They had some code
which had a data structure which preallocated some fairly large buffers
for performance reasons, but they didn't think the buffers were really
helping. So, they altered the code to not use the buffers, and that was fine.
Then they removed the buffers, and suddenly got a 25% performance penalty.

Now, if ceasing to use the buffers cost you 25% of your performance, that
could make sense; obviously in that case the buffers are useful. But in this
case, the buffers weren't being used at all. You could rename the fields,
not change any code, and everything still compiles and runs.

There's one option here for bisecting: Leave the buffers in, but don't
allocate them. Which is to say, separate "the slice field exists in the
struct" from "the slice has been created with `make(...)`". And that reveals
that it's *making* the slices that helps.

So, what's happening? Go's garbage collector tries pretty aggressively to keep
the total allocated memory within about a factor of two of the size of the
active heap. What this means is that, on a microbenchmark where there's very
little going on except the specific data structures being tested, the allocation
of those large buffers meant the garbage collector was much less aggressive, and
ran significantly less often.

In other words: The microbenchmark is not an accurate reflection of real
workloads. In a real workload, there's going to be a lot more other things
on the heap, and thus the creation of a couple of buffers isn't going to
materially affect the garbage collector's behavior.

## Manual Loop Faster Than `range` Operator

I used to do a lot of C with primitive compilers, and I still tend to
think "oh hey I bet I can do this faster" and be completely wrong, but one
day, I thought "I wonder whether manually looping through this slice would
be faster or slower than using the range operator", so I made a test case
and discovered that it was faster. Then I adjusted the test case a bit and
discovered that it was slower. This went back and forth a few times.

The breakthrough happened when, being in a hurry, I made a stupid mistake,
and ended up comparing the wrong two implementations: Specifically, instead
of comparing two different implementations, I compared two copies of the
same implementation. And discovered that one of them was well over 30%
faster than the other.

Here's a link to a version of this program on the Go Playground:

https://play.golang.org/p/T_FJ81SzAUZ

The relevant code:

```go
func BenchmarkRange(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RangeOp(a1, a2)
		RangeOp(a2, a1)
	}
}

func BenchmarkHand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HandRange(a1, a2)
		HandRange(a2, a1)
	}
}
```

The `RangeOp` and `HandRange` functions are different, but those differences
turn out not to matter. Instead, the *order* of these functions matters; if
I move the second one above the first in the source file, the performance
changes. Sometimes. It depends on the exact day, the exact version of the
compiler, and so on.

What's actually happening? Well, honestly, *I don't know precisely*. Running
these under `perf` shows that they get different behavior for cache misses
and loads, and sometimes slightly different behavior for branch prediction.

What appears to be at issue is, in effect, the *addresses* of these functions.
A friend who has done CPU microarchitecture work at some length suggests that
it's possible this is hitting CPU cache associativity limits -- if the code
for one of these functions ends up in the same cache bucket as the code for
the function it's calling, they can evict each other from the cache sometimes,
which could result in significantly lower performance. It could also be
alignment of some type, or the exact ranges of jump instructions.

What's important is to understand that you can get a 30% variance in
performance for a hot path due to essentially unforseeable code layout changes.
Oops.

Worse yet, in this case, that *could* affect a real workload, but there's
nothing you could really do about it. Any change elsewhere in the source
file could affect this -- adding a `fmt.Printf` call in a function that
isn't even called could, after all, change the generated code for the whole
file.

But it's still not a reflection of real workloads, in that the observed
performance difference isn't actually even *related* to the code being executed.

## The Computer Can Press Its Own Turbo Button

Long ago, some software was dependent on CPU speed enough that early machines
which could run at speeds other than the standard 4.77MHz often had a button
to toggle whether the computer ran at some reasonably native clock speed, or
at the much slower rate that the software had been written for.

Modern software is much less likely to fail completely when clockspeed changes,
and we have clocks other than cycle counters, but when you're benchmarking,
it does matter what speed the CPU runs at.

Unfortunately, what speed the CPU runs at varies. A lot. On my laptop, for
instance, the nominal clock speed is about 3GHz. If not much is happening,
it is probably running around 1100MHz. But even talking about the speed of
"the" CPU is misleading; despite this being a single physical chip, it has
four cores, and 8 virtual CPUs thanks to hyperthreading. So let's see what
I'm running at under light load:

```
$ grep MHz /proc/cpuinfo
cpu MHz         : 2328.913
cpu MHz         : 2083.069
cpu MHz         : 2042.439
cpu MHz         : 2951.134
cpu MHz         : 2948.966
cpu MHz         : 1883.699
cpu MHz         : 1998.988
cpu MHz         : 2378.981
```

Note that there's exactly one CPU reporting <1900MHz, despite the CPU having
two virtual CPUs per core. Why? Who knows. But under these circumstances, the
same code getting switched from one core to another could move it from a
1.9GHz CPU to a 2.9GHz CPU. That kind of difference could be fairly significant.

One partial solution, for x86 systems, is [perflock](https://github.com/aclements/perflock),
which does two things. First, it is a sort of performance-testing mutex; it
won't let two things that are using it run at once, so they won't interfere
with each other. (This won't help as much if you're also running CPU-intensive
things, such as a text editor, or a text chat client, that can consume
gigabytes of memory and entire CPUs.) Secondly, it will interact with the
system's power management and CPU performance stuff to try to lock a CPU at
a fixed rate. Note that the default value may be higher than a system can
sustain -- you may want to run with `-governor 70%` or something similar to
pick something that won't cause thermal throttling, or else you still get
unexpected performance variance.

In this case, the highly erratic behavior of CPU performance *does* reflect
real workloads; modern systems have highly erratic behavior sometimes!
Unfortunately, that's useless when comparing the performance of different
versions of code to see which is faster. It's useful to do that testing in
an environment which *intentionally* doesn't represent real-world workloads,
when you want reproducible data.

## Computer Too Bored, Not Paying Attention

Someone was doing some latency tests on a service, using
[vegeta](https://github.com/tsenart/vegeta), and was getting really strange
results. Intuitively, my expectation would be that in general, higher loads lead
to more latency; there's more probability that you are waiting for something
else to complete, or contending for a lock. What they actually observed was
dramatically *lower* latency at higher request rates.

[Here's the Github issue where this showed up.](https://github.com/tsenart/vegeta/issues/278)

In this case, the issue appears to be that, on modern systems, CPUs tend to
very quickly go into idle states and lower their clock speed when there's not
enough workload to saturate them. They don't immediately speed up fully again,
so if you're on a processor that's nominally 3GHz, and it's currently running
at 800MHz, it's going to be running at about 25% of nominal speed, and that
means things will take 4x longer.

In experimentation, I found that having even one low-priority task on a
machine that woke up every nanosecond or so kept the machine from doing
this. I would summarize the problem as "the computer is bored and is not really
paying attention".

The problem here is that usually when we're talking about performance, we're
thinking about amount of work done over time, or how long it takes per work
item, and for that purpose, it doesn't matter how fast things are when there
aren't nearly enough to saturate the CPU. However, latency of individual
requests can also matter, and having latency suddenly get noticably worse
when things are quiet is going to be a problem for some workloads. It isn't
totally solved by running a small amount of busywork to keep the CPU up,
or using perflock, but that seems to address most of the gap.



Special thanks to:

- The entire `#performance` channel.

## Links

* [perflock](https://github.com/aclements/perflock)
