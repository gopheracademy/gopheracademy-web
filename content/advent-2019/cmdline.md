+++
title = "Writing Friendly Command Line Applications"
date = "2019-12-08T00:00:00+00:00"
series = ["Advent 2019"]
author = ["Miki Tebeka"]
linktitle = "flag"
+++

Let me tell you a story...

In 1986 [Knuth](https://en.wikipedia.org/wiki/Donald_Knuth) wrote a program to
demonstrate [literate
programming](https://en.wikipedia.org/wiki/Literate_programming).

The task was to read a file of text, determine the n most frequently used
words, and print out a sorted list of those words along with their frequencies.
Knuth wrote a beautiful 10 page monolithic program.

Doug Mcllory read this and said 
`tr -cs A-Za-z '\n' | tr A-Z a-z | sort | uniq -c | sort -rn | sed ${1}q`

It's 2019, why am I telling you a story that happened 33 years ago? (Probably
before some of you were born). The computation landscape has changed a lot...
or has it?

The [Lindy effect](https://en.wikipedia.org/wiki/Lindy_effect) is a concept
that the future life expectancy of some non-perishable things like a technology
or an idea is proportional to their current age. TL;DR - old technologies are
here to stay.

If you don't believe me, see:

- [oh-my-zsh](https://github.com/ohmyzsh/ohmyzsh) having close to 100,000 starts on GitHub
- [Data Science at the Command Line](https://www.datascienceatthecommandline.com/) book
- [Command-line Tools can be 235x Faster than your Hadoop Cluster](https://adamdrake.com/command-line-tools-can-be-235x-faster-than-your-hadoop-cluster.html)
- ...

Now that you are convinced, let's talk on how to make your Go programs command
line friendly.

## Design

When writing command line application, try to adhere to the [basics of Unix
philosophy](http://www.catb.org/esr/writings/taoup/html/ch01s06.html)

- Rule of Modularity: Write simple parts connected by clean interfaces.
- Rule of Composition: Design programs to be connected with other programs.
- Rule of Silence: When a program has nothing surprising to say, it should say nothing.

If you don't follow these and let your command line interface grow organically,
you might end up in the following situation

[![](https://imgs.xkcd.com/comics/tar.png)](https://xkcd.com/1168/)


## Help

Let's assume your team have a `nuke-db` utility. You forgot how to invoke it
and you do:

```
$ ./nuke-db --help
database nuked
```

Ouch!

Using the [flag](https://golang.org/pkg/flag/), you can add support for `--help` in 2 extra lines of code

```go
package main

import (
	"flag" // extra line 1
	"fmt"
)

func main() {
	flag.Parse() // extra line 2
	fmt.Println("database nuked")
}
```

Now your program behaves

```
$ ./nuke-db --help
Usage of ./nuke-db:
$ ./nuke-db
database nuked
```

If you'd like to provide more help, use `flag.Usage`

```go
package main

import (
	"flag"
	"fmt"
	"os"
)

var usage = `usage: %s [DATABASE]

Delete all data and tables from DATABASE.
`

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	fmt.Println("database nuked")
}
```

And now
```
$ ./nuke-db --help
usage: ./nuke-db [DATABASE]

Delete all data and tables from DATABASE.
```



# About the Author
Hi there, I'm Miki, nice to e-meet you ☺. I've been a long time developer and
have been working with Go for about 10 years now. I write code professionally as
a consultant and contribute a lot to open source. Apart from that I'm a [book
author](https://www.amazon.com/Forging-Python-practices-lessons-developing-ebook/dp/B07C1SH5MP),
an author on [LinkedIn
learning](https://www.linkedin.com/learning/search?keywords=miki+tebeka), one of
the organizers of [GopherCon Israel](https://www.gophercon.org.il/) and [an
instructor](https://www.353.solutions/workshops).  Feel free to [drop me a
line](mailto:miki@353solutions.com) and let me know if you learned something
new or if you'd like to learn more.
