+++
author = ["Sebastien Binet"]
date = "2017-12-17"
linktitle = "Go and wasm"
series = ["Advent 2017"]
title = "Go and wasm: generating and executing wasm with Go"
+++

Today we will see how we can interact with WebAssembly, from Go: how to execute WebAssembly bytecode from Go and how to generate WebAssembly bytecode with Go.
But first of all: what is WebAssembly?

## WebAssembly

According to [webassembly.org](http://webassembly.org), WebAssembly (`wasm` for short) is a new portable, size- and load-time-efficient format suitable for compilation to the web.
In a way, wasm is the next evolution of asm.js and PNaCl: it's a new way to run code, on the web.
But faster.

WebAssembly is currently being designed as an open standard by a W3C Community Group that includes representatives from all major browsers.
As of late 2017, WebAssembly is supported by all major web browsers (Chrome, Edge, Firefox and Safari.)

With wasm, developers get:

- a compact, binary format to send over the wire and incorporate into their project,
- a near-native performance boost,
- performant and safe code executed inside the browser sand box,
- another language (besides JavaScript) to develop their projects: one can target wasm from C/C++, Rust, ... (and, eventually, Go.)

### wasm format

WebAssembly is formally defined by a set of [specifications](https://webassembly.github.io/spec/index.html).
The wasm binary format is described [here](https://webassembly.github.io/spec/binary/index.html).

A `.wasm` file is the result of the compilation of C/C++, Rust or Go code with the adequate toolchain.
It contains bytecode instructions to be executed by the browser or any other program that can decode and interpret that binary format.

A `.wasm` file contains a wasm module.
Every wasm module starts with a magic number `\0asm` (_ie:_ `[]byte{0x00, 0x61, 0x73, 0x6d}`) and then a version number (`0x1` at the moment.)
After that come the different sections of a module:

- the types section: function signature declarations,
- the imports section: imports declarations,
- the functions section: function declarations,
- the tables section: indirect function table,
- the memories section,
- the globals section,
- the exports section,
- the start function section: the `func main()` equivalent,
- the code segments section: function bodies, and
- the data segments section.

The full description with all the details is available [here](https://webassembly.github.io/spec/binary/modules.html).
A more gentle and less dry introduction to wasm can be found at https://rsms.me/wasm-intro.

A toolchain like [emscripten](http://kripken.github.io/emscripten-site/) will thus take a set of C/C++ source code and generate a `.wasm` file containing type definitions, function definitions, function bodies with the corresponding wasm instructions (_e.g.:_ `i32.store` to store a signed 32b integer to memory, `if` and `return` control instructions, etc...)

We will see how to generate `.wasm` files in a bit but let us first play with it.
Consider this [`basic.wasm`](https://github.com/go-interpreter/wagon/raw/master/exec/testdata/basic.wasm):

```
$> curl -O -L https://github.com/go-interpreter/wagon/raw/master/exec/testdata/basic.wasm
$> ls -lh ./basic.wasm
-rw-r--r-- 1 binet binet 38 Dec 12 17:01 basic.wasm

$> file ./basic.wasm 
./basic.wasm: WebAssembly (wasm) binary module version 0x1 (MVP)

$> hexdump -C ./basic.wasm 
00000000  00 61 73 6d 01 00 00 00  01 05 01 60 00 01 7f 03  |.asm.......`....|
00000010  02 01 00 07 08 01 04 6d  61 69 6e 00 00 0a 07 01  |.......main.....|
00000020  05 00 41 2a 0f 0b                                 |..A*..|
00000026
```

So all the tools at our disposal agree: it is indeed a wasm file.
But couldn't we do something to extract some more informations about that file?

Like for object files, there is an `objdump`-like command that allows to inspect the contents of a binary wasm file.
The [wabt](https://github.com/WebAssembly/wabt) project provides a `wasm-objdump` command, written in C/C++.
But as this is a Go advent post, we will instead use the one provided by [wagon](https://github.com/go-interpreter/wagon):

```
$> go get github.com/go-interpreter/wagon/cmd/wasm-dump
$> wasm-dump -help
Usage: wasm-dump [options] file1.wasm [file2.wasm [...]]

ex:
 $> wasm-dump -h ./file1.wasm

options:
  -d	disassemble function bodies
  -h	print headers
  -s	print raw section contents
  -v	enable/disable verbose mode
  -x	show section details
```

Running it with `basic.wasm` as argument gives:

```
$> wasm-dump -h ./basic.wasm
./basic.wasm: module version: 0x1

sections:

     type start=0x0000000a end=0x0000000f (size=0x00000005) count: 1
 function start=0x00000011 end=0x00000013 (size=0x00000002) count: 1
   export start=0x00000015 end=0x0000001d (size=0x00000008) count: 1
     code start=0x0000001f end=0x00000026 (size=0x00000007) count: 1
```

This `basic.wasm` file has 4 sections.

Let's dig deeper:

```
$> wasm-dump -x ./basic.wasm 
./basic.wasm: module version: 0x1

section details:

type:
 - type[0] <func [] -> [i32]>
function:
 - func[0] sig=0
export:
 - function[0] -> "main"
```

This wasm module exports a function `"main"`, which takes no argument and returns an `int32`.

The content of these sections is the following:

```
$> wasm-dump -s ./basic.wasm 
./basic.wasm: module version: 0x1

contents of section type:
0000000a  01 60 00 01 7f                                    |.`...|

contents of section function:
00000011  01 00                                             |..|

contents of section export:
00000015  01 04 6d 61 69 6e 00 00                           |..main..|

contents of section code:
0000001f  01 05 00 41 2a 0f 0b                              |...A*..|
```

If you read wasm speak fluently you won't be surprised by the content of the next snippet, showing the disassembly of the `func[0]`:

```
$> wasm-dump -d ./basic.wasm
./basic.wasm: module version: 0x1

code disassembly:

func[0]: <func [] -> [i32]>
 000000: 41 2a 00 00 00             | i32.const 42
 000006: 0f                         | return
 000008: 0b                         | end
```

It puts the `int32` constant `42` on the stack and returns it to the caller.
It's just the wasm equivalent of:

```go
func f0() int32 {
	return 42
}
```

Can we test this?

## Executing wasm with wagon

`wagon` actually exposes a very limited non-interactive (yet!) interpreter of wasm: `wasm-run`.

```
$> go get github.com/go-interpreter/wagon/cmd/wasm-run
$> wasm-run -h
Usage of wasm-run:
  -v	enable/disable verbose mode
  -verify-module
    	run module verification
```

Let's try it on our `basic.wasm` file:

```
$> wasm-run ./basic.wasm
main() i32 => 42 (uint32)
```

*Victory !*

[wasm-run](https://github.com/go-interpreter/wagon/blob/master/cmd/wasm-run/main.go) is a rather simple and limited (yet!) wasm embedder:

- it reads the provided wasm file,
- it (optionally) verifies the wasm module,
- it creates a VM with `go-interpreter/wagon/exec`, that VM will execute the `start` section of the module (if any)
- it runs all the exported functions that take no input parameters

and _voila!_

Ok, but wasn't wasm designed for the web and its browsers?

## Executing wasm in the browser

Switching gears, let us write a little web server that will serve a simple wasm module:

```go
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":5555", "server address:port")
	flag.Parse()
	http.HandleFunc("/", rootHandle)
	http.HandleFunc("/wasm", wasmHandle)

	log.Printf("listening on %q...", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func rootHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, page, html.EscapeString(hex.Dump(wasmAdd)))
}

func wasmHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/wasm")
	n, err := w.Write(wasmAdd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if n != len(wasmAdd) {
		http.Error(w, io.ErrShortWrite.Error(), http.StatusServiceUnavailable)
	}
}

var wasmAdd = []byte{
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
	0x01, 0x07, 0x01, 0x60, 0x02, 0x7f, 0x7f, 0x01,
	0x7f, 0x03, 0x02, 0x01, 0x00, 0x07, 0x07, 0x01,
	0x03, 0x61, 0x64, 0x64, 0x00, 0x00, 0x0a, 0x09,
	0x01, 0x07, 0x00, 0x20, 0x00, 0x20, 0x01, 0x6a,
	0x0b,
}

const page = `
<html>
	<head>
		<title>Testing WebAssembly</title>
		<script type="text/javascript">

		function fetchAndInstantiate(url, importObject) {
			return fetch(url).then(response =>
				response.arrayBuffer()
			).then(bytes =>
				WebAssembly.instantiate(bytes, importObject)
			).then(results =>
			    results.instance
			);
		}

		var mod = fetchAndInstantiate("/wasm", {});

		window.onload = function() {
			mod.then(function(instance) {
				var div = document.getElementById("wasm-result");
				div.innerHTML = "<code>add(1, 2)= " + instance.exports.add(1, 2) + "</code>";
			});
		};

		</script>
	</head>

	<body>
		<h2>WebAssembly content</h2>
		<div id="wasm-content">
			<pre>%s</pre>
		</div>

		<h2>WebAssembly</h2>
		<div id="wasm-result"><code>add(1, 2)= N/A</code></div>
	</body>
</html>
`
```

Running this in a terminal:

```
$> go run ./main.go 
2017/12/14 12:45:21 listening on ":5555"...
```

and then navigating to that location, you should be presented with:

```
WebAssembly content

00000000  00 61 73 6d 01 00 00 00  01 07 01 60 02 7f 7f 01  |.asm.......`....|
00000010  7f 03 02 01 00 07 07 01  03 61 64 64 00 00 0a 09  |.........add....|
00000020  01 07 00 20 00 20 01 6a  0b                       |... . .j.|

WebAssembly
add(1, 2)= 3
```

_Victory!_ again!

For more informations about the JavaScript API that deals with WebAssembly, there are these useful references:

- https://developer.mozilla.org/en-US/docs/WebAssembly/Using_the_JavaScript_API
- https://developer.mozilla.org/en-US/docs/WebAssembly/Loading_and_running

But, up to now, we have only been able to inspect and execute already existing wasm files.
How do we create these files?

## Generating wasm

We briefly mentioned at the beginning of this post that wasm files could be generated from C/C++ (using emscripten) or from Rust (using cargo or rustup).
Instructions related to these tasks are available here:

- https://developer.mozilla.org/en-US/docs/WebAssembly/C_to_wasm
- https://github.com/raphamorim/wasm-and-rust

Compiling Go code to wasm is also doable, but the support for this backend hasn't been yet integrated into `gc`.
An issue is tracking the progress of this feature: https://github.com/golang/go/issues/18892.
As that discussion is quite long, here is the executive summary: a development branch with preliminary support for wasm has been created by [@neelance (Richard Musiol)](https://github.com/neelance) (yeah!).

Here are the instructions to compile a `gc` toolchain with a `GOOS=js GOARCH=wasm` environment:

```
$> cd somewhere
$> git clone https://go.googlesource.com/go
$> cd go
$> git remote add neelance https://github.com/neelance/go
$> git fetch --all
$> git checkout wasm-wip
$> cd src
$> ./make.bash
$> cd ../misc/wasm
```

The `misc/wasm` directory contains all the files (save the actual `wasm` module) to execute a `wasm` module with `nodejs`.

Let us compile the following `main.go` file:

```
package main

func main() {
	println("Hello World, from wasm+Go")
}
```

with our new wasm-capable `go` binary:

```
$> GOARCH=wasm GOOS=js go build -o test.wasm main.go
$> ll
total 4.0K
-rw-r--r-- 1 binet binet   68 Dec 14 14:30 main.go
-rwxr-xr-x 1 binet binet 947K Dec 14 14:30 test.wasm
```

Copy over the `misc/wasm` files under this directory, and then *finally*, run the following `server.go` file:

```
package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":5555", "server address:port")
	flag.Parse()
	srv := http.FileServer(http.Dir("."))
	log.Printf("listening on %q...", *addr)
	log.Fatal(http.ListenAndServe(*addr, srv))
}
```

Like so:

```
$> ll
total 968K
-rw-r--r-- 1 binet binet   68 Dec 14 14:30 main.go
-rw-r--r-- 1 binet binet  268 Dec 14 14:38 server.go
-rwxr-xr-x 1 binet binet 947K Dec 14 14:30 test.wasm
-rw-r--r-- 1 binet binet  482 Dec 14 14:32 wasm_exec.html
-rwxr-xr-x 1 binet binet 7.9K Dec 14 14:32 wasm_exec.js

$> go run ./server.go
2017/12/14 14:39:18 listening on ":5555"...

```

Navigating to `localhost:5555/wasm_exec.html` will present you with a `[Run]` button that, when clicked should display `"Hello World, from wasm+Go"` in the console.

We've just had our browser run a `wasm` module, generated with our favorite compilation toolchain!

## Conclusions

In this blog post, we have:

- learned about some of the internals of the `wasm` binary format,
- inspected `wasm` files,
- interpreted a `wasm` file,
- served a `wasm` file via `net/http`, and
- compiled a `wasm` module with a modified `go` toolchain.

I hope this has inspired some of you to try this at home.

WebAssembly is poised to take the Web by storm, bring "native" performances to the Web platform and allow developers to use other languages besides JavaScript to build Web applications.

But even if WebAssembly has been designed for the Web, nothing prevents it from being used outside of the Web.
Indeed, the `wasm` binary format and the bytecode that it contains can very well become a very popular (and effective) Intermediate Representation format, like LLVM's IR.
For example, the [go-interpreter/wagon](https://github.com/go-interpreter/wagon) project could build (_Help wanted_) a complete interpreter in Go, generating wasm bytecode from Go source code, and then executing that wasm bytecode.
_"Building an interpreter in Go, for Go"_... what's not to love!?
This could even be used as a backend for [Neugram](https://neugram.io), but that's another story...
