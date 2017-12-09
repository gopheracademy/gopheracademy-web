+++
author = ["Chewxy"]
title = "Property Based Testing"
linktitle = "Short Title (use when necessary)"
date = 2017-12-09T06:40:42Z
+++

# Useful Programs #

I'll start this post with a bold but fairly uncontroversial statement: Programs that do not interact with the outside world are useless - they do nothing except consume cycles. A corollary of this is that pure functional programming is useless. Here is a (mostly) pure functional program written in Go that does absolutely nothing:

```go
package foo

func incr(i int) int { return i+1 }
func decr(i int) int { return i-1 }
func foo()           { decr(incr(0)) }
```

While obviously there are other things that make a program useful, the most fundamental thing that makes a program useful is that it needs to interact with things outside itself. This can come in the form of reading a file, reading a network input, reading user input, printing an output, or writing to a file. Indeed, so fundamental is input/output to the notion of programming that most languages have a standardized definition for a program entry point. In Go, it's the `func main(){}` function. 

Another feature about useful programs that they have to be robust. Here, the notion of fragile programs can be induced by an example - imagine a program that breaks down at every other input that is fed to it. It's not very useful, is it?

Lastly, a program has to do what we intend for it to do in order to be useful. If I want a program to add two numbers together, a useless program would be one that plays music instead of calculating the sum of two numbers. 

To recap, a useful program has to:

* Deal with I/O
* Be robust to various inputs
* Do the correct thing

## Mo' Inputs Mo' Problems ##

We've established that inputs and outputs are what makes programs useful. For the rest of this post, I'll talk mostly about inputs. And to spare readers of the philosophical conundrum that may occur, I'll limit to talking about inputs of things that are controllable by the programmer. In practice, this means inputs to functions. 

The problem with inputs is that there can be many many different inputs. Let's say we define a function `Add`. `Add` takes two inputs, `a` and `b` and spits out a result. We'll write it like this: `Add(a, b)`. Some combinations of `a` and `b` will not make sense: adding a Duck and a Flower does not make sense - they're two different types of things! 

So now there's an intuition that in order for programs to return meaningful results, there are different types of things at play - we should write define our functions to be explicit about what things are being accepted. This is typically the first line defence - the type system. But this post is about testing, not about type systems. So why bring it up? Most modern type systems have an escape hatch, allowing programmers to subvert it, either for I/O purposes or performance purposes. 

Clearly a type system helps. But if our job is to write a robust program, we would need more help than just a type system.

# Testing 101 #

Recall the three things that a useful program requires. The last one is "Do the correct thing". A good type system would prevent a lot of wrong inputs from being entered to a program (preferably at compile time). But the actual actions that your program does could actually be wrong. Consider this function:

```go
func Add(a, b int) int { return a - b }
```

According to the type system it's correct. It takes two `int` inputs, and outputs an `int`. But it's doing the wrong thing! I wouldn't want this function to be anywhere near my accounting software! This is why we write tests - to ensure that the code does what it's meant to do. 

Go supports testing as a first class citizen of the language. To write tests, you simply write a function that looks like this:

```go
func TestAdd(t *testing.T) {
	if out := Add(1, 2); out != 3 {
		t.Errorf("Add failed. Expected 3. Got %d instead", out)
	}
}
```

Then run `go test`. It just works. 

## The Case Of Perverse Incentives ##

Testing is great. It ensures that many people can work on the same project without stepping (much) on each others' toes. But blind insistence on testing can lead to poor software. 

Quite a while back, when I was running my own startup, I'd hire freelancers to help out with my workload. I'd require tests to be written, to ensure at least minimum quality of work. At first everything went fine. Eventually bugs were found in the algorithms written by my freelancer. It had been determined that it was a whole class of edge cases that I had never anticipated. I'd send the code back, and it'd get fixed, tests included. Now the following is entirely my fault - I had reviewed the code entirely too cursorily. If Travis-CI is green, I'd approve and merge the changes in.

