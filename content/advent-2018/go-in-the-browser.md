+++
author = ["Agniva De Sarker"]
title = "Go and WebAssembly: running Go programs in your browser"
linktitle = "Go in the browser"
date = 2018-12-07T00:00:00Z
series = ["Advent 2018"]
+++

For a long time, Javascript was the lingua franca amongst web developers. If you wanted to write a stable, mature webapp, writing in javascript was pretty much the only way to go.

WebAssembly is going to that change soon. Using WebAssembly (also called wasm), one can write webapps in _any_ language now. In this post, we will see how to write Go programs and run them in the browser using wasm.

## But first, what is WebAssembly

The [webassembly.org](https://webassembly.org/) site defines it as "WebAssembly is a binary instruction format for a stack-based virtual machine". That is a great definition, but let us break it down to something we can easily understand.

Essentially, wasm is a binary format; just like ELF, Mach, and PE. The only difference is that it is for a virtual compilation target, not an actual physical machine. Why virtual? because unlike C/C++ binaries, wasm binaries are not targeted for a specific platform. So you can use the same binary in Linux, Windows and Mac without changing anything. As a result, we need another "agent" which translates the wasm instructions inside the binary into platform specific instructions and actually run them. This "agent" is actually the browser. But, in theory it can just as well be anything else.

This gives a common compilation target for us to build webapps using any programming language of our choice ! We don't need to worry about the target platform, as long as we compile to the wasm format. Exactly like we write a webapp, but now we have the advantage of writing the in whatever language we choose.

## Hello WASM

Let us start with a simple hello world program to get a taste of things. Ensure that your Go version is atleast 1.11. We can write something like this -

```go
package main

import (
	"fmt"
)

func main() {
	fmt.Println("hello wasm")
}
```

Save this in a file `test.go`. This just looks like a regular Go program. Now let us compile this to target the wasm platform. We need to set the `GOOS` and `GOARCH` for that.

`$GOOS=js GOARCH=wasm go build -o test.wasm test.go`

So now we have the wasm binary generated. But unlike in native systems, we need to run this inside the browser. For this, we need to throw in a few more things to accomplish this.

- A webserver which will serve our webapp.
- An index.html file which contains some js glue code needed to load the wasm binary.
- And a js file which serves as the communication interface between the browser and our wasm binary.

I like to think of it just like the things required to make The PowerPuff Girls.

![wasmrequirements](/postimages/advent-2018/go-in-the-browser/powerpuff.jpg)

And **BOOM**, we have a WebAssembly application !

We already have the html and the js file available in our Go distribution, so we will just copy them over.

```
$cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" .
$cp "$(go env GOROOT)/misc/wasm/wasm_exec.html" .
$# we rename the html file to index.html for convenience.
$mv wasm_exec.html index.html
$ls -l
total 8960
-rw-r--r-- 1 agniva agniva    1258 Dec  6 12:16 index.html
-rwxrwxr-x 1 agniva agniva 6721905 Sep 24 12:28 serve
-rw-rw-r-- 1 agniva agniva      76 Dec  6 12:08 test.go
-rwxrwxr-x 1 agniva agniva 2425246 Dec  6 12:09 test.wasm
-rw-r--r-- 1 agniva agniva   11905 Dec  6 12:16 wasm_exec.js
```

`serve` is a simple Go binary to serve files in the current directory. But just about any web server will do.

Once we run this, and open our browser. We see a `Run` button, clicking which, will execute our application. Then we click it and check the console -

![hellowasm](/postimages/advent-2018/go-in-the-browser/hellowasm.png)

Neat ! We just wrote a program in Go and ran it inside the browser.

So far so good. But this was a simple hello world program. A real world webapp needs to interact with the DOM. We need to respond to button click events, take input data from text boxes, and send data back to the DOM. Now we will build a minimal image editor in the browser which will use all of these capabilities.

## DOM API

But first, to interact with the browser from Go code, we need a DOM API. We have the `syscall/js` library to help us out with that. It is a very basic, but nevertheless powerful form of a DOM API, on top of which we can build our app. Let us very quickly see some of its capabilities before we move over to making our app.

**Callbacks**

To respond to DOM events, we declare callbacks and hook them up with events like this -

```
import “syscall/js”

// Declare callback
cb := js.NewEventCallback(js.PreventDefault, func(ev js.Value) {
	// handle event
})


// Hook it up with a DOM event
js.Global().Get("document").
	Call("getElementById", "myBtn").
	Call("addEventListener", "click", cb)


// Call cb.Release() on your way out.
```

**Updating the DOM**

To update the DOM from inside Go, we can do-

```
import “syscall/js”

js.Global().Get("document").
		Call("getElementById", "myTextBox").
		Set("value", "hello wasm")
```

You can even call JS functions and manipulate native native JS Objects like `FileReader` or `Canvas`. Feel free to check out the [syscall/js](https://golang.org/pkg/syscall/js/) documentation for further details.

Ok, now on with building our app !

## A proper webapp

We will build a small app which will take an input image, then perform some manipulations on the image like brightness, contrast, hue, saturation, and finally send the output image back to the browser. There will be sliders for each of these effects, which the user can change and see the target image change in real time.

First, we need to get the input image from the browser to our Go code, so that we can work on it. To efficiently do this, we need to resort to some `unsafe` tricks, the details of which I will skip here. Once we have the image, it is fully in our control and we are free to do whatever with it. Below is a brief snippet from the image loader callback -

```go
s.onImgLoadCb = js.NewCallback(func(args []js.Value) {
	reader := bytes.NewReader(s.inBuf)
	var err error
	s.sourceImg, _, err = image.Decode(reader)
	if err != nil {
		s.log(err.Error())
		return
	}
	// Now the sourceImg is an image.Image with which we are free to do anything !
})
```

Then we take user values from any of the effect sliders, and manipulate the image. We use the awesome [bild](https://github.com/anthonynsimon/bild) library for that. Here is a small snippet of the contrast callback -

```go
s.contrastCb = js.NewEventCallback(js.PreventDefault, func(ev js.Value) {
	delta := ev.Get("target").Get("valueAsNumber").Float()
	res := adjust.Contrast(s.sourceImg, delta)
})
```

After this, we encode the target image to jpeg and send it back to the browser. Here is the full app in action -

We load the image:

![initial](/postimages/advent-2018/go-in-the-browser/initial.png)

Change contrast:

![contrast](/postimages/advent-2018/go-in-the-browser/contrast.png)

Change hue:

![hue](/postimages/advent-2018/go-in-the-browser/hue.png)

Awesome, we are able to natively manipulate images in the browser without writing a single line of Javascript ! The source code can be found [here](https://github.com/agnivade/shimmer).

Note that all this is being done natively in the browser itself. There are no Flash plugins, Java Applets or Silverlight magic happening here. WebAssembly is supported natively in the browser out of the box.

## Final words

Some of my closing remarks:

- Since Go is a garbage collected language, the entire runtime is shipped inside the wasm binary. Hence it is common for binaries to have large sizes in the order of MBs. This is still a sore point compared to other languages like C/Rust; because shipping MBs of data to the browser is not ideal. However, if the wasm spec supports GC by itself, then this can change.
- Wasm support in Go is officially experimental. The `syscall/js` API itself is in flux and might change in future. If you see a bug, please feel free to file an issue at our [issue tracker](https://github.com/golang/go/issues).
- Like all technologies, WebAssembly is not a silver bullet. Sometimes, simple JS is faster and easier to write. However, the wasm spec itself is very much in development, and there are more features coming soon. Thread support is one such feature.

Hopefully, this post showed some of the cool aspects of WebAssembly and how you can write a fully-functioning webapp using Go. Do try it out, and file issues if you see a bug. If you need any help, feel free to drop in to the [#webassembly](https://gophers.slack.com/) channel.
