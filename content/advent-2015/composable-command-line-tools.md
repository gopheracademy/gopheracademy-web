+++
author = ["Mat Ryer"]
date = "2015-12-12T00:00:00-08:00"
title = "Composable command-line tools"
series = ["Advent 2015"]
+++

Go's simplicity and exhaustive standard library means writing command-line tools is easy and enjoyable. Following the Go philosophy, if you write programs that are small and focussed, you can end up with a pretty powerful little toolbelt of utilities.

But pretty soon, programs will need to communicate with each other in some way; to share data or issue commands. But before you invest in a messaging queue and heavily complicate your otherwise neat little world, consider using a staple of the OS; pipes (`stdin` standard in and `stdout` standard out - and `stderr` standard error for if things go wrong).

I first explored this technique in depth in [Chapter 4 of Go Programming Blueprints](http://bit.ly/goblueprints), where we built a domain-name generator by mashing up many disparate services. I have since built tools that are now in production that use pipes to stream lots of data from one process to another.

In this article, we will build some descrete but composable programs to explore this idea further. The programs we write are illustrative, calculating the mean average from a set of numbers etc. You would probably never build these for real - it's probably easier to just write a new Go program each time you need it. But it's the festive season, and why shouldn't we have a little fun?

# How it works

Even if you're not familiar with piping, you've certainly seen it in action whenever you've interacted with a program in a shell. Whether it's providing input (like when answering a question) or just reading the output of a program in a terminal, you're using pipes.

The standard out pipe (called `stdout` in computer speak) is, by default, connected to the terminal - which is how you can see what the program is outputting. You can redirect this if you want to, like to a file, or even "pipe it" into another program's standard in pipe (using the pipe character):

    one | two

Here the output from `one` would become the input for `two`.

The standard in pipe (also known as `stdin`) is the opposite; the input for a program. In its simplist form, this is just for prompting the user to continue or not (`Are you sure? [Yn]`), or it can be used to take more information in, such as your name. Or you can pipe in many gigabytes of data.

To see this in action, on a unix machine (I'm sure you can do the same on Windows), open a terminal and type:

    $ echo "Hello"
    Hello

The `echo` command is used to write something to (in the default case) stdout. It just echos back what you give it. By default, this comes through stdout and we see it printed in the terminal.

We are now going to redirect it into another program using the pipe (`|`) character:

    $ echo -n "Hello" | md5
    09f7e02f1290be211da707a266f153b3

Here, we pipe the stdout of `echo "Hello"` into the stdin of the `md5` command - which calculates an MD5 hash of whatever you give it. In this case, we learn that the MD5 hash of "Hello" (minus the quotes) is `09f7e02f1290be211da707a266f153b3`. (The `-n` flag asks echo to omit the trailing line feed).

When `echo` has finished echoing, it terminates which closes the pipe. If we run the `md5` program without piping something into it, it will connect its stdin to the terminal, and we can type what we want hashing.

We're going to use this technique to build some tools of our own.

# Mean

The first program we will write will calculate the mean average from a set of numbers. The updated value will be output each time we have new input. The program will maintain a single mean value for its lifecycle.

We'll put each program in its own folder, and run them from a common parent folder.

Create a folder called `piping`, which will be our parent folder, and another inside it called `mean`.

Add the following code:

```
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
)

func main() {
	var sum, vals int
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		val, err := strconv.Atoi(s.Text())
		if err != nil {
			io.WriteString(os.Stderr, "(ignoring) expected whole numbers: "+err.Error())
			continue
		}
		vals++
		sum += val
		fmt.Println(sum / vals)
	}
}
```

The first thing we do is create a `Scanner`, an extremely useful type from the `bufio` package, that has the ability to read (scan) chunks of data from an input source. In our case, the input source is the famous `os.Stdin`.

We then start scanning with a call to the `Scan` method. Execution will block here until the scanner deems that we have something worth working with - which it decides, by default, when it encounters a line feed character. 

Given the `s.Text()`, we use `strconv` to turn it into an integer that we can work with. If something goes wrong, we write an informative error out (to stderr - since we don't want to pollute our standard out stream). All being well, we increase `vals`, add the value to the `sum`, and output the mean average.

## Building

For now, we'll build each program individually and place it inside a special `piping/cmds` folder. This will make them easy to run. In the real world, they'll likely end up in some appropriate `bin` folder.

  * Create a folder `piping/cmds`
  * Navigate to your program source folder (with `cd mean`)
  * Build it and use the `-o` flag to place it inside the `cmds` folder: `go build -o ../cmds/mean`

## Running

Try this program out by navigating to the `cmds` folder in a terminal, running it, and entering some numbers:

    $ ./mean
    10
    10 << mean

(I've added the `mean` tag to make it a little clearer).

Enter some more numbers, and you'll see the mean average start to migrate:

    $ go run mean/main.go
    10
    10 << mean
    15
    12 << mean
    20
    15 << mean
    5
    12 << mean
    4
    10 << mean
    3
    9 << mean

## Ending programs

The program will only end once the `Scan()` method returns `false` - like when the input stream is closed, which we can do in the terminal by hitting `Ctrl+D`. Then use `Ctrl+C` to end the program altogether.

# Split

The next program we will write is called `split`, which will take a single line of text and split it onto many lines. This will allow us to echo `1,2,3,4` and have it changed into:

    1
    2
    3
    4

This is nice becuase it will allow us to pipe many numbers into our `mean` program without having to keep typing `\n` all over the place.

Create a folder alongside `mean` (inside the `piping` parent folder) called `split`, and add the following code into `main.go`:

```
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	var (
		delimiter = flag.String("delimiter", ",", "character to split by")
	)
	flag.Parse()
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		println(strings.Split(s.Text(), *delimiter)...)
	}
}

func println(s ...string) {
	for _, ss := range s {
		fmt.Println(ss)
	}
}
```

This time, we are using the `flag` package to allow our users a little more control over which character becomes the delimiter. By default, it'll be a comma (`,`) and we won't change this. This was left in to highlight how you can easily take in configuration settings to your programs using only the standard library.

We create the same `Scanner` as before, and use `strings.Split` to break up the input. We then call `fmt.Println` for each value.

Build and run the program in a terminal, and type in `1,2,3`:

    $ ./split
    1,2,3
    1
    2
    3

Press `Ctrl+C` to terminate the program.

Instead of manually typing in the input, we can use the piping technique to take numbers in from another command - such as the `echo` command. Try this:

    $ echo "1,2,3" | ./split
    1
    2
    3

Not only did the program work as before, but notice that it also automatically terminated. This is because `echo` closed the input stream, and `Scan()` therefore returned `false`.

## One mean result

Now we can rely on this stopping mechanism, let's adjust our Mean program to only print the output once, at the end of the program (like the `md5` command does). Update the `main` function in `mean/main.go` with the following code (move the `fmt.Println` line outside the loop):

```
func main() {
	var sum, vals int
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		val, err := strconv.Atoi(s.Text())
		if err != nil {
			io.WriteString(os.Stderr, "(ignoring) expected whole numbers: "+err.Error())
			continue
		}
		vals++
		sum += val
	}
	fmt.Println(sum / vals) // print one result at the end
}
```

Rebuild Mean as you did before, replacing the old binary in the `cmds` folder.

## Split and mean

Now we're going to combine our two programs. We're going to pipe the lines of output from Split into the input of Mean. Be sure to be inside the `cmds` folder:

    $ echo "10,20,35" | ./split | ./mean
    21

Let's trace through what's happening here:

  * Print out `10,20,35` with the `echo` command
  * Take the output from echo and pipe it into the input for Split
  * Split it into many lines, and pipe each line into Mean
  * Mean keeps track of the number of values and sum
  * Echo closes the input
  * Split gets `Scan() == false` and closes its input
  * Mean then gets `Scan() == false` and exits the for loop - where it prints the latest mean average value

Finally, we see the result of the mean average of the numbers 10, 20 and 35, which is 21.

# Summing

To sum a range of numbers, we just need to copy our `mean/main.go` code, and remove the actual mean calculating piece. Create a file at `sum/main.go` with the following:

```
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
)

func main() {
	sum := 0
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		val, err := strconv.Atoi(s.Text())
		if err != nil {
			io.WriteString(os.Stderr, "(ignoring) expected whole numbers: "+err.Error())
			continue
		}
		sum += val
	}
	fmt.Println(sum)
}
```

Build this too and play with it, remmeber you can use the Split program if it's easier:

```
$ echo "1,2,3" | ./split | ./sum
6

$ echo "1,2,3,4,5" | ./split | ./sum
15
```

# Averages and sums of Fibonacci numbers

If we want to find out the mean average of Fibonacci numbers, all we have to do is write a small program that generates Fibonacci numbers for us.

Create a new folder in `piping` called `fib` - and add the following code:

```
package main

import (
	"flag"
	"fmt"
)

func main() {
	var (
		max = flag.Int("max", 21, "maximum number")
	)
	flag.Parse()
	f := fib()
	for {
		i := f()
		if i > *max {
			break
		}
		fmt.Println(i)
	}
}

// fib returns a function that returns
// successive Fibonacci numbers.
// The state is stored in closures.
func fib() func() int {
	a, b := 0, 1
	return func() int {
		a, b = b, a+b
		return a
	}
}
```

  * Some of this code was stolen from the Go website - and [I wrote an explanation how the fibonacci code works](https://medium.com/@matryer/golang-advent-calendar-day-seven-how-the-go-fibonacci-demo-works-538da6b559e9#.y2kca58pt) on my blog for those interested.

Build the program and put the binary in the `cmds` folder as before:

    $ go build -o ../cmds/fib

Running `fib` will generate all Fibonnaci numbers until the specified `max`:

```
$ ./fib -max=21
1
1
2
3
5
8
13
21
```

To see the mean average of these numbers, we just pipe that output into the Mean command:

```
$ ./fib -max=21 | ./mean
6
```

To see the sum of a range of Fibonnaci numbers, just pipe them into our Sum command:

```
$ ./fib -max=21 | ./sum
54
```

This demonstrates how we can compose the programs in different ways, to do different things.

To push it a little, why not try some big numbers too:

```
$ ./fib -max=1000000000000000000 | ./sum
1779979416004714188

$ ./fib -max=1000000000000000000 | ./mean
20459533517295565
```

# So what?

Although our example was a little far fetched, we saw the power of writing small programs with simple interfaces (e.g. lines of numbers) and how they can be composed together to solve deeper problems.

Of course, it doesn't have to be numbers that you are passing around - what about lines of JSON data, configuration files, pictures etc?

Consider the power of mixing your own tools with those of the operating system. The `find` command prints out paths to files matching a pattern, you could use that to extract the sizes of the files, and pipe that into your `sum` command to get the average file size for each directory. Calculating the total size for a directory becomes trivial using your tools.

Whatever your need, consider how you might solve it in an isolated way, and before you know it - you'll be on your way to a library of useful, reusable, composable shell tools.
