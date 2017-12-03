+++
author = ["Chewxy"]
title = "Property Based Testing"
linktitle = "Short Title (use when necessary)"
date = 2017-12-04T06:40:42Z
+++

# Useful Programs #

I'll start this post with a bold but fairly uncontroversial statement: Programs that do not interact with the outside world are useless - they do nothing except consume cycles. A corollary of this is that pure functional programming is useless. Here is a (mostly) pure functional program written in Go that does absolutely nothing:

```go
package foo

func incr(i int) int { return i+1 }
func decr(i int) int { return i-1 }

func foo() {
	decr(incr(0))
}
```

While obviously there are other things that make a program useful, the most fundamental thing that makes a program useful is that it needs to interact with things outside itself. This can come in the form of reading a file, reading a network input, reading user input, printing an output, or writing to a file. Indeed, so fundamental is input/output to the notion of programming that most languages have a standardized definition for a program entry point. In Go, it's the `func main(){}` function. 

Another feature about useful programs that they have to be robust. Here, the notion of fragile programs can be induced by an example - imagine a program that breaks down at every other input that is fed to it. It's not very useful, is it?

Lastly, a program has to do what we intend for it to do in order to be useful. If I want a program to add two numbers together, a useless program would be one that plays music instead of calculating the sum of two numbers. 

To recap, a useful program has to:

* Deal with I/O
* Be robust to various inputs
* Do the correct thing

# Mo' Inputs Mo' Problems #

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
		t.Errorf("Add failed. Expected the result to be 3. Got %d instead", out)
	}
}
```

Then run `go test`. It just works.

# The Case Of Perverse Incentives #

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

Enter the notion of [fuzz testing](https://en.wikipedia.org/wiki/Fuzzing): we feed the program random inputs, and then watch for it to fail. Fuzz testing often leads one to find subtle bugs you don't expect. Here's one that Chris Marshall discovered on my [skiprope](https://github.com/chewxy/skiprope) library. I have to this day, no idea what the correct fix is. Solutions welcome

# A More Disciplined Approach #

Fuzz testing works by throwing everything at the wall and seeing what sticks. It's extremely useful for finding bugs you hadn't reasoned out yet. That typically falls under the purview of "be robust to various inputs" part of building useful programs. But it's not good enough to test what inputs work and what inputs cause a `panic`. You also want to test that those inputs are doing the correct things. 

"Doing the correct things" means thinking deeply about what your program is supposed to do. You gotta be honest with yourself - if you are testing an `Add` function on integers, you can't use `Add` or `+` to verify the results. That's the circular logic trap that many fall into. I find myself also occasionally falling into the same trap due to laziness.

Testing for results is tedious anyway - you have to know that the result of `1 + 2` is `3`. And that's for the simple stuff. Imagine having to know the answers ahead of time for random input for more complicated programs.

# Properties #

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

So, with the Whys out of the way, let's talk about how to do property testing in Go. Go actually comes with a built in property testing tool, `"testing/quick"`. Given that property based testing is based on Haskell's QuickCheck, naturally the testing function is called `quick.Check`. The gist is simple: write your property as a function, pass it into the `quick.Check` function, and it will generate values to throw at the function to test.

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

The `"testing/quick"` library comes with functions that generate values based on types. This is what is used to generate the inputs to be used in the property.


# How To Think About Properties #

## Property Based Testing You Probably Already Do ##

One of the interesting things about property based testing is that you probably already do some of it. Take the `gob` package for example. The example in the gob package is a form of property based testing - granted it's not tested a large variety of inputs, but the example is an excellent starting point of a property based testing methodology.


# Advanced Libraries # 

TODO: GoPTer

