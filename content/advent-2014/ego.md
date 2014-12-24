+++
author = ["Ben Johnson"]
date = "2014-12-24T00:00:00-08:00"
title = "Type safe templating with ego"
series = ["Advent 2014"]
+++

Go has seen many converts from scripting languages such as Ruby and Python.
These newcomers love the type safety, the language simplicity, and the static
binaries that Go offers. Unfortunately, these features are lost once
developers dive into the built-in templating libraries:
[`text/template`][text-template] & [`html/template`][html-template].

We'll look at [ego][] -- a type safe templating language -- that preserves all
these features and lets you write your templates in your favorite language: Go!

[text-template]: http://golang.org/pkg/text/template/
[html-template]: http://golang.org/pkg/html/template/
[ego]: https://github.com/benbjohnson/ego


# Installing ego

The ego templating language is a port of Ruby's [ERb][] so much of the syntax
is similar. Ego works by generating Go code from your templates so let's
start by installing the command line tool:

```
$ go get github.com/benbjohnson/ego/...
```

This creates an `ego` binary in your `$GOPATH/bin`.

[ERb]: http://ruby-doc.org/stdlib-2.1.5/libdoc/erb/rdoc/ERB.html


# Intro to ego

To start, let's create a simple web site that prints out an HTML page of
widgets. You can find the full code for this [sample application here][ego-example].

Here is our simple `Widget` type:

```
type Widget struct {
    Name    string
    Price   int
}
```

We'll create a template called `index.ego` in our project directory and start
by declaring the template's function signature:

```
<%! func RenderIndex(w io.Writer, widgets []*Widget) error %>
```

This is a declaration block and it goes at the top of every ego file. It defines
the function that will be declared for this template. It must take an
`io.Writer` and return an error. We can also add our own arguments such as a
slice of widgets.

Next we'll write our HTML as we'd expect. Anywhere we want to use Go code we
simply wrap it in `<% ... %>` tags. Any place we want to output Go variables
in our template we wrap those variables in `<%= ... %>` tags:

```
<%! func RenderIndex(w io.Writer, widgets []*Widget) error %>

<html>
<body>
  <h1>Widgets for Sale!</h1>

  <ul>
    <% for _, widget := range widgets { %>
      <li><%= widget.Name %> for $<%= widget.Price %></li>
    <% } %>
  </ul>
</body>
</html>
```

In our template we are looping over the `widgets` argument and outputting a
list item (`<li>`) for each one.

[ego-example]: https://github.com/benbjohnson/ego-example

# Wiring up our web application

We can compile our ego template into an `ego.go` file by running:

```
$ ego .
```

To use our template let's create a simple HTTP application:

```
import "net/http"

func main() {
	http.HandleFunc("/", HandleIndex)
	http.ListenAndServe(":10000", nil)
}

// HandleIndex renders the home page using our RenderIndex ego template.
func HandleIndex(w http.ResponseWriter, r *http.Request) {
	// Our list of available widgets.
	widgets := []*Widget{
		{Name: "Blue Widget", Price: 100},
		{Name: "Red Widget", Price: 20},
	}

	// Generate the template and write it to the response body.
	RenderIndex(w, widgets)
}
```

If we build and run our application and visit the home page we'll see:

```
<html>
<body>
  <h1>Widgets for Sale!</h1>

  <ul>
    <li>Blue Widget for $100</li>
    <li>Red Widget for $20</li>
  </ul>
</body>
</html>
```

# Using Go inside ego

So how does ego work? It's suprisingly simple. The `ego` tool parses the
templates and creates a function that outputs any text outside of a `<% %>`
tag as plain text using `fmt.Fprint(w, ...)`. Any text that is inside the `<% %>` tags
is output directly into the template function. That means you can use the full
power of Go inside your templates.


The ego tool also outputs preprocessor line directives so Go's error reporting
can reference your original source. Line directives are an undocumented feature
in Go. They allow you to put a simple comment in to tell Go where the original
source came from. We can see this in our generated function:

```
//line index.ego:1
func RenderIndex(w io.Writer, widgets []*Widget) error {
//line index.ego:2
    _, _ = fmt.Fprintf(w, "\n\n<html>\n<body>\n  <h1>Widgets for Sale!</h1>\n\n  <ul>\n    ")
//line index.ego:8
    for _, widget := range widgets {
//line index.ego:9
        _, _ = fmt.Fprintf(w, "\n      <li>")
//line index.ego:9
        _, _ = fmt.Fprintf(w, "%v", widget.Name)
//line index.ego:9
        _, _ = fmt.Fprintf(w, " for $")
//line index.ego:9
        _, _ = fmt.Fprintf(w, "%v", widget.Price)
//line index.ego:9
        _, _ = fmt.Fprintf(w, "</li>\n    ")
//line index.ego:10
    }
//line index.ego:11
    _, _ = fmt.Fprintf(w, "\n  </ul>\n</body>\n</html>\n")
    return nil
}
```

If we renamed our `Name` field on `Widget` to `Description` and rebuilt then
we'd see a standard Go build error with a line number referencing our original
template:

```
$ go build .
# github.com/benbjohnson/ego-example
index.ego:9[ego.go:17]: widget.Name undefined (type *Widget has no field or method Name)
```

This type safety is incredibly useful as a project grows.


# Ego is Go

Having ego generate pure `.go` files keeps your workflow consistent. Instead of
using an asset compilation tool to bundle text templates into your final binary,
ego templates will be automatically included in your binary.

Pure Go is also fast. There is no parsing done at runtime and templates are
written directly to the `io.Writer`. You can also use Go functions in your
package (or call and nest other templates) because they're all plain Go code!


# Conclusion

There are many templating libraries available for Go but few provide the
tight integration of ego. With ego, you don't have to learn a new language --
it's just Go!

If you have any questions on getting started with ego, feel free to contact
me on Twitter at [@benbjohnson][twitter].

[twitter]: https://twitter.com/benbjohnson
