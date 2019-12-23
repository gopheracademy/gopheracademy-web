+++
linktitle = "Test Driven Advent of Code"
title = "Test Driven Advent of Code"
date = "2019-12-10T00:00:00+00:00"
author = ["Jon Calhoun"]
series = ["Advent 2019"]
+++

By now most of you have probably heard of [Advent of Code](https://adventofcode.com/). If not, go check it out. I'll give you a few moments...

*Just in case you are too lazy to read, Advent of Code is a 25-day "[advent calendar](https://en.wikipedia.org/wiki/Advent_calendar)" (similar to this blog series) where every day a new problem is unlocked and each problem is typically solved using some code in whatever language you want.*

In this post we are going to look at how to approach and solve Advent of Code problems using Go and tests! 🥳

So first - **why are we using tests?**

When solving problems like the ones in the Advent of Code we are very typically given sample inputs and their correct answers. This already lends itself to using tests, as we can plug every sample input into our test file and continually ensure that they all work as expected as we make changes to our code.

Go's table-driven tests are also great for figuring out what is wrong with your code if you get a problem wrong. Rather than running your code over and over again, you just add new test cases and their expected output and viola, you can quickly start to debug issues.

Finally, tests are great at encouraging you to break your code into smaller, more reusable components which often end up being easier to verify. That means if you have a mistake, you can test each part separately until you figure out which one has the issue.

## On to the code!

I'm going to be using the first two problems from [Advent of Code 2018](https://adventofcode.com/2018) for this post because I don't want to ruin the 2019 problems for anyone. If you decide to try this on your own, I encourage you to use the 2019 problems!

Go ahead and check out the first problem. Get a sense of what it is asking, then return here where we can start discussing how we are going to solve it with Go and tests.

*The first problem can be found here: <https://adventofcode.com/2018/day/1>*

Ready to start? Sweet!

In this first problem we need to add or subtract a series of numbers and keep track of the resulting value as we go. That means we really have two problems we need to solve, and then we need to put those solutions together:

1. We need to parse a string like `+1, +1, +1` into numbers.
2. We need to sum up those numbers.
3. We need a `ChronalCalibration` function that puts these two steps together.

Let's start with the string parsing.

```go
// ParseInts will take a list of integers separated by a comma and a space and
// return an integer slice of those values. It handles positive (+) and negative
// (-) signs in front of the numbers.
func ParseInts(input string) []int {
  return nil
}
```

I'm going to return `nil` for now and go add a few test casts.

```go
func TestParseInts(t *testing.T) {
	eq := func(a, b []int) error {
		if len(a) != len(b) {
			return fmt.Errorf("lengths differ")
		}
		for i := 0; i < len(a); i++ {
			if a[i] != b[i] {
				return fmt.Errorf("index %d", i)
			}
		}
		return nil
	}
	tests := map[string]struct {
		input string
		want  []int
	}{
		"ex1": {"+1, +1, +1", []int{1, 1, 1}},
		"ex2": {"+1, +1, -2", []int{1, 1, -2}},
		"ex3": {"-1, -2, -3", []int{-1, -2, -3}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := ParseInts(tc.input)
			if err := eq(got, tc.want); err != nil {
				t.Errorf("ParseInts() = %v; want %v; mismatch = %v", got, tc.want, err)
			}
		})
	}
}
```

Now this is way more code than we are going to write solving the problem, but most of this is really simple boilerplate that will help verify that our code is correct. We could also opt to use some assertion libraries and other tools to speed this up, but I'm sticking with standard library Go for now.

If we run `go test` we should see some failing tests:

```bash
$ go test
--- FAIL: TestParseInts (0.00s)
    --- FAIL: TestParseInts/ex1 (0.00s)
        01_chronal_calibration_test.go:32: ParseInts() = []; want [1 1 1]; mismatch = lengths differ
    --- FAIL: TestParseInts/ex2 (0.00s)
        01_chronal_calibration_test.go:32: ParseInts() = []; want [1 1 -2]; mismatch = lengths differ
    --- FAIL: TestParseInts/ex3 (0.00s)
        01_chronal_calibration_test.go:32: ParseInts() = []; want [-1 -2 -3]; mismatch = lengths differ
FAIL
exit status 1
FAIL	advent2018	0.043s
```

Next we are going to actually write the `ParseInts` function.

*Sidenote: The [`strings`](https://golang.org/pkg/strings/) and [`strconv`](https://golang.org/pkg/strconv/) packages are incredibly helpful when parsing string inputs. You should definitely check them out if you are unfamiliar.*

```go
func ParseInts(input string) []int {
	entries := strings.Split(input, ", ")
	var ints []int
	for _, entry := range entries {
    // Hmm... does this handle things like "+1" and "-1"?
    num, err := strconv.Atoi(entry)
    if err != nil {
			panic(err)
		}
		ints = append(ints, num)
	}
	return ints
}
```

While writing this we might ask ourselves - does the `strconv.Atoi` function handle things like the positive and negative sign in a number? Or do we need to handle that ourselves?

A quick glance at the docs isn't very telling. Chances are it will support the negative sign fine, but I'm not really sure on the positive sign. Luckily we have some tests to help us figure that out, so let's give it a shot!

```bash
$ go test
PASS
ok  	advent2018	0.149s
```

Cool, it looks like it works! That's one less thing we need to handle.

Now let's move on to our second step - we need a function that sums up a [slice](https://tour.golang.org/moretypes/7) of integers.

```go
// Sum will sum up all of the values in the provided slice and return the result
func Sum(nums []int) int {
	return 0
}
```

And now we can add some tests.

```go
func TestSum(t *testing.T) {
	tests := map[string]struct {
		input []int
		want  int
	}{
		"ex1": {[]int{1, 1, 1}, 3},
		"ex2": {[]int{1, 1, -2}, 0},
		"ex3": {[]int{-1, -2, -3}, -6},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := Sum(tc.input)
			if got != tc.want {
				t.Errorf("Sum() = %v; want %v", got, tc.want)
			}
		})
	}
}
```

*Don't be afraid to use tools your editor provides you, like multiselect and data from your last test case!*

![Using VS Code's multiselect to add the second test case](/postimages/advent-2019/advent-of-code/advent2019-editor.gif)

One of our tests happens to test without doing anything. Don't worry though, everything is fine; this is just dumb luck. Our stubbed out `Sum` function happens to be returning `0` which is the correct answer every once in a while 😬.

```go
// intentionally wrong!
func Sum(nums []int) int {
  var sum int
	for num := range nums {
		sum += num
	}
	return sum
}
```

And we run our tests...

```bash
$ go test
--- FAIL: TestSum (0.00s)
    --- FAIL: TestSum/ex3 (0.00s)
        01_chronal_calibration_test.go:51: Sum() = 3; want -6
    --- FAIL: TestSum/ex2 (0.00s)
        01_chronal_calibration_test.go:51: Sum() = 3; want 0
FAIL
exit status 1
FAIL	advent2018	0.140s
```

Uhhh, what? Why is this wrong?

If you go back to our `Sum` function you might be able to find the bug yourself. We are using the index of each item in the `nums` slice rather than its value. 🤦‍♂️

Let's fix that up and re-run our tests.

```go
func Sum(nums []int) int {
	var sum int
	for _, num := range nums {
		sum += num
	}
	return sum
}
```

```bash
$ go test
PASS
ok  	advent2018	0.154s
```

Sweet!. Now our last step is to plug this all together to create the `ChronalCalibration` function. I'm just going to go ahead and implement this.

```go
func ChronalCalibration(input string) int {
  nums := ParseInts(input)
  return Sum(nums)
}
```

Then we can add tests.

```go
func TestChronalCalibration(t *testing.T) {
	tests := map[string]struct {
		input string
		want  int
	}{
		"ex1": {"+1, +1, +1", 3},
		"ex2": {"+1, +1, -2", 0},
		"ex3": {"-1, -2, -3", -6},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := ChronalCalibration(tc.input)
			if got != tc.want {
				t.Errorf("ChronalCalibration() = %v; want %v", got, tc.want)
			}
		})
	}
}
```

And run then with the `-v` flag to make sure everything is running as expected.

```bash
$ go test -v
=== RUN   TestParseInts
=== RUN   TestParseInts/ex1
=== RUN   TestParseInts/ex2
=== RUN   TestParseInts/ex3
--- PASS: TestParseInts (0.00s)
    --- PASS: TestParseInts/ex1 (0.00s)
    --- PASS: TestParseInts/ex2 (0.00s)
    --- PASS: TestParseInts/ex3 (0.00s)
=== RUN   TestSum
=== RUN   TestSum/ex2
=== RUN   TestSum/ex3
=== RUN   TestSum/ex1
--- PASS: TestSum (0.00s)
    --- PASS: TestSum/ex2 (0.00s)
    --- PASS: TestSum/ex3 (0.00s)
    --- PASS: TestSum/ex1 (0.00s)
=== RUN   TestChronalCalibration
=== RUN   TestChronalCalibration/ex2
=== RUN   TestChronalCalibration/ex3
=== RUN   TestChronalCalibration/ex1
--- PASS: TestChronalCalibration (0.00s)
    --- PASS: TestChronalCalibration/ex2 (0.00s)
    --- PASS: TestChronalCalibration/ex3 (0.00s)
    --- PASS: TestChronalCalibration/ex1 (0.00s)
PASS
ok  	advent2018	0.044s
```

We are finally ready to get our input for the real test from Advent of Code, but wait a minute... This test input looks different!

```
+11
+9
+15
-17
...
```

The input we are given is separate by lines, not commas!

There are a few ways to handle this, but the simplest is probably to just add a separator field to our `ParseInts` and `ChronalCalibration` functions.

```go
func ParseInts(input, sep string) []int {
	entries := strings.Split(input, sep)
	var ints []int
	for _, entry := range entries {
		num, err := strconv.Atoi(entry)
		if err != nil {
			panic(err)
		}
		ints = append(ints, num)
	}
	return ints
}

func ChronalCalibration(input, sep string) int {
	nums := ParseInts(input, sep)
	return Sum(nums)
}
```

And to then update our tests.

```go
func TestParseInts(t *testing.T) {
	eq := func(a, b []int) error {
		if len(a) != len(b) {
			return fmt.Errorf("lengths differ")
		}
		for i := 0; i < len(a); i++ {
			if a[i] != b[i] {
				return fmt.Errorf("index %d", i)
			}
		}
		return nil
	}
	tests := map[string]struct {
		input string
		sep   string
		want  []int
	}{
		"ex1": {"+1, +1, +1", ", ", []int{1, 1, 1}},
		"ex2": {"+1, +1, -2", ", ", []int{1, 1, -2}},
		"ex3": {"-1, -2, -3", ", ", []int{-1, -2, -3}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := ParseInts(tc.input, tc.sep)
			if err := eq(got, tc.want); err != nil {
				t.Errorf("ParseInts() = %v; want %v; mismatch = %v", got, tc.want, err)
			}
		})
	}
}

func TestChronalCalibration(t *testing.T) {
	tests := map[string]struct {
		input string
		sep   string
		want  int
	}{
		"ex1": {"+1, +1, +1", ", ", 3},
		"ex2": {"+1, +1, -2", ", ", 0},
		"ex3": {"-1, -2, -3", ", ", -6},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := ChronalCalibration(tc.input, tc.sep)
			if got != tc.want {
				t.Errorf("ChronalCalibration() = %v; want %v", got, tc.want)
			}
		})
	}
}
```

Now we can add a new test case with multi-line inputs. To do this I'm just going to add a constant to the bottom of my test file.

```go
const chronalCalibrationP1 = `+11
+9
+15
-17
+8
+16
+5
...
+3
-12
+124236`
```

*Be sure to actually use the real input, as mine is truncated here.*

Now we can add the test case to `TestChronalCalibration`:

```go
"real": {chronalCalibrationP1, "\n", 0},
```

And run it, expecting our test to fail but to also give us the correct answer in the failure message.

```bash
$ go test
--- FAIL: TestChronalCalibration (0.00s)
    --- FAIL: TestChronalCalibration/real (0.00s)
        01_chronal_calibration_test.go:73: ChronalCalibration() = 430; want 0
FAIL
exit status 1
FAIL	advent2018	0.061s
```

Then we plug 430 into the Advent of Code website and... it is correct! On to part two of the problem.


## Part Two

Hopefully part two will allow us to reuse some of our work, so let's go ahead and read and find out. Part two can be found below the first part on the [day 1](https://adventofcode.com/2018/day/1) page of 2018.

It looks like we are hunting cycles now, so let's copy our part 1 `TestChronalCalibration` test and create our new set of tests.

```go
func TestChronalCalibrationP2(t *testing.T) {
	tests := map[string]struct {
		input string
		sep   string
		want  int
	}{
		"ex0": {"+1, -2, +3, +1", ", ", 2},
		"ex1": {"+1, -1", ", ", 0},
		"ex2": {"+3, +3, +4, -2, -4", ", ", 10},
		"ex3": {"-6, +3, +8, +5, -6", ", ", 5},
		"ex4": {"+7, +7, -2, -7, -4", ", ", 14},
		// "real": {chronalCalibrationP1, "\n", 430},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := ChronalCalibrationP2(tc.input, tc.sep)
			if got != tc.want {
				t.Errorf("ChronalCalibrationP2() = %v; want %v", got, tc.want)
			}
		})
	}
}
```

Now we need to add the `ChronalCalibrationP2` function. I typically stub it out, run the tests to see them fail, then start implementing it, but right now I'm just going to present an (incorrect) solution using a map to keep track of seen values.

```go
func ChronalCalibrationP2(input, sep string) int {
	nums := ParseInts(input, sep)

	seen := make(map[int]bool, 0)
	var sum int
	for _, val := range nums {
		if _, ok := seen[sum]; ok {
			return sum
		}
		seen[sum] = true
		sum += val
	}
  // We shouldn't be able to get here.
	panic("inconceivable!")
}
```

If we test this, it will panic. Hmm, that means something must be wrong. How are we getting to a part of our code that isn't reachable?

It looks like we made a mistake and forgot to wrap our for loop in an infinite loop! Let's fix that and see what happens.

```go
func ChronalCalibrationP2(input, sep string) int {
	nums := ParseInts(input, sep)

	seen := make(map[int]bool, 0)
	var sum int
	for {
		for _, val := range nums {
			if _, ok := seen[sum]; ok {
				return sum
			}
			seen[sum] = true
			sum += val
		}
	}
	// We shouldn't be able to get here.
	panic("inconceivable!")
}
```

```bash
$ go test
PASS
ok  	advent2018	0.044s
```

Now that is what we wanted!

The test input for this problem hasn't changed, so we can go ahead and just reuse our input as the new test case.

```go
func TestChronalCalibrationP2(t *testing.T) {
  // ...

    "real": {chronalCalibrationP1, "\n", 0},

  // ...
}
```

Running our test shows us that it got 462 as the answer. Let's plug that into the Advent of Code website and see if it is right.

```
That's the right answer! You are one gold star closer to fixing the time stream.
```

Awesome, we have completed Day 1!

## Wrapping up

Hopefully this post has helped demonstrate how Go's tests fit in pretty nicely with problems like those presented in the Advent of Code. You can also take this approach when working on things like [Google Code Jam](https://codingcompetitions.withgoogle.com/codejam/archive), but just remember that you might need to handle multiple inputs in those contests so you may need a `main` package to drive your code. Even then, tests can be incredibly helpful when trying to ensure your code is working as expected.

Now go out there and rock the 2019 version of [Advent of Code](https://adventofcode.com/)!
