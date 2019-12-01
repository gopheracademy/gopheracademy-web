+++
title = "Fun With Flags"
date = "2019-12-08T00:00:00+00:00"
series = ["Advent 2019"]
author = ["Miki Tebeka"]
+++

In a [previous article](FIXME) we discussed why command line applications are
important and talked about few guidelines. In this article we'll see how we can
use the built-in [flag](https://golang.org/pkg/flag/) package to write command
line applications.

There are other third-party packages for writing command line interfaces, see
[here](https://github.com/avelino/awesome-go#command-line) for a list. However
depending on third-party package [carries a
risk](https://research.swtch.com/deps) and I prefer to use the standard library
as much as I can.

## httpd

Let's write a HTTP server. It'll take the host & port to listen on from the
command line.

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

var config struct {
	port int
	host string
}

const (
	usage = `usage: %s
Run HTTP server

Options:
`
)

func main() {
	flag.IntVar(&config.port, "port", config.port, "port to listen on")
	flag.StringVar(&config.host, "host", config.host, "host to listen on")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	http.HandleFunc("/", handler)
	addr := fmt.Sprintf("%s:%d", config.host, config.port)
	fmt.Printf("server ready on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("error: %s", err)
	}

}

func init() {
	// Set defaults
	s := os.Getenv("HTTPD_PORT")
	p, err := strconv.Atoi(s)
	if err == nil {
		config.port = p
	} else {
		config.port = 8080
	}

	h := os.Getenv("HTTPD_HOST")
	if len(h) > 0 {
		config.host = h
	} else {
		config.host = "localhost"
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello Gophers\n")
}

```

- I tend to use a `config` struct for configuration instead of separate
  variables. When applications evolve, the number of configuration option will
  grows and I'd like to keep them in one place.
- `flag.IntVar` will bind `config.port` to the `-port` command line option
- `flag.StringVar` will bind `config.host` to the `-host` command line option
- Set `flag.Usage` to a function that will print your help
- `flag.Parse` will parse command line arguments and will print help when
  calling your application with `-h` or `--help`. `flag.Parse` will exit the
  program on any command line error
- You can use `init` to set default values and populate values from environment
  variables

## Validation

A good practice is to validate all the command line switches at program start.
The `flag` packages have built in function for integers, floats, boolean,
time.Duration and more. However sometimes you'd like to have your own type.
Using `flag.Var` we can achieve this.

We'll define `portVar` struct that will implement the
[flag.Value](https://golang.org/pkg/flag/#Value) interface. We'll also provide
a `PortVar` function to create such a variable.

```go
func PortVar(port *int) *portVar {
	return &portVar{port}
}

type portVar struct {
	port *int
}

func (p *portVar) String() string {
	if p.port == nil {
		return ""
	}

	return fmt.Sprintf("%d", *p.port)
}

func (p *portVar) Set(s string) error {
	val, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	const minPort, maxPort = 1, 65535
	if val < minPort || val > maxPort {
		return fmt.Errorf("port %d out of range [%d:%d]", val, minPort, maxPort)
	}

	*p.port = val
	return nil
}
```

## Sub Commands

Instead of having one executable to start the HTTP server and another to check
it's alive. We can have one executable that does both commands (same as `git`
have many commands). We can do that with [flag.FlagSet](https://golang.org/pkg/flag/#FlagSet).

```go

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s httpd|check\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(os.Args) < 2 {
		log.Fatalf("error: wrong number of arguments")
	}

	switch os.Args[1] {
	case "httpd":
		runHTTPD()
	case "check":
		runCheck()
	default:
		log.Fatalf("error: unknown command - %s", os.Args[1])
	}

}

func runCheck() {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), checkUsage, os.Args[0])
		fs.PrintDefaults()
	}

	fs.Parse(os.Args[2:])
	if fs.NArg() != 1 {
		log.Fatalf("error: wrong number of arguments")
	}

	url := fs.Arg(0)
	resp, err := http.Get(url)
	switch {
	case err != nil:
		log.Fatalf("error: %s", err)
	case resp.StatusCode != http.StatusOK:
		log.Fatalf("error: bad status - %s", resp.Status)
	}
}

func runHTTPD() {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	fs.Var(PortVar(&config.port), "port", "port to listen on")
	fs.StringVar(&config.host, "host", config.host, "host to listen on")
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), httpdUsage, os.Args[0])
		fs.PrintDefaults()
	}
	fs.Parse(os.Args[2:])

	http.HandleFunc("/", handler)
	addr := fmt.Sprintf("%s:%d", config.host, config.port)
	fmt.Printf("server ready on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("error: %s", err)
	}
}
```


# Conclusion

The `flag` package is flexible and will probably support all of your command
line parsing needs. It might be more verbose than other packages but it's in
the standard library so you don't need any extra dependencies and can count on
it's API keeping the [Go compatibility
promise](https://golang.org/doc/go1compat).


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