After a while, I started noticing that the class of edge cases I thought he'd fixed were coming back. That's when I went back to the code only to discover that the tests have passed because the algorithm was fixed to only include the specific edge case I had specified, and nothing else.

Along the way I discovered other things that made me really quite upset (mostly at myself for not reviewing the code carefully). Here's what happened. Imagine I asked you to test the `Add` function from above. What is one valid way to make sure it always passes? Here's one way:

```go
func TestAdd(t *testing.T) {
	a := 1
	b := 2
	out := Add(a, b)
	if out != a + b {
		t.Errorf("Add failed...")
	}
}
```

If you can't tell what's wrong, here's a less subtle version of it:

```go
func TestAdd(t *testing.T) {
	a := 1
	b := 2
	out := Add(a, b)
	if out != Add(a, b) {
		t.Errorf("Add failed...")
	}
}
```

If you're testing function `Foo`, you can't use `Foo` to verify that `Foo` works! That's just nuts!

# Testing On More Inputs #

"Be robust to various inputs" means that you have to test for a lot of different inputs. OK codebases will have tests for the happy path, where everything  goes well. Better code bases will have test cases where the program fails - the so-called tragic path - to test the ability of the program to handle failure conditions correctly. Being able to correctly handle failure conditions is the prerequisite to robust software.

Testing along the tragic path is not easy. For many functions and programs it's difficult to reason out where it may go wrong. In Go, this is somewhat alleviated by making errors a value. Even so, a codebase may often choose to ignore error checking:

```go
func returnsError() error { return errors.New("This Error") }

func foo() {
	returnsError() // bad! Don't do this! Always check for and handle errors!
}
```

But let's say you are a diligent programmer, and you checked all the errors, other forms of errors may occur too: panics from slice indices being out of bounds, etc. 

So an idea emerges: why don't we test on all possible values? Or at least as many values as we can? 

