+++
author = ["Dave Cheney"]
date = "2014-12-08T08:00:00+00:00"
title = "Nigel's WebDAV package"
series = ["Advent 2014"]
+++

Nigel Tao and Nick Cooper have been working on a new WebDAV package for the [golang.org/x/net](http://godoc.org/golang.org/x/net) repository. The package is still in its formative stages, so this isn't a review of the package itself. 

Instead what I want to discuss is the design of one of the package's types, and how it made me re-evaluate some of my ideas about Go package design.

The Handler type
----------------

The central type in the WebDAV package is the [`Handler`](http://godoc.org/golang.org/x/net/webdav#Handler), which I've reproduced below

	type Handler struct {
		// FileSystem is the virtual file system.
		FileSystem FileSystem
		// LockSystem is the lock management system.
		LockSystem LockSystem
		// PropSystem is an optional property management system. If non-nil, TODO.
		PropSystem PropSystem
		// Logger is an optional error logger. If non-nil, it will be called
		// whenever handling a http.Request results in an error.
		Logger func(*http.Request, error)
	}

The `Handler` type has a [`ServeHTTP`](http://godoc.org/golang.org/x/net/webdav#Handler.ServeHTTP) method so it implements the [`http.Handler`](http://godoc.org/net/http#Handler) interface. Let's look at the start of `ServeHTTP` in more detail 

	func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
		status, err := http.StatusBadRequest, error(nil)
		if h.FileSystem == nil {
			status, err = http.StatusInternalServerError, errNoFileSystem
		} else if h.LockSystem == nil {
			status, err = http.StatusInternalServerError, errNoLockSystem
		} else {
			// preconditions succeeded, switch on method and handle request.
		}               
	}

As you know, I'm passionate about the ideas of [initialisation and configuration](http://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis), so my first response when I read the code review was 

>"`h.FileSystem` and `h.LockSystem` are mandatory, the `Handler` can't operate without them, so the package shouldn't permit the caller to create an invalid `Handler`"

Fortunately, I stopped myself before pressing send. To explain why, I want to walk through the ways that the WebDAV package could be changed to enforce this precondition.

A constructor you say
---------------------

Go doesn't have constructors, or destructors for that matter, but we do have the convention of a `New` function who's job is to return a properly initialised value. Here is a simplified version of what I thought a `webdav.NewHandler` function would look like

	type Handler struct {
		fs FileSystem // now private
		ls LockSystem // now private
		// more fields
     }

	func NewHandler(fs FileSystem, ls LockSystem) (*Handler, error) {
		if fs == nil {
			return nil, errNoFileSystem
		}
		if ls == nil {
			return nil, errNoLockSystem
		}
		return &Handler{fs: fs, ls: ls}, nil
    }

By making `Handler`'s fields private, callers from outside the package cannot access or assign them. 

This is a perfectly valid approach, especially when you need more complicated initialisation, usually involving starting some goroutines. However it does not achieve the goal we set; preventing callers from creating invalid `Handler` values. Consider this incorrect usage

    http.Handle("/dav", new(webdav.Handler)) // oops

The only way to prevent this misuse would be to make the `webdav.Handler` type private, forcing callers to go through `NewHandler`, and requiring `NewHandler` to return a value of an unexported type -- a highly questionable practice.
Let's try interfaces
--------------------

Another approach may be to have our `NewHandler` function return an interface value. Something like this

	// NewHandler returns a Handler that implements the WebDAV protocol.
	func NewHandler(fs FileSystem, ls LockSystem) (http.Handler, error) {
        if fs == nil {
			return nil, errNoFileSystem
	    }
	    if ls == nil {
			return nil, errNoLockSystem
	    }
	    return &Handler{FileSystem: fs, LockSystem: ls}, nil
	}

In this example we are very lucky that there is a pre made interface type, `http.Handler` that we can use, but in return we now have to document that the type of the interface is a `webdav.Handler`.

This solution is also not without its problems. `Handler` is still public, along with its fields, and to ensure that callers could not usurp `NewHandler`, `Handler` would again have to be made private, making it less discoverable and hiding the documentation attached to that type.

More fundamentally, applying this convention broadly implies that every type would also need an interface declared for the explicit purpose of hiding its fields, _not its implementation_ -- this is exactly the sort of ceremony that Go can do without.

A question of performance
-------------------------

Some readers may be wondering if there is a performance cost doing those two nil checks for every request. Isn't this `Handler.ServeHTTP` the "inner loop" for this package ?

This package talks to clients over the network so its performance will be dominated by network transmission. Compared to that `nil` checks are cheap, along with slice bounds checks and interface dynamic method dispatch. Also, we know that the processor can predict that these `nil` checks will never be true, because if they were, the server would not be able to serve any traffic.

Simpler is better
-----------------

In this post I talked about two common approaches to ensuring that types are properly initialised, but in the end I'm glad I didn't suggest either to Nigel and Nick. I realised that their approach was better, and not just better, but simpler and more idiomatic.

Using the `webdav` package is beautifully simple and shows the elegance of composite literals

    handler := webdav.Handler{
        FileSystem: webdav.Dir{"/somepath"},
        LockSystem: new(webdav.MemLS), // this type is still in code review
    }
    http.Handle("/dav", &handler)

What Nigel and Nick have done is deferred checking the `handler` value is properly initialised until the point you care about it, which in this case is the only public entry point into this type, its `ServeHTTP` method.

I see this pattern of deferred error checking evolving in Go code as we learn more about the language. A great example of this is evolving style is the `bufio.Scanner` type.

	// ReadLines reads all lines from r.
	func ReadLines(r io.Reader) ([]string, error) {
		var lines []string
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		return lines, scanner.Err()
	}

In this concocted example, `scanner` reads lines of text from `r` and appends them to the result. At some point `scanner.Scan()` will return false and the loop will exit, and it is at that point that we need to check if an error occurred. Comparing this example to one that uses `bufio.Readline` is left as an exercise to the reader. 

My take away from this experience: it is easy to dogmatically apply ideas like construction from other languages, but this episode taught me that this should not be my default especially when simpler options are available.

Lastly, if you're passionate about WebDAV, please get involved with the design on this package; many hands make light work.
