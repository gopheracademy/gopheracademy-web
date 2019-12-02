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
[here](https://github.com/avelino/awesome-go#command-line) for a list of them.
However depending on third-party package [carries a
risk](https://research.swtch.com/deps) and I prefer to use the standard library
as much as I can.

## httpd

Let's write an HTTP server. It'll take the host & port to listen on from the
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

var config struct { // [1]
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
	flag.IntVar(&config.port, "port", config.port, "port to listen on")    // [2]
	flag.StringVar(&config.host, "host", config.host, "host to listen on") // [3]
	flag.Usage = func() {                                                  // [4]
		fmt.Fprintf(flag.CommandLine.Output(), usage, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse() // [5]

	http.HandleFunc("/", handler)
	addr := fmt.Sprintf("%s:%d", config.host, config.port)
	fmt.Printf("server ready on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("error: %s", err)
	}

}

func init() { // [6]
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

1. I tend to use a `config` struct for configuration instead of separate
  variables. When applications evolve, the number of configuration option will
  grows and I'd like to keep them in one place
2. `flag.IntVar` will bind `config.port` to the `-port` command line option
3. `flag.StringVar` will bind `config.host` to the `-host` command line option
4. Set `flag.Usage` to a function that will print your help
5. `flag.Parse` will parse command line arguments and will print help when
  calling your application with `-h` or `--help`. `flag.Parse` will exit the
  program on any command line error
6. You can use `init` to set default values and populate values from environment
  variables

## Validation

A good practice is to validate all the command line switches at program start.
The `flag` packages have built in function for integers, floats, boolean,
time.Duration and more. However sometimes you'd like to have your own type.
Using `flag.Var` we can achieve this.

We'll define `portVar` struct that will implement the
[flag.Value](https://golang.org/pkg/flag/#Value) interface. We'll also provide
a `PortVar` function to create such a variable.

Then we'll change our main to use `PortVar` instead of `IntVar`.

```go
func main() {
	flag.Var(PortVar(&config.port), "port", "port to listen on")
	// ...
}

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
have many sub-commands - `clone`, `add`, `diff` ...). We can do that with
[flag.FlagSet](https://golang.org/pkg/flag/#FlagSet).

```go
const (
	httpdUsage = `usage: %s httpd
Run HTTP server

Options:
`
	checkUsage = "usage: %s check URL\n"
)

func main() {
	flag.Usage = func() { // [1]
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s check|run\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(os.Args) < 2 { // [2]
		log.Fatalf("error: wrong number of arguments")
	}

	var err error
	switch os.Args[1] { // [3]
	case "run":
		err = runHTTPD()
	case "check":
		err = checkHTTPD()
	default:
		err = fmt.Errorf("error: unknown command - %s", os.Args[1])
	}

	if err != nil {
		log.Fatalf("error: %s", err)
	}

}

func checkHTTPD() error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError) // [4]
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), checkUsage, os.Args[0])
		fs.PrintDefaults()
	}

	if err := fs.Parse(os.Args[2:]); err != nil { // [5]
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("error: wrong number of arguments")
	}

	url := fs.Arg(0) // [6]
	resp, err := http.Get(url)
	switch {
	case err != nil:
		return err
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("error: bad status - %s", resp.Status)
	}

	return nil
}

func runHTTPD() error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.Var(PortVar(&config.port), "port", "port to listen on")
	fs.StringVar(&config.host, "host", config.host, "host to listen on")
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), httpdUsage, os.Args[0])
		fs.PrintDefaults()
	}

	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}

	http.HandleFunc("/", handler)
	addr := fmt.Sprintf("%s:%d", config.host, config.port)
	fmt.Printf("server ready on %s\n", addr)
	return http.ListenAndServe(addr, nil)
}

```

1. Usage for the main executable
2. We should have at lest two `os.Args` - the executable and the sub command name
3. `os.Args[1]` is the subcommand (`os.Args[1]` is the executable name)
4. Create a new `FlagSet` to parse the command line for this sub command. Use
   `flag.ContinueOnError` so parse error will not exit the program. The only function
   that should exit the program is `main`, all others should return an error
5. Pass rest of arguments. e.g. `["app", "check", "http://localhost:8080"]` →
   `["localhost:8081]`
6. [fs.Arg](https://golang.org/pkg/flag/#FlagSet.Arg) returns the nth command
   line argument (not including the program name) after parsing



# Conclusion

The `flag` package is flexible and will probably support all of your command
line parsing needs. It might be more verbose than other packages but it's in
the standard library so you don't need any extra dependencies and can count on
its API keeping the [Go compatibility
promise](https://golang.org/doc/go1compat).

You can see the full source code for the examples
[here](https://github.com/gopheracademy/gopheracademy-web/blob/master/content/advent-2019/flag).


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