Enter the notion of [fuzz testing](https://en.wikipedia.org/wiki/Fuzzing): we feed the program random inputs, and then watch for it to fail. Fuzz testing often leads one to find subtle bugs you don't expect. Here's [one](https://github.com/chewxy/skiprope/issues/9) that Chris Marshall discovered on my [skiprope](https://github.com/chewxy/skiprope) library. 

Here's [another](https://github.com/gorgonia/randomkit/blob/master/utils_test.go#L40), from a random number generator library I wrote for Gorgonia, where a [Kahan summation algorithm](https://en.wikipedia.org/wiki/Kahan_summation_algorithm) yielded a different result from expected due to the way errors in floats are handled.

I have to this day, no idea what the correct fixes are. Solutions welcome.

## A More Disciplined Approach ##

Fuzz testing works by throwing everything at the wall and seeing what sticks. It's extremely useful for finding bugs you hadn't reasoned out yet. That typically falls under the purview of "be robust to various inputs" part of building useful programs. But it's not good enough to test what inputs work and what inputs cause a `panic`. You also want to test that those inputs are doing the correct things. 

"Doing the correct things" means thinking deeply about what your program is supposed to do. You gotta be honest with yourself - if you are testing an `Add` function on integers, you can't use `Add` or `+` to verify the results. That's the circular logic trap that many fall into. I find myself also occasionally falling into the same trap due to laziness.

Testing for results is tedious anyway - you have to know that the result of `1 + 2` is `3`. And that's for the simple stuff. Imagine having to know the answers ahead of time for random input for more complicated programs.

## Properties ##

Instead of doing that, we can adopt a different approach to testing. Instead of asking "what is the expected result of this program given these example inputs?", we ask the question: "what is the property of the result and program that doesn't change given inputs?"

All that sounds a bit abstract. But applying it on the `Add` function that we've become so familiar - what doesn't change in `Add`? 

Well, we know through much real life experience that `a + b` is equal to `b + a`. Mathematicians call this property [commutativity](https://en.wikipedia.org/wiki/Commutative_property). So test for it. Write a test `Add`, that takes two pairs of random inputs, perform them once, and perform them with the operands' orders switched. They should have the same result.

Of course, you could have a test that reads something like this:

```go
func TestAdd(t *testing.T) {
	// a function that checks for commutativity
	comm := func(fn func(a, b int) int) bool {...} 
	if !comm(Add) {
		t.Error("Not Commutative")
	}
}

func Add(a, b int) int { return a * b } // PEBKAC error
```

The test above will pass. But it'd be doing the wrong thing. Here you need to be asking, what makes `Add` different from `Mul`? Or more generally: What makes any function that has a signature `func(a, b int) int` different from `Add`?

There are obviously a few guidelines, which I will list below, but before that, do have a think. 

The point I'm trying to make here is that there are multiple properties to be tested for every program. You can't just test one property and be done with it. 

# How To Do Property Based Testing In Go #

So, with the Whys out of the way, let's talk about How to do property based testing in Go. Go actually comes with a built in property testing tool, `"testing/quick"`. Given that property based testing is based on Haskell's [QuickCheck](https://hackage.haskell.org/package/QuickCheck), naturally the testing function is called `quick.Check`. The gist of how it works is simple: write your property test as a function, pass it into the `quick.Check` function, and Bob's your uncle.

Here's how to test for commutativity with `"testing/quick"`:

```go
import "testing/quick"

func TestAdd(t *testing.T) {
	// property test
	comm := func(a, b int) bool {
		if Add(a, b) != Add(b, a) {
			return false
		}
		return true
	}
	if err := quick.Check(comm, nil); err != nil {
		t.Error(err)
	}
}
```

Here we see that `comm` was defined rather simply: `if Add(a, b) != Add(b, a)` then we say that the `Add` function does not fulfil the commutativity property.

What the `quick.Check` function then does is it generates values based on the input arguments of the testing function. In this case, `comm` is a function that takes two `int`. The package understands that it needs to generate two `int` to be fed into the function. 

The package works for non-primitive types too. Further extensions to functionality can be had by types that implement the `quick.Generator` interface.

Take for example, a coordinate:

```go
type Point struct {
	x, y int
}
```

As long as `Point` implements `quick.Generator`, you can put it in the input of the property testing function, like so:

```go
func (Point) Generate(r *rand.Rand, size int) reflect.Value {
	p := Point{}
	p.x = rand.Int()
	p.y = rand.Int()
	return reflect.ValueOf(p)
}
```

This is especially useful for types where there are unexported properties. If all your fields are exported, `testing/quick` can typically generate values for them.

Because Go doesn't have value-constrained types (aka ranged types in Ada, for example), it may be instructive to want to test on a subset of values, like `int`s that are even. I'd go so far to say create separate tests for different classes of ranged values. 

# How To Think About Properties #

So we've now been briefly introduced to the world of properties: there is now a notion that there are some properties that functions and programs have. Here are some properties that an addition function for numbers should have:

* **Commutativity**: `Add(a, b) == Add(b, a)`
* **Associativity**: `Add(a, Add(b, c)) == Add(Add(a, b), c)`
* **Identity**: `Add(a, IDENTITY) == a`. 

Let's talk about the Identity property. The Identity property states that calling the function with any value and the identity will always yield the same value. When you think about that, the `IDENTITY` of `Add` is `0`. Conversely the `IDENTITY` value of `Mul` is `1` because any number multiplied by 1 will always be that number itself. This is what separates `Add` and `Mul`. 

This opens a new avenue for us to think about properties. Often properties are described in scary mathematical language, like "Commutativity", or "Idempotency" or "Invariants". But look closer, and we'll find hints on how to think about creating properties to test for. 

I've actually mentioned this earlier, but it bears repeating: what doesn't change? In the first two property above, commutativity and associativity, the results don't change when the order of operations are changed. In the identity property, the value itself doesn't change.

The trick to thinking of properties to test for is to **figure out what does not change under various situations**. The jargon frequently used is "invariant". 

For example, let's say you invented a new sort algorithm. What properties that all sort algorithms have to hold in order to be useful? Well, for one, the container size should not change. Imagine a sorting algorithm that sorts a list and the result has fewer or more elements than when it started. Surely something fishy is happening there.

As another example, recently I wrote a parser. What are the properties of a parser? One thing I tested was that I was able to retrieve the input from the parser - that is to say, I'm able to make the input and output the same. That test found a bunch of bugs in the parsing logic that I have yet to fix.

## Good Properties To Test ##
Some properties are better than other properties. For example, let's go back to the `Add` example. We know from real life experience, that when you add two numbers together, the result is bigger than either operands. 

Except that isn't true. What if one of the operands were negative numbers? 

OK, so we decide that our `Add` function will only work on positive numbers only. Certainly, adding two positive numbers will result in a number that is larger than either operand. 

That is true for all positive numbers - we can consider this as a property of addition on unsigned integers. In real life though, we have to deal with machines and data types that have physical limitations. Consider this:

```go
func Add(a, b byte) byte { return a + b  }
```

What happens when you do this: `Add(255, 1)`? The result isn't larger! Because of overflows the result will be `1`. Now it can be quite difficult to figure out if an overflow has happened (unless if you are writing in assembly, then just check for the overflow flag). So, maybe this isn't a great property to test. 

## PBT You Probably Already Do ##

One of the interesting things about property based testing is that you probably already do some of it. Take the `encoding/gob` package for example. The [EncodeDecode example](https://golang.org/pkg/encoding/gob/#pkg-examples) in the gob package is a very restricted form of property based testing - it's not testing as many inputs as possible, but it wouldn't be difficult to add a generator to do that.

Think about the properties that the `gob` package is trying to test. It's testing for the fact that the **data does not change** after the encoding/decoding process.

## What Is Property Based Testing ##

Given the descriptions of the above, it's easy to figure out what property-based testing, and what it isnt'. Property-based testing is all about testing that a program satisfies the specified properties, across a wide variety of input. Properties are usually abstract notions of something that is invariant. 

Usually a property based testing library comes with generators to generate input values to test. But that's just a technical part. Theoretically you *could* do property-based testing if you provide your own corpus of test inputs. I once consulted for a media company. Because of the vast library available, we could just use those as inputs to test (videos make for good random inputs). 

A notion many people get confused about is that property-based testing has to be exactly like what QuickCheck does. Anything that doesn't implement combinators and use a HM-ish type system therefore aren't property-based testing. While I will say that there are some nice things that exist in QuickCheck due to those, the idea of **property-based testing does not require them**. 

# Real World Property Based Testing #

All the text above is quite dry and hypothetical in nature. You might be thinking, "but I don't write functions that are commutative or associative". Rest assured that property based testing is useful in real life programming. Real life property based testing is also a lot less strict in definition in what a property is.

The [Gorgonia Tensor](https://gorgonia.org/tensor) package that provides fast-ish generic multidimensional slices. It has fairly complicated machinery to make it fast and generic enough for deep learning work (if you're interested, I laid out the foundations [in a separate post](https://blog.chewxy.com/2017/09/11/tensor-refactor/)). 

It's generic to both data type (`float64`, `float32` etc), execution engines (CPU, GPU, or even the cloud) and execution options (whether to perform operations in place, etc). This leaves a LOT of possible code paths. The more code paths there are, the more chances of error there are. 

For that package I use property based testing to check the assumptions that I have. Here for example, is a snippet for checking that `Log10` works. 

```go
func TestLog10(t *testing.T) {
	var r *rand.Rand

	// default function operation
	invFn := func(q *Dense) bool {
		a := q.Clone().(*Dense)
		correct := a.Clone().(*Dense)
		we, willFailEq := willerr(a, floatTypes, nil)
		_, ok := q.Engine().(Log10er)
		we = we || !ok

		// we'll exclude everything other than floats
		if err := typeclassCheck(a.Dtype(), floatTypes); err != nil {
			return true
		}
		ret, err := Log10(a)

		// ok is a bool indicating it's ok to return early
		if err, ok := qcErrCheck(t, "Log10", a, nil, we, err); ok {
			if err != nil {
				return false
			}
			return true
		}

		ten := identityVal(10, a.Dtype())
		Pow(ten, ret, UseUnsafe())

		cd := correct.Data()
		rd := ret.Data()
		if !qcEqCheck(t, a.Dtype(), willFailEq, cd, rd) {
			return false
		}
		return true
	}
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	if err := quick.Check(invFn, &quick.Config{Rand: r}); err != nil {
		t.Errorf("Inv tests for Log10 failed: %v", err)
	}
}
```


Allow me to walk through the test. The test is testing the function `Log10`. The property of the function that this test performs is that the inverse function should yield a result that is the input. In other words, we want to prove that `Log10` works by saying that `10 ^ (Log10(x)) == x`.

Because there are some helper functions, here's the line-by-line walkthrough:

* `invFn := func(q *Dense) bool` - this line defines the property we're testing. 
* `correct := a.Clone().(*Dense)` - the end result should be the same as the input.
* `we, willFailEq := willerr(a, floatTypes, nil)` - Sometimes values may be inaccessible (for example, values in a GPU can only be accessed in the GPU. Attempting to access values via Go will cause the program to crash). Fortunately these information would be known upfront. If the generator generates an inaccessible value, then we know that the function will return an error. Furthermore, some functions may not return an error, but the values cannot be compared for equality. 
* `_, ok := q.Engine().(Log10er)` - if the associated execution engine cannot perform `Log10`, it will return an error. Again, more upfront information we'd need to know from what was generated.
* `if err := typeclassCheck(a.Dtype(), floatTypes)` - because the `tensor` package is fairly generic, there would be data types of which `Log10` wouldn't make sense - `string` for example. In this case, we'd get rid of any generated `*Dense` tensors that aren't float types.
* `ret, err := Log10(a)` - actually perform the `Log10` operation
* `err, ok := qcErrCheck(t, "Log10", a, nil, we, err); ok` - check that if the operation errors as expected, or created no errors. If there were errors, and it's safe to return early (i.e. an error was indeed expected), the function will return early.
* `ten := identityVal(10, a.Dtype())` - create a value that is a representation of `10`, in the `Dtype` provided.
* `Pow(ten, ret, UseUnsafe())` - perform `10 ^ ret`. This is the inverse operation of `Log10`. `UseUnsafe` is a function option of Gorgonia's tensor, which allows the operation to be done in-place so additional memory wouldn't have to be allocated.
* `if !qcEqCheck(t, a.Dtype(), willFailEq, cd, rd)` - check that the result is the same.

It should be noted that the code contains some bad practices - `willFailEq` will always be `True` in the code above. But in the checking code (`qcEqCheck`), if the data type is actually a float type (`float32`, `float64` etc), the `willFailEq` will be ignored, and a float [approximate-equality check](http://floating-point-gui.de/errors/comparison/) will be used instead.  Floats are treated differently - because floats are used a lot more in machine learning, and there is a greater imperative that they be tested more thoroughly.

As you can see there is quite a bit of set up for a fairly complex machinery. The upside is that there are many thousands of values tested and I'm fairly sure that for a large percentage of inputs, the function wouldn't break.


# Pitfalls #

As with anything, there are some pitfalls to property based testing. For one, property based testing doesn't exist in a vacuum. I don't want readers walking away from the post thinking that property-based testing is the end-all of writing robust software. 

Instead I think it's a combination of various types of testing (traditional unit testing, fuzz testing, property based testing, integration testing), type systems and sound practices (code review, etc).

However, by itself, property based testing is not exhaustive. If you only test for one property of the function you are testing, you only test for that one property. When you test multiple properties of a function, you are approaching a more complete testing for the semantic correctness of the program. 

There are also some edge cases with the notion of property based testing that needs some understanding. Take for example, the equality testing of tensors in Gorgonia: 

```go
func TestDense_Eq(t *testing.T) {
	eqFn := func(q *Dense) bool {
		a := q.Clone().(*Dense)
		if !q.Eq(a) {
			t.Error("Expected a clone to be exactly equal")
			return false
		}
		a.Zero()

		// Bools are excluded because the probability 
		// of having an array of all false is very high
		if q.Eq(a) && a.len() > 3 && a.Dtype() != Bool {
			t.Errorf("a %v", a.Data())
			t.Errorf("q %v", q.Data())
			t.Error("Expected *Dense to be not equal")
			return false
		}
		return true
	}
	conf := &quick.Config{Rand: newRand(), MaxCount: 2000}
	if err := quick.Check(eqFn, conf); err != nil {
		t.Errorf("Failed to perform equality checks")
	}
}
```

Here we are doing two tests (which I rolled into one):

1. Any clone of the `*Dense` *has* to be equal with its original. This test runs on the assumption that the `Clone` method works correctly - this is a very loose form of reflexitivity
2. Any `*Dense` would be not be equal to a `*Dense` that is filled with its zero value (with the exception that the original is all zeroes).

Here we run into the edge case of probabilities: it's highly probable that the generator generates `[]bool{False, False, False}` - so if that condition is met, then we just say the test passes.

This is a sign of poor thinking. I clearly didn't devote enough time and effort into thinking about the properties of an equality test across all possible types. To be clear, here are the things that *should* have been tested:

* Transitivity - `a == b` and `b == c` implies `a == c`
* Symmetry - `a == b` implies `b == a`
* Reflexivity - `a == a`. This was originally done without using `Clone`. Reflexivity is all about testing of values. If `a.Eq(a)` is done, the `Eq` function would note that they're the same object, and return `true`. Instead, a clone, which is a different object but has the same value is used.

These tests were written for most of the comparison operators, but not for equality. I attribute the problem firstly to my bad habit of [selectively choosing what to test](https://blog.chewxy.com/2017/01/04/what-to-test/), followed by undermining the purpose of property-based tests. If you find yourself writing a lot of exceptions to the rules of the property that you're testing, maybe it's time to rethink the test.

# Advanced Libraries #

One of the nice things that Haskell's QuickCheck does that Go's built in `"testing/quick"` doesn't have is shrinking. Shrinking is the idea that the library is able to find the minimum reproductible test case for the test to fail. And as you can see with my real world example, I was pushing `"testing/quick"` to the limits, with a lot of weird looking escape hatches for my code.

Enter [GOPTER](https://github.com/leanovate/gopter). It does shrinking, and has many helpers for stateful tests. It also comes with some code generators to help with writing tests.

While I have played with GOPTER once or twice, I haven't had an opportunity to actually use GOPTER in a real world setting. The biggest benefit I see of GOPTER over `"testing/quick"` is shrinking. The problem is most of my code deals with mathematical primitives - when an error occurs, I don't need a shrinker, I can automatically deduce what went wrong in my head. I aim to actually use GOPTER a lot more than `"testing/quick"` in the coming year. 


# Conclusion #

In this post, I introduced the notion of a useful program, how we would test for some correctness, and the different tools available before we come to the meat of the article: property based testing. I then gave an overview of how to do basic property testing, a real world example, and some pitfalls. Finally I showcased an advanced library for property based testing. If you would like to read more, here are some real world projects that use property based testing to good effect - it was very instructive for me to study them to see how other people do property based testing:

* [time](https://golang.org/src/time/time_test.go) 
* [miekg/dns](https://github.com/miekg/dns/blob/master/parse_test.go) 
* The standard library is actually quite littered with `testing/quick` based tests.

Lastly, please feel free to reach out to me if there are any questions. I'm almost always available to help.

# About The Author #

Chewxy is an author of the [Gorgonia](https://gorgonia.org/gorgonia) suite of libraries. He has a vested interest in making Go the de facto language for machine learning and AI related things (mainly because Go is the right balance between high development speed and high maintainability). He's passionate about natural language processing, linguistics and statistics. Follow [@chewxy](https://twitter.com/chewxy) on Twitter.