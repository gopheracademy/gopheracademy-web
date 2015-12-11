+++
author = ["Sarah Adams"]
date = "2015-12-14T22:04:31-08:00"
series = ["Advent 2015"]
title = "test2doc: Generate Your API Docs"

+++

API docs are important. We all know that.
They are also painful and tedious to maintain.
When your docs aren't accurate, you get more questions. And everyone 
loses time.

I've always thought of unit tests as a great source of documentation. 
But the non-Go engineers consuming my API don't often agree.

Yet most of the same information is there:

- request format:
    - HTTP method
    - URI
    - query params
    - request headers
    - request body
- response format:
    - status code
    - response headers
    - response body

<br>
All that's missing from this list is the 
**high-level descriptions**, the **context**.

Enter *Go doc*.<br>
We can find descriptions of an HTTP handler in its Go doc string, eg:
```go
// GetWidget retrieves a single Widget
func GetWidget(w http.ResponseWriter, req *http.Request) {
    // ...
}
```
<br>
And thus began **test2doc** - automatically generate complete API 
documentation from your existing Go unit tests + Go doc strings.


## Example

Given an HTTP handler func, eg.:

```go
// GetWidget retrieves a single Widget
func GetWidget(w http.ResponseWriter, req *http.Request) {
    // ...
}
```

And a test for this handler func, eg.:

```go
func TestGetWidget(t *testing.T) {
    urlPath := fmt.Sprintf("/widgets/%d", 2)

    resp, err := http.Get(server.URL + urlPath)
    // assert all the things...
}
```

Test2doc will automatically generate markdown documentation for this 
endpoint in the [API Blueprint](https://github.com/apiaryio/api-blueprint/blob/master/API%20Blueprint%20Specification.md) 
format as your tests run, like so:

```
# Group widgets

## /widgets/{id}

+ Parameters
    + id: `2` (number)

### Get Widget [GET]
retrieves a single Widget

+ Response 200 

    + Body

            {
                "Id": 2,
                "Name": "Pencil",
                "Role": "Utensil"
            }        
```

Which you can then parse and host w/ 
[Apiary.io](http://docs.testingit.apiary.io/#): 
![screenshot](/postimages/advent-2015/test2doc-widgets-api.jpg)

Or use a custom parser and host yourself.

## Getting started with test2doc
A big goal of mine as I was writing this project was to limit the 
amount of additional code needed to get test2doc up and running.

Requirements:

1. You must have a `TestMain` for each of the packages you wish to 
document.
2. All of the tests in your package must share a single `test.Server` 
instance (from [test2doc/test](https://godoc.org/github.com/adams-sarah/test2doc/test) package)


### 3 Code Additions

3 additions, and only to your testing code:

```go
package widgets_test

import (
	"testing"

	"github.com/adams-sarah/test2doc/test"
)

var server *test.Server

func TestMain(m *testing.M) {
	// 1. Tell test2doc how to get URL vars out of your HTTP requests
	//
	//    The 'URLVarExtractor' function must have the following signature:
	//      func(req *http.Request) map[string]string
	//      where the returned map is of the form map[key]value
	test.RegisterURLVarExtractor(myURLVarExtractorFn)


	// 2. You must use test2doc/test's wrapped httptest.Server instead of
	//    the raw httptest.Server, so that test2doc can listen to and
	//    record requests & responses.
	//
	//    NewServer takes your HTTP handler as an argument
	server, err := test.NewServer(router)
	if err != nil {
		panic(err.Error())
	}

	// .. then run your tests as usual
	exitCode := m.Run()


	// 3. Finally, you must tell the wrapped server when you are done testing
	//    so that the buffer can be flushed to an API Blueprint doc file
	server.Finish()

	// note that os.Exit does not respect defers.
	os.Exit(exitCode)
}

```

Some example `URLVarExtractor`s for different routers:

`gorilla/mux`:

```go
// NOTE: if you are using gorilla/mux, you must set the 
// router's 'KeepContext' to true, so that url parameters 
// can be accessed after the request has been handled.
router.KeepContext = true
    
// Use mux.Vars func as URLVarExtractor
test.RegisterURLVarExtractor(mux.Vars)
```
<br>

`julienschmidt/httprouter`:

```go
import (
	"net/http"

	"github.com/adams-sarah/test2doc/doc/parse"
	"github.com/julienschmidt/httprouter"
)

// MakeURLVarExtractor returns a func which extracts 
// url vars from a request for test2doc documentation generation
func MakeURLVarExtractor(router *httprouter.Router) parse.URLVarExtractor {
	return func(req *http.Request) map[string]string {
		// httprouter Lookup func needs a trailing slash on path
		path := req.URL.Path
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}

		_, params, ok := router.Lookup(req.Method, path)
		if !ok {
			return nil
		}

		paramsMap := make(map[string]string, len(params))
		for _, p := range params {
			paramsMap[p.Key] = p.Value
		}

		return paramsMap
	}
}

// and then..
test.RegisterURLVarExtractor(MakeURLVarExtractor(router))
```
<br>

## How the magic happens

### Recording Requests and Responses:
Recording the requests and responses was pretty straight-forward:

1. Write the HTTP handler's response initially to an `http.ResponseRecorder`
2. Add the response header/body to the documentation
3. Copy the response back to the original `http.ResponseWriter`
4. Continue executing the test


### Fetching the handler's Go doc string:
First, let's take a look at the `http.ResponseWriter` interface
from Go's `net/http` package:

```go
type ResponseWriter interface {
        Header() Header
        Write([]byte) (int, error)
        WriteHeader(int)
}
```


test2doc has its own `ResponseWriter`, which implements
`http.ResponseWriter`, and looks something like this:

```go
type ResponseWriter struct {
	HandlerInfo HandlerInfo
	URLVars     map[string]string
	W           *httptest.ResponseRecorder
}
```

test2doc's `ResponseWriter` implements the `Header` and 
`WriteHeader` methods by just falling back to those of its 
embedded `httptest.ResponseRecorder`:

```go
func (rw *ResponseWriter) Header() http.Header {
	return rw.W.Header()
}

func (rw *ResponseWriter) WriteHeader(c int) {
	rw.W.WriteHeader(c)
}
```

The magic is in the `Write` implementation. 
Since every HTTP handler will call the `Write` method at 
some point to write out the response, we hijack our 
`ResponseWriter`'s `Write` to inspect the call stack and find
our HTTP handler before performing the `Write`:

```go
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	rw.setHandlerInfo()
	return rw.W.Write(b)
}
```

Here, `setHandlerInfo` iterates up the call stack (using Go's `runtime` 
package) until it finds a caller whose declaration is inside the 
package we are testing (in all likelihood, the handler).

Once we have the handler's function name, we can get the Go doc string 
using the `go/doc` package.


## Thoughts for the future
I'd like to convert the main types to interfaces, allowing support for 
formats other than the API Blueprint format.

I'd also like to improve upon the "handler-finding" algorithm (above), 
to make it more reliably accurate.

## Contributions welcome!
Drop me a line at sadams.codes@gmail.com, or [adams-sarah](https://github.com/adams-sarah) on github.
