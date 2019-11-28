+++
title = "Fun With Flags"
date = "2019-12-08T00:00:00+00:00"
series = ["Advent 2019"]
author = ["Miki Tebeka"]
+++

In a [previous article](TODO) we discussed why command line applications are
important and talk about few guidelines. In this article we'll see how we can
use the built-in [flag](https://golang.org/pkg/flag/) package to write command
line applications.

There are other third-party packages for writing command line interfaces, see
[here](https://github.com/avelino/awesome-go#command-line) for a list. However
depending on third-party package [carries a
risk](https://research.swtch.com/deps) and I prefer to use the standard library
as much as I can.

## Converting Time Zones

Let's write a small application to convert time from one time zone to the
other.

# About the Author
Hi there, I'm Miki, nice to e-meet you ☺. I've been a long time developer and
have been working with Go for about 10 years. I write code professionally as
a consultant and contribute a lot to open source. Apart from that I'm a [book
author](https://www.amazon.com/Forging-Python-practices-lessons-developing-ebook/dp/B07C1SH5MP) author, an author on [LinkedIn
learning](https://www.linkedin.com/learning/search?keywords=miki+tebeka), one of
the organizers of [GopherCon Israel](https://www.gophercon.org.il/) and [an
instructor](https://www.353.solutions/workshops).  Feel free to [drop me a
line](mailto:miki@353solutions.com) and let me know if you learned something
new or if you'd like to learn more.

---
- options struct
- FlagSet
- flag.Var

